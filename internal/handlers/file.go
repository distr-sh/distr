package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
	"go.uber.org/zap"
)

func FileRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Files"))
	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Post("/", createFileHandler).
			With(option.Description("Upload a new file")).
			With(option.Request(nil, option.ContentType("multipart/formdata"))).
			With(option.Response(http.StatusOK, uuid.Nil))
		r.Route("/{fileId}", func(r chiopenapi.Router) {
			type FileIDRequest struct {
				FileID uuid.UUID `path:"fileId"`
			}

			r.Use(fileMiddleware)
			r.Get("/", getFileHandler).
				With(option.Description("Get a file by ID")).
				With(option.Request(FileIDRequest{})).
				With(option.Response(http.StatusOK, nil))
			r.Delete("/", deleteFileHandler).
				With(option.Description("Delete a file")).
				With(option.Request(FileIDRequest{}))
		})
	})
}

func getFileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	file := internalctx.GetFile(ctx)

	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.FileName))
	w.Header().Set("Cache-Control", "max-age=604800, private")

	// Write file data to response
	if _, err := w.Write(file.Data); err != nil {
		internalctx.GetLogger(ctx).Warn("failed to write file to response", zap.Error(err))
		http.Error(w, "failed to write file to response", http.StatusInternalServerError)
	}
}

func PublicFileRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Files"))
	type FileIDRequest struct {
		FileID uuid.UUID `path:"fileId"`
	}
	r.Get("/{fileId}", getPublicFileHandler).
		With(option.Description("Get a public file by ID")).
		With(option.Request(FileIDRequest{})).
		With(option.Response(http.StatusOK, nil))
}

func getPublicFileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	if fileID, err := uuid.Parse(r.PathValue("fileId")); err != nil {
		http.NotFound(w, r)
	} else if file, err := db.GetFileWithID(ctx, fileID); err != nil {
		if errors.Is(err, apierrors.ErrNotFound) {
			http.NotFound(w, r)
		} else {
			log.Error("error getting public file", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	} else if !file.Public {
		http.NotFound(w, r)
	} else if !isPublicServableContentType(file.ContentType) {
		// Only serve safe image types publicly to avoid hosting executable content
		// (e.g. text/html or image/svg+xml) on the application's own origin.
		http.NotFound(w, r)
	} else {
		w.Header().Set("Content-Type", file.ContentType)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "max-age=604800, public")
		if _, err := w.Write(file.Data); err != nil {
			log.Warn("failed to write file to response", zap.Error(err))
			http.Error(w, "failed to write file to response", http.StatusInternalServerError)
		}
	}
}

// isPublicServableContentType reports whether a file may be served via the public
// file endpoint. Only image types are allowed, and image/svg+xml is excluded
// because SVGs can contain scripts and would execute in the app's origin.
func isPublicServableContentType(contentType string) bool {
	mediaType, _, _ := strings.Cut(contentType, ";")
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	return strings.HasPrefix(mediaType, "image/") && mediaType != "image/svg+xml"
}

func deleteFileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)
	file := internalctx.GetFile(ctx)

	if err := db.DeleteFileWithID(ctx, file.ID); err != nil {
		log.Warn("error deleting file", zap.Error(err))
		if errors.Is(err, apierrors.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent)
		} else if errors.Is(err, apierrors.ErrConflict) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

func createFileHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := internalctx.GetLogger(ctx)

	auth := auth.Authentication.Require(ctx)

	if file, err := getFileFromRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	} else {
		var orgID *uuid.UUID
		scope := r.FormValue("scope")
		if types.FileScope(scope) != types.FileScopePlatform {
			orgID = auth.CurrentOrgID()
		}
		if err := db.CreateFile(ctx, orgID, file); err != nil {
			log.Warn("error uploading file", zap.Error(err))
			sentry.GetHubFromContext(ctx).CaptureException(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			RespondJSON(w, file.ID)
		}
	}
}

func getFileFromRequest(r *http.Request) (*types.File, error) {
	if err := r.ParseMultipartForm(102400); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file := types.File{Public: r.FormValue("public") == "true"}

	if multiPartFile, fileHeader, err := r.FormFile("file"); err != nil {
		return nil, errors.New("file not found")
	} else if fileHeader.Size > 102400 {
		return nil, errors.New("file too large (max 100 KiB)")
	} else if data, err := io.ReadAll(multiPartFile); err != nil {
		return nil, err
	} else {
		file.Data = data
		file.FileSize = fileHeader.Size
		file.FileName = fileHeader.Filename
		if contentType := util.PtrTo(fileHeader.Header.Get("Content-Type")); contentType != nil {
			file.ContentType = *contentType
		} else {
			file.ContentType = "application/octet-stream"
		}
	}

	return &file, nil
}

func fileMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)
		auth := auth.Authentication.Require(ctx)

		if fileID, err := uuid.Parse(r.PathValue("fileId")); err != nil {
			http.NotFound(w, r)
		} else if file, err := db.GetFileWithID(ctx, fileID); err != nil {
			if errors.Is(err, apierrors.ErrNotFound) {
				http.NotFound(w, r)
			} else {
				log.Error("error getting file", zap.Error(err))
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		} else {
			if file.OrganizationID != nil && *file.OrganizationID != *auth.CurrentOrgID() {
				http.NotFound(w, r)
			} else {
				h.ServeHTTP(w, r.WithContext(internalctx.WithFile(ctx, file)))
			}
		}
	})
}

func patchImageHandler(patchImage func(ctx context.Context, body api.PatchImageRequest) (any, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log := internalctx.GetLogger(ctx)

		body, err := JsonBody[api.PatchImageRequest](w, r)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		} else if body.ImageID == uuid.Nil {
			http.Error(w, "imageId can not be empty", http.StatusBadRequest)
			return
		}

		if result, err := patchImage(ctx, body); err != nil {
			log.Warn("error patching image id", zap.Error(err))
			if errors.Is(err, apierrors.ErrNotFound) {
				w.WriteHeader(http.StatusNoContent)
			} else if errors.Is(err, apierrors.ErrConflict) {
				http.Error(w, err.Error(), http.StatusBadRequest)
			} else {
				sentry.GetHubFromContext(ctx).CaptureException(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		} else {
			RespondJSON(w, result)
		}
	}
}
