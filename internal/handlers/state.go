package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/getsentry/sentry-go"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type lockInfo struct {
	ID        string `json:"ID"`
	Operation string `json:"Operation"`
	Info      string `json:"Info"`
	Who       string `json:"Who"`
	Version   string `json:"Version"`
	Created   string `json:"Created"`
	Path      string `json:"Path"`
}

func resolveStateDeployment(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, *types.DeploymentTargetFull, bool) {
	deploymentTarget := internalctx.GetDeploymentTarget(r.Context())

	if deploymentTarget.Type != types.DeploymentTypeOpenTofu {
		http.Error(w, "state backend is only available for opentofu deployment targets", http.StatusBadRequest)
		return uuid.Nil, nil, false
	}

	deploymentID, err := uuid.Parse(chi.URLParam(r, "deploymentID"))
	if err != nil {
		http.Error(w, "deploymentID is not a valid UUID", http.StatusBadRequest)
		return uuid.Nil, nil, false
	}

	if !slices.ContainsFunc(deploymentTarget.Deployments, func(d types.DeploymentWithLatestRevision) bool {
		return d.ID == deploymentID
	}) {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return uuid.Nil, nil, false
	}

	return deploymentID, deploymentTarget, true
}

func stateGetHandler(s3Client *s3.Client, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, _, ok := resolveStateDeployment(w, r)
		if !ok {
			return
		}
		handleStateGet(w, r, s3Client, bucket, deploymentID, internalctx.GetLogger(r.Context()))
	}
}

func statePostHandler(s3Client *s3.Client, bucket string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, dt, ok := resolveStateDeployment(w, r)
		if !ok {
			return
		}
		handleStatePost(w, r, s3Client, bucket, deploymentID, dt.OrganizationID, internalctx.GetLogger(r.Context()))
	}
}

func stateLockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, dt, ok := resolveStateDeployment(w, r)
		if !ok {
			return
		}
		handleStateLock(w, r, deploymentID, dt.OrganizationID, internalctx.GetLogger(r.Context()))
	}
}

func stateUnlockHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deploymentID, _, ok := resolveStateDeployment(w, r)
		if !ok {
			return
		}
		handleStateUnlock(w, r, deploymentID, internalctx.GetLogger(r.Context()))
	}
}

func handleStateGet(
	w http.ResponseWriter, r *http.Request,
	s3Client *s3.Client, bucket string,
	deploymentID uuid.UUID, log *zap.Logger,
) {
	ctx := r.Context()
	s3Key := fmt.Sprintf("state/%s", deploymentID.String())

	obj, err := s3Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &s3Key,
	})
	if err != nil {
		var nf *s3types.NoSuchKey
		var notFound *s3types.NotFound
		if errors.As(err, &nf) || errors.As(err, &notFound) {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		log.Error("failed to get state from S3", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer obj.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, obj.Body); err != nil {
		log.Warn("failed to write state response", zap.Error(err))
	}
}

func handleStatePost(
	w http.ResponseWriter, r *http.Request,
	s3Client *s3.Client, bucket string,
	deploymentID, organizationID uuid.UUID, log *zap.Logger,
) {
	ctx := r.Context()

	if _, err := db.GetOrCreateState(ctx, deploymentID, organizationID); err != nil {
		log.Error("failed to upsert state metadata", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	const maxStateSize = 20 << 20 // 20 MB
	r.Body = http.MaxBytesReader(w, r.Body, maxStateSize)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("failed to read state body", zap.Error(err))
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, http.StatusText(http.StatusRequestEntityTooLarge), http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}
		return
	}

	s3Key := fmt.Sprintf("state/%s", deploymentID.String())
	contentType := "application/json"

	_, err = s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &s3Key,
		Body:        bytes.NewReader(body),
		ContentType: &contentType,
	})
	if err != nil {
		log.Error("failed to store state in S3", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleStateLock(
	w http.ResponseWriter, r *http.Request,
	deploymentID, organizationID uuid.UUID, log *zap.Logger,
) {
	ctx := r.Context()

	li, err := JsonBody[lockInfo](w, r)
	if err != nil {
		return
	}

	if _, err := db.GetOrCreateState(ctx, deploymentID, organizationID); err != nil {
		log.Error("failed to upsert state metadata", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	lockInfoJSON, err := json.Marshal(li)
	if err != nil {
		log.Error("failed to marshal lock info", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := db.LockState(ctx, deploymentID, li.ID, string(lockInfoJSON)); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			state, getErr := db.GetState(ctx, deploymentID)
			if getErr != nil {
				log.Error("failed to get existing lock info", zap.Error(getErr))
				sentry.GetHubFromContext(ctx).CaptureException(getErr)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			if state.LockInfo != nil {
				_, _ = w.Write([]byte(*state.LockInfo))
			}
			return
		}
		log.Error("failed to lock state", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func handleStateUnlock(
	w http.ResponseWriter, r *http.Request,
	deploymentID uuid.UUID, log *zap.Logger,
) {
	ctx := r.Context()

	li, err := JsonBody[lockInfo](w, r)
	if err != nil {
		return
	}

	if err := db.UnlockState(ctx, deploymentID, li.ID); err != nil {
		if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, "lock ID mismatch", http.StatusConflict)
			return
		}
		log.Error("failed to unlock state", zap.Error(err))
		sentry.GetHubFromContext(ctx).CaptureException(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func newStateS3Client(ctx context.Context) (*s3.Client, string) {
	s3Config := env.RegistryS3Config()
	opts := func(o *s3.Options) {
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

	var s3Client *s3.Client
	config, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(s3Config.Region))
	if err != nil {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			hub.CaptureException(fmt.Errorf("failed to load AWS config for state S3 client: %w", err))
		}
		s3Client = s3.New(s3.Options{}, opts)
	} else {
		s3Client = s3.NewFromConfig(config, opts)
	}
	return s3Client, s3Config.Bucket
}
