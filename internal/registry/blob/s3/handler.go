package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/util"
	"github.com/glasskube/pkg/seekbuf"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

const (
	chunksPrefix  = "chunks"
	maxS3PartSize = int64(5) * 1024 * 1024 * 1024 // S3 maximum per-part size is 5 GiB
	splitPartSize = int64(1) * 1024 * 1024 * 1024 // target size per UploadPart call
)

type blobHandler struct {
	s3Client        *s3.Client
	s3PresignClient *s3.PresignClient
	allowRedirect   bool
	bucket          string
}

var (
	_ blob.BlobHandler       = &blobHandler{}
	_ blob.BlobStatHandler   = &blobHandler{}
	_ blob.BlobPutHandler    = &blobHandler{}
	_ blob.BlobDeleteHandler = &blobHandler{}
)

func NewBlobHandler(ctx context.Context, s3Client *s3.Client) (blob.BlobHandler, error) {
	s3Config := env.RegistryS3Config()

	if s3Config.CreateBucket {
		if err := ensureBucketExists(ctx, s3Client, s3Config.Bucket, s3Config.Region); err != nil {
			return nil, err
		}
	}

	h := blobHandler{
		s3Client:      s3Client,
		allowRedirect: s3Config.AllowRedirect,
		bucket:        s3Config.Bucket,
	}

	if h.allowRedirect {
		h.s3PresignClient = s3.NewPresignClient(s3Client)
	}

	return &h, nil
}

func ensureBucketExists(ctx context.Context, client *s3.Client, bucket string, region string) error {
	log := internalctx.GetLogger(ctx).With(zap.String("bucket", bucket))
	log.Debug("initializing object store bucket")

	_, err := client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
		CreateBucketConfiguration: &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		},
	})
	if err != nil {
		if _, ok := errors.AsType[*s3types.BucketAlreadyOwnedByYou](err); ok {
			log.Debug("bucket already exists")
			return nil
		}

		return err
	}

	log.Info("bucket created")
	return nil
}

// Get implements blob.BlobHandler.
func (handler *blobHandler) Get(
	ctx context.Context,
	repo string,
	h digest.Digest,
	allowRedirect bool,
) (io.ReadCloser, error) {
	key := h.String()
	if handler.allowRedirect && allowRedirect {
		resp, err := handler.s3PresignClient.PresignGetObject(ctx,
			&s3.GetObjectInput{Bucket: util.PtrTo(handler.bucket), Key: &key})
		if err != nil {
			return nil, convertErrNotFound(err)
		} else {
			return nil, blob.RedirectError{
				Code:     http.StatusTemporaryRedirect,
				Location: resp.URL,
			}
		}
	} else {
		obj, err := handler.s3Client.GetObject(ctx, &s3.GetObjectInput{Bucket: &handler.bucket, Key: &key})
		if err != nil {
			return nil, convertErrNotFound(err)
		}
		return obj.Body, nil
	}
}

// Stat implements blob.BlobStatHandler.
func (handler *blobHandler) Stat(ctx context.Context, repo string, h digest.Digest) (int64, error) {
	key := h.String()
	obj, err := handler.s3Client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &handler.bucket, Key: &key})
	if err != nil {
		return 0, convertErrNotFound(err)
	}
	return *obj.ContentLength, nil
}

// Put implements blob.BlobPutHandler.
func (handler *blobHandler) Put(
	ctx context.Context,
	repo string,
	h digest.Digest,
	contentType string,
	r io.Reader,
) error {
	key := h.String()
	if rc, ok := r.(io.Closer); ok {
		defer rc.Close()
	}

	// The AWS S3 SDK requires a io.ReadSeeker event though the interface only specifies io.Reader
	if _, ok := r.(io.Seeker); !ok {
		if s, err := seekbuf.New(r); err != nil {
			return err
		} else {
			defer func() {
				if err := s.Destroy(); err != nil {
					internalctx.GetLogger(ctx).Warn("ephemeral resource cleanup error", zap.Error(err))
				}
			}()
			if sr, err := s.Get(); err != nil {
				return err
			} else {
				defer sr.Close()
				r = sr
			}
		}
	}

	input := s3.PutObjectInput{
		Bucket: &handler.bucket,
		Key:    &key,
		Body:   r,
	}

	if contentType != "" {
		input.ContentType = &contentType
	}

	_, err := handler.s3Client.PutObject(ctx, &input)
	if err != nil {
		return convertErrNotFound(err)
	}
	return nil
}

