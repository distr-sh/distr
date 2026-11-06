package handlers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/distr-sh/distr/api"
	"github.com/distr-sh/distr/internal/auth"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/middleware"
	"github.com/distr-sh/distr/internal/respond"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/oaswrap/spec/adapter/chiopenapi"
	"github.com/oaswrap/spec/option"
)

// A file cannot change after its upload, so clients cache it indefinitely without ever revalidating.
const (
	cacheControlPrivateFile = "max-age=31536000, private, immutable"
	cacheControlPublicFile  = "max-age=31536000, public, immutable"
)

var imageDoesNotExist = respond.BadRequest("image does not exist")

func FileRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Files"))
	r.With(middleware.RequireOrgAndRole).Group(func(r chiopenapi.Router) {
		r.Post("/", respond.Handler(createFileHandler)).
			With(option.Description("Upload a new file")).
			With(option.Request(nil, option.ContentType("multipart/formdata"))).
			With(option.Response(http.StatusOK, uuid.Nil))
		r.Route("/{fileId}", func(r chiopenapi.Router) {
			type FileIDRequest struct {
				FileID uuid.UUID `path:"fileId"`
			}

			r.Use(respond.Middleware(fileMiddleware))
			r.Get("/", respond.Handler(getFileHandler)).
				With(option.Description("Get a file by ID")).
				With(option.Request(FileIDRequest{})).
				With(option.Response(http.StatusOK, nil))
		})
	})
}

func PublicFileRouter(r chiopenapi.Router) {
	r.WithOptions(option.GroupTags("Files"))
	type FileIDRequest struct {
		FileID uuid.UUID `path:"fileId"`
	}
	r.Get("/{fileId}", respond.Handler(getPublicFileHandler)).
		With(option.Description("Get a public file by ID")).
		With(option.Request(FileIDRequest{})).
		With(option.Response(http.StatusOK, nil))
}

func getPublicFileHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	if fileID, err := uuid.Parse(r.PathValue("fileId")); err != nil {
		return fileNotFound, nil
	} else if file, err := db.GetFileWithID(ctx, fileID); err != nil {
		if errors.Is(err, distrerr.ErrNotFound) {
			return fileNotFound, nil
		} else {
			return nil, fmt.Errorf("error getting public file: %w", err)
		}
	} else if !file.Public {
		return fileNotFound, nil
	} else if !isServableImageContentType(file.ContentType) {
		// Only serve image types publicly to avoid hosting executable content (e.g. text/html) on the
		// application's own origin. SVGs are allowed but served with script-blocking headers below.
		return fileNotFound, nil
	} else {
		// SVGs are served with a sandboxing CSP so embedded scripts cannot execute in the app's origin.
		header := http.Header{}
		writeSafeImageHeaders(header, file.ContentType)
		header.Set("Cache-Control", cacheControlPublicFile)
		return respond.StreamResponse{ContentType: file.ContentType, Header: header, Body: bytes.NewReader(file.Data)}, nil
	}
}

const mediaTypeSVG = "image/svg+xml"

func parseMediaType(contentType string) string {
	mediaType, _, _ := strings.Cut(contentType, ";")
	return strings.ToLower(strings.TrimSpace(mediaType))
}

// isServableImageContentType reports whether a file is an image that may be served to browsers. It also allows
// image/svg+xml, which callers must serve with script-blocking headers (see writeSafeImageHeaders) so embedded
// scripts cannot execute when the file is opened directly.
func isServableImageContentType(contentType string) bool {
	return strings.HasPrefix(parseMediaType(contentType), "image/")
}

// writeSafeImageHeaders sets the content type and hardening headers for serving a (potentially untrusted) image.
// For SVGs it adds a sandboxing Content-Security-Policy so any embedded scripts cannot run when the file is opened
// as a top-level document; scripts never run when the image is loaded via <img>, which is how logos are displayed.
func writeSafeImageHeaders(header http.Header, contentType string) {
	header.Set("Content-Type", contentType)
	header.Set("X-Content-Type-Options", "nosniff")
	if parseMediaType(contentType) == mediaTypeSVG {
		header.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	}
}

func getFileHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()
	file := internalctx.GetFile(ctx)

	return respond.Stream(file.ContentType, bytes.NewReader(file.Data)).
		WithHeader("Content-Disposition", attachment(file.FileName)).
		WithHeader("Cache-Control", cacheControlPrivateFile), nil
}

