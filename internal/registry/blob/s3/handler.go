package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/blob"
	"github.com/distr-sh/distr/internal/tmpstream"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/opencontainers/go-digest"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-sdk-go-v2/otelaws"
	"go.uber.org/zap"
)

const (
	chunksPrefix = "chunks"
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

func NewBlobHandler(ctx context.Context) (blob.BlobHandler, error) {
	s3Config := env.RegistryS3Config()

	var s3Client *s3.Client
	if config, err := awsconfig.LoadDefaultConfig(ctx); err != nil {
		s3Client = s3.New(s3.Options{}, clientOpts(s3Config))
		if s3Config.ResignForGCP {
			s3Client = s3.New(s3.Options{}, clientOpts(s3Config), ResignForGCP)
		}
	} else {
		otelaws.AppendMiddlewares(&config.APIOptions)
		s3Client = s3.NewFromConfig(config, clientOpts(s3Config))
		if s3Config.ResignForGCP {
			s3Client = s3.NewFromConfig(config, clientOpts(s3Config), ResignForGCP)
		}
	}

	if s3Config.CreateBucket {
		if err := ensureBucketExists(ctx, s3Client, s3Config.Bucket, s3Config.Region); err != nil {
			return nil, err

		}
	}

	return &blobHandler{
		s3Client:        s3Client,
		s3PresignClient: s3.NewPresignClient(s3Client),
		allowRedirect:   s3Config.AllowRedirect,
		bucket:          s3Config.Bucket,
	}, nil
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

func clientOpts(s3Config env.S3Config) func(o *s3.Options) {
	return func(o *s3.Options) {
		o.Region = s3Config.Region
		o.BaseEndpoint = s3Config.Endpoint
		o.UsePathStyle = s3Config.UsePathStyle
		if s3Config.RequestChecksumCalculationWhenRequired {
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		}
		if s3Config.ResponseChecksumValidationWhenRequired {
			o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		}
		if s3Config.AccessKeyID != nil && s3Config.SecretAccessKey != nil {
			o.Credentials = aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(*s3Config.AccessKeyID, *s3Config.SecretAccessKey, ""),
			)
		}
	}
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
		if s, err := tmpstream.New(r); err != nil {
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

	if s, err := tmpstream.New(r); err != nil {
		return 0, fmt.Errorf("failed to create tmp stream: %w", err)
	} else {
		defer func() {
			if err := s.Destroy(); err != nil {
				internalctx.GetLogger(ctx).Warn("ephemeral resource cleanup error", zap.Error(err))
			}
		}()
		if sr, err := s.Get(); err != nil {
			return 0, fmt.Errorf("failed to get tmp stream reader: %w", err)
		} else {
			defer sr.Close()
			if srl, err := io.Copy(io.Discard, sr); err != nil {
				return 0, fmt.Errorf("failed to get stream reader length: %w", err)
			} else {
				size += srl
				if _, err := sr.Seek(0, io.SeekStart); err != nil {
					return 0, fmt.Errorf("failed to reset stream reader position to 0: %w", err)
				}
			}
			r = sr
		}
	}

	if _, err := handler.s3Client.UploadPart(ctx, &s3.UploadPartInput{
		Bucket:     &handler.bucket,
		Key:        &uploadKey,
		UploadId:   uploadID,
		PartNumber: &partNumber,
		Body:       r,
	}); err != nil {
		return 0, err
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
		} else if _, err := handler.s3Client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket:     &handler.bucket,
			Key:        new(digest.String()),
			CopySource: new(path.Join(handler.bucket, uploadKey)),
		}); err != nil {
			return err
		} else if _, err := handler.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: &handler.bucket,
			Key:    &uploadKey,
		}); err != nil {
			return err
		} else {
			return nil
		}
	}
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
	if result, err := handler.s3Client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   &handler.bucket,
		Key:      &uploadKey,
		UploadId: &uploadID,
	}); err != nil {
		return nil, err
	} else if result.IsTruncated != nil && *result.IsTruncated {
		// ListParts returns at most 1000 elements.
		// Thus, we can not currently handle uploads with more than 1000 chunks!
		return nil, blob.NewErrBadUpload("blob uploads with more than 1000 chunks are not supported")
	} else {
		return result.Parts, nil
	}
}

func convertErrNotFound(err error) error {
	var nf *s3types.NotFound
	var nsk *s3types.NoSuchKey
	if errors.As(err, &nf) || errors.As(err, &nsk) {
		err = fmt.Errorf("%w: %w", blob.ErrNotFound, err)
	}
	return err
}