func (handler *blobHandler) StartSession(ctx context.Context, repo string) (string, error) {
	if id, err := uuid.NewRandom(); err != nil {
		return "", err
	} else {
		return id.String(), nil
	}
}

func (handler *blobHandler) PutChunk(ctx context.Context, id string, r io.Reader, start int64) (int64, error) {
	if rc, ok := r.(io.Closer); ok {
		defer rc.Close()
	}

	uploadKey := path.Join(chunksPrefix, id)
	var uploadID *string
	var partNumber int32
	var size int64

	if start == 0 {
		if _, err := handler.getUploadID(ctx, uploadKey); err == nil {
			// upload ID must not exist if start == 0!
			return 0, blob.NewErrBadUpload("range is not as expected")
		} else if !errors.Is(err, blob.ErrBadUpload) {
			return 0, err
		} else if upload, err := handler.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
			Bucket: &handler.bucket,
			Key:    &uploadKey,
		}); err != nil {
			return 0, err
		} else {
			uploadID = upload.UploadId
			partNumber = 1
		}
	} else {
		if id, err := handler.getUploadID(ctx, uploadKey); err != nil {
			return 0, err
		} else {
			uploadID = &id
		}

		if parts, err := handler.getExistingParts(ctx, uploadKey, *uploadID); err != nil {
			return 0, err
		} else {
			partNumber = int32(len(parts) + 1)
			for _, part := range parts {
				size += *part.Size
			}
		}
	}

	if size != start {
		return 0, blob.NewErrBadUpload("range is not as expected")
	}

	s, err := seekbuf.New(r)
	if err != nil {
		return 0, fmt.Errorf("failed to create buffer: %w", err)
	}
	defer func() {
		if err := s.Destroy(); err != nil {
			internalctx.GetLogger(ctx).Warn("ephemeral resource cleanup error", zap.Error(err))
		}
	}()

	sr, err := s.Get()
	if err != nil {
		return 0, fmt.Errorf("failed to get buffer reader: %w", err)
	}
	defer sr.Close()

	chunkSize, err := sr.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, fmt.Errorf("failed to measure chunk size: %w", err)
	}
	size += chunkSize

	// Split the chunk into 1 GiB parts. The last part absorbs any remainder so that
	// no non-final part is smaller than the S3 minimum (5 MiB).
	numParts := max(int64(1), chunkSize/splitPartSize)
	for i := range numParts {
		pn := partNumber + int32(i)
		offset := i * splitPartSize
		partSize := splitPartSize
		if i == numParts-1 {
			partSize = chunkSize - offset
		}
		if _, err := handler.s3Client.UploadPart(ctx, &s3.UploadPartInput{
			Bucket:        &handler.bucket,
			Key:           &uploadKey,
			UploadId:      uploadID,
			PartNumber:    &pn,
			Body:          io.NewSectionReader(sr, offset, partSize),
			ContentLength: &partSize,
		}); err != nil {
			return 0, err
		}
	}

	return size, nil
}

func (handler *blobHandler) GetUploadedPartsSize(ctx context.Context, id string) (int64, error) {
	uploadKey := path.Join(chunksPrefix, id)
	var size int64

	if uploadID, err := handler.getUploadID(ctx, uploadKey); err != nil {
		return 0, err
	} else if parts, err := handler.getExistingParts(ctx, uploadKey, uploadID); err != nil {
		return 0, err
	} else {
		for _, part := range parts {
			size += *part.Size
		}
		return size, nil
	}
}

func (handler *blobHandler) CompleteSession(ctx context.Context, repo, id string, digest digest.Digest) error {
	uploadKey := path.Join(chunksPrefix, id)
	if uploadID, err := handler.getUploadID(ctx, uploadKey); err != nil {
		return err
	} else if uploadedParts, err := handler.getExistingParts(ctx, uploadKey, uploadID); err != nil {
		return err
	} else {
		completionParts := make([]s3types.CompletedPart, len(uploadedParts))
		for i, part := range uploadedParts {
			completionParts[i] = s3types.CompletedPart{PartNumber: part.PartNumber, ETag: part.ETag}
		}

		// TODO:
		//   CompleteSession should check if the completed object has the correct digest before copying it to the
		//   final location. AWS supports calculating checksums automatically, but we would need a SHA256 for the
		//   complete object which, unfortunately, is explicitly not supported.
		//   https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity.html#Full-object-checksums
		if _, err := handler.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
			Bucket:          &handler.bucket,
			Key:             &uploadKey,
			UploadId:        &uploadID,
			MultipartUpload: &s3types.CompletedMultipartUpload{Parts: completionParts},
		}); err != nil {
			return err
		}

		finalKey := digest.String()
		if err := handler.copyObject(ctx, uploadKey, finalKey); err != nil {
			return err
		}

		_, err := handler.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &handler.bucket,
			Key:    &uploadKey,
		})
		return err
	}
}

