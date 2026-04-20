package cleanup

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/util"
	"github.com/opencontainers/go-digest"
	"go.uber.org/zap"
)

func RunArtifactBlobCleanup(ctx context.Context) error {
	log := internalctx.GetLogger(ctx)
	s3Client := internalctx.GetS3Client(ctx)

	bucket := env.RegistryS3Config().Bucket
	cutoff := time.Now().Add(-env.CleanupArtifactBlobMinAge())

	var referencedDigests map[string]struct{}
	if digests, err := db.GetAllReferencedBlobDigests(ctx); err != nil {
		return err
	} else {
		referencedDigests = make(map[string]struct{}, len(digests))
		util.InsertKeys(referencedDigests, slices.Values(digests))
	}

	var skippedStillReferenced int
	var skippedMinAge int
	var deleted int
	var errs []error

	flushDelete := func(batch []s3types.ObjectIdentifier) error {
		result, err := s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &bucket,
			Delete: &s3types.Delete{Objects: batch, Quiet: new(true)},
		})
		if err != nil {
			return fmt.Errorf("deleting S3 objects: %w", err)
		}
		deleted += len(batch) - len(result.Errors)
		for _, e := range result.Errors {
			err := fmt.Errorf("failed to delete %s: %s", util.PtrDerefOrDefault(e.Key), util.PtrDerefOrDefault(e.Message))
			errs = append(errs, err)
		}
		return nil
	}

	var toDelete []s3types.ObjectIdentifier
	paginator := s3.NewListObjectsV2Paginator(s3Client, &s3.ListObjectsV2Input{Bucket: &bucket})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("listing S3 objects: %w", err)
		}
		for _, obj := range page.Contents {
			if obj.Key == nil {
				continue
			}
			if _, err := digest.Parse(*obj.Key); err != nil {
				continue
			}
			if _, ok := referencedDigests[*obj.Key]; ok {
				skippedStillReferenced++
				continue
			}
			if obj.LastModified != nil && obj.LastModified.After(cutoff) {
				skippedMinAge++
				continue
			}
			toDelete = append(toDelete, s3types.ObjectIdentifier{Key: obj.Key})
			if len(toDelete) >= 1000 {
				if err := flushDelete(toDelete); err != nil {
					return err
				}
				toDelete = toDelete[:0]
			}
		}
	}

	if len(toDelete) > 0 {
		if err := flushDelete(toDelete); err != nil {
			return err
		}
	}

	log.Info("ArtifactBlob cleanup finished",
		zap.Int("skippedMinAge", skippedMinAge),
		zap.Int("skippedStillReferenced", skippedStillReferenced),
		zap.Int("deleted", deleted),
		zap.Int("errors", len(errs)),
	)

	return errors.Join(errs...)
}