func createFileHandler(r *http.Request) (respond.Responder, error) {
	ctx := r.Context()

	auth := auth.Authentication.Require(ctx)

	if file, err := getFileFromRequest(r); err != nil {
		return respond.BadRequest(err.Error()), nil
	} else {
		var orgID *uuid.UUID
		scope := r.FormValue("scope")
		if types.FileScope(scope) != types.FileScopePlatform {
			orgID = auth.CurrentOrgID()
		}
		if err := db.CreateFile(ctx, orgID, file); err != nil {
			return nil, fmt.Errorf("error uploading file: %w", err)
		} else {
			return respond.JSON(file.ID), nil
		}
	}
}

// maxFileSize has to stay well below the request body limit of the API routes (requestSize1MiB in
// internal/routing/routing.go), which also counts the multipart encoding around the file.
const maxFileSize = 500 * 1024

func getFileFromRequest(r *http.Request) (*types.File, error) {
	if err := r.ParseMultipartForm(maxFileSize); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	file := types.File{Public: r.FormValue("public") == "true"}

	if multiPartFile, fileHeader, err := r.FormFile("file"); err != nil {
		return nil, errors.New("file not found")
	} else if fileHeader.Size > maxFileSize {
		return nil, errors.New("file too large (max 500 KiB)")
	} else if data, err := io.ReadAll(multiPartFile); err != nil {
		return nil, err
	} else {
		file.Data = data
		file.FileSize = fileHeader.Size
		file.FileName = fileHeader.Filename
		// Browsers may report a generic content type (e.g. application/octet-stream for .ico files on systems
		// without a matching MIME mapping). Sniff the actual type in that case so the file can later be served
		// via the public file API, which only delivers real image content types.
		if contentType := fileHeader.Header.Get("Content-Type"); contentType != "" &&
			contentType != "application/octet-stream" {
			file.ContentType = contentType
		} else {
			file.ContentType = http.DetectContentType(data)
		}
	}

	return &file, nil
}

func fileMiddleware(r *http.Request) (*http.Request, respond.Responder, error) {
	ctx := r.Context()
	auth := auth.Authentication.Require(ctx)

	if fileID, err := uuid.Parse(r.PathValue("fileId")); err != nil {
		return nil, fileNotFound, nil
	} else if file, err := db.GetFileWithID(ctx, fileID); errors.Is(err, distrerr.ErrNotFound) {
		return nil, fileNotFound, nil
	} else if err != nil {
		return nil, nil, fmt.Errorf("failed to get file: %w", err)
	} else if file.OrganizationID != nil && *file.OrganizationID != *auth.CurrentOrgID() {
		return nil, fileNotFound, nil
	} else {
		return r.WithContext(internalctx.WithFile(ctx, file)), nil, nil
	}
}

// requireImageOwnedBy rejects an image that does not belong to organizationID, where nil stands for a file
// without an organization, which is what an avatar is.
func requireImageOwnedBy(
	ctx context.Context,
	imageID uuid.UUID,
	organizationID *uuid.UUID,
) (respond.Responder, error) {
	if file, err := db.GetFileMetadataWithID(ctx, imageID); errors.Is(err, distrerr.ErrNotFound) {
		return imageDoesNotExist, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	} else if !util.PtrEq(file.OrganizationID, organizationID) {
		return imageDoesNotExist, nil
	} else {
		return nil, nil
	}
}

func patchImage(
	r *http.Request,
	ownerOrganizationID *uuid.UUID,
	update func(ctx context.Context, body api.PatchImageRequest) (any, error),
) (respond.Responder, error) {
	ctx := r.Context()
	if body, resp := decodeJSON[api.PatchImageRequest](r); resp != nil {
		return resp, nil
	} else if body.ImageID == uuid.Nil {
		return respond.BadRequest("imageId can not be empty"), nil
	} else if resp, err := requireImageOwnedBy(ctx, body.ImageID, ownerOrganizationID); resp != nil || err != nil {
		return resp, err
	} else if result, err := update(ctx, body); err != nil {
		return nil, fmt.Errorf("error patching image id: %w", err)
	} else {
		return respond.JSON(result), nil
	}
}