// copyObject copies srcKey to dstKey within the same bucket. For objects larger than the 5 GB CopyObject limit,
// it falls back to a multipart copy using UploadPartCopy.
func (handler *blobHandler) copyObject(ctx context.Context, srcKey, dstKey string) error {
	head, err := handler.s3Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &handler.bucket,
		Key:    &srcKey,
	})
	if err != nil {
		return err
	}
	objectSize := *head.ContentLength

	if objectSize <= maxS3PartSize {
		copySource := handler.bucket + "/" + srcKey
		_, err = handler.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     &handler.bucket,
			Key:        &dstKey,
			CopySource: &copySource,
		})
		return err
	}

	upload, err := handler.s3Client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &handler.bucket,
		Key:    &dstKey,
	})
	if err != nil {
		return err
	}

	abort := func() {
		_, _ = handler.s3Client.AbortMultipartUpload(context.WithoutCancel(ctx), &s3.AbortMultipartUploadInput{
			Bucket:   &handler.bucket,
			Key:      &dstKey,
			UploadId: upload.UploadId,
		})
	}

	numParts := (objectSize + maxS3PartSize - 1) / maxS3PartSize
	completedParts := make([]s3types.CompletedPart, numParts)
	copySource := handler.bucket + "/" + srcKey

	for i := range numParts {
		partNumber := int32(i + 1)
		rangeStart := i * maxS3PartSize
		rangeEnd := min(rangeStart+maxS3PartSize, objectSize) - 1
		copySourceRange := fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd)

		result, err := handler.s3Client.UploadPartCopy(ctx, &s3.UploadPartCopyInput{
			Bucket:          &handler.bucket,
			Key:             &dstKey,
			UploadId:        upload.UploadId,
			PartNumber:      &partNumber,
			CopySource:      &copySource,
			CopySourceRange: &copySourceRange,
		})
		if err != nil {
			abort()
			return err
		}
		completedParts[i] = s3types.CompletedPart{PartNumber: &partNumber, ETag: result.CopyPartResult.ETag}
	}

	if _, err = handler.s3Client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          &handler.bucket,
		Key:             &dstKey,
		UploadId:        upload.UploadId,
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: completedParts},
	}); err != nil {
		abort()
		return err
	}
	return nil
}

// Delete implements blob.BlobDeleteHandler.
func (handler *blobHandler) Delete(ctx context.Context, repo string, h digest.Digest) error {
	key := h.String()
	_, err := handler.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &handler.bucket, Key: &key})
	if err != nil {
		return convertErrNotFound(err)
	}
	return nil
}

func (handler *blobHandler) getUploadID(ctx context.Context, uploadKey string) (string, error) {
	if uploads, err := handler.s3Client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: &handler.bucket,
		Prefix: &uploadKey,
	}); err != nil {
		return "", err
	} else {
		for _, upload := range uploads.Uploads {
			if *upload.Key == uploadKey {
				return *upload.UploadId, nil
			}
		}
		// ListMultipartUploads returns at most 1000 elements.
		// This means that if there are more than 1000 multipart uploads in progress at the same time, finding the upload
		// ID for a specific multipart upload can fail, since it might not be in among the returned elements!
		if uploads.IsTruncated != nil && *uploads.IsTruncated {
			return "", errors.New("too many concurrent uploads. please try again later")
		}
		return "", blob.NewErrBadUpload("unknown upload session")
	}
}

func (handler *blobHandler) getExistingParts(
	ctx context.Context,
	uploadKey string,
	uploadID string,
) ([]s3types.Part, error) {
	paginator := s3.NewListPartsPaginator(handler.s3Client, &s3.ListPartsInput{
		Bucket:   &handler.bucket,
		Key:      &uploadKey,
		UploadId: &uploadID,
	})
	var parts []s3types.Part
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		parts = append(parts, page.Parts...)
	}
	return parts, nil
}

func convertErrNotFound(err error) error {
	var nf *s3types.NotFound
	var nsk *s3types.NoSuchKey
	if errors.As(err, &nf) || errors.As(err, &nsk) {
		err = fmt.Errorf("%w: %w", blob.ErrNotFound, err)
	}
	return err
}
