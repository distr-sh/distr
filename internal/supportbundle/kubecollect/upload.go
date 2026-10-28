package kubecollect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/distr-sh/distr/internal/buildconfig"
	"github.com/distr-sh/distr/internal/types"
	"golang.org/x/sync/errgroup"
)

const (
	uploadConcurrency     = 4
	uploadRequestTimeout  = time.Minute
	uploadErrorBodyMaxLen = 1024
)

// BundleFile is a file of a bundle collected into a directory, named by its slash-separated path
// relative to that directory.
type BundleFile struct {
	Name        string
	Path        string
	Size        int64
	ContentType types.SupportBundleResourceContentType
}

// ListBundleFiles lists the regular files below dir. The customer may have edited or deleted files
// before the upload, so this is read from the directory rather than remembered from the collection.
func ListBundleFiles(dir string) ([]BundleFile, error) {
	var files []BundleFile
	err := filepath.WalkDir(dir, func(filePath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		} else if !entry.Type().IsRegular() {
			return nil
		} else if info, err := entry.Info(); err != nil {
			return err
		} else if rel, err := filepath.Rel(dir, filePath); err != nil {
			return err
		} else {
			name := filepath.ToSlash(rel)
			files = append(files, BundleFile{
				Name:        name,
				Path:        filePath,
				Size:        info.Size(),
				ContentType: types.SupportBundleResourceContentTypeOfFile(name),
			})
			return nil
		}
	})
	return files, err
}

// Uploader sends every file of a bundle to Distr as a resource of its own, through the endpoint
// the Docker collect script uploads to as well.
type Uploader struct {
	// URL is the collect API of the bundle, ending in /support-bundle-collect/<bundle ID>.
	URL    string
	Secret string
	Client *http.Client
}

type UploadResult struct {
	Uploaded int
	Failed   []string
}

// UploadRejectedError is a response that every further upload would receive as well, for example
// because the bundle was canceled or finalized in the meantime.
type UploadRejectedError struct {
	Status  string
	Message string
}

func (e *UploadRejectedError) Error() string {
	return fmt.Sprintf("Distr rejected the upload (%s): %s", e.Status, e.Message)
}

// Upload uploads the files and reports each finished one to progress. A file that fails is recorded
// in the result and the upload continues, unless Distr rejects it with an UploadRejectedError.
func (u Uploader) Upload(ctx context.Context, files []BundleFile, progress func(done int)) (UploadResult, error) {
	var (
		mu     sync.Mutex
		result UploadResult
	)
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(uploadConcurrency)
	for _, file := range files {
		group.Go(func() error {
			if ctx.Err() != nil {
				return nil
			}
			err := u.upload(ctx, file)
			if rejected, ok := errors.AsType[*UploadRejectedError](err); ok {
				return rejected
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				result.Failed = append(result.Failed, fmt.Sprintf("%s: %v", file.Name, err))
			} else {
				result.Uploaded++
			}
			if progress != nil {
				progress(result.Uploaded + len(result.Failed))
			}
			return nil
		})
	}
	err := group.Wait()
	slices.Sort(result.Failed)
	return result, err
}

func (u Uploader) upload(ctx context.Context, file BundleFile) error {
	content, err := os.ReadFile(file.Path)
	if err != nil {
		return err
	}
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	if err := form.WriteField("name", file.Name); err != nil {
		return err
	} else if err := form.WriteField("contentType", string(file.ContentType)); err != nil {
		return err
	} else if part, err := form.CreateFormFile("content", path.Base(file.Name)); err != nil {
		return err
	} else if _, err := part.Write(content); err != nil {
		return err
	} else if err := form.Close(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, uploadRequestTimeout)
	defer cancel()
	requestURL := u.URL + "/resources?" + url.Values{"bundleSecret": {u.Secret}}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, &body)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", form.FormDataContentType())
	request.Header.Set("User-Agent", "DistrSupportBundleCollector/"+buildconfig.Version())

	response, err := u.Client.Do(request)
	if urlErr, ok := errors.AsType[*url.Error](err); ok {
		// The URL holds the bundle secret, which must not end up on the terminal.
		return urlErr.Err
	} else if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusBadRequest {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}

	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, uploadErrorBodyMaxLen))
	message := strings.TrimSpace(string(responseBody))
	if rejectsEveryUpload(response.StatusCode) {
		return &UploadRejectedError{Status: response.Status, Message: message}
	}
	return fmt.Errorf("%s: %s", response.Status, message)
}

// rejectsEveryUpload tells the client errors apart that depend on the bundle or the secret rather
// than on the file, since the requests differ in nothing else.
func rejectsEveryUpload(status int) bool {
	return status >= http.StatusBadRequest && status < http.StatusInternalServerError &&
		status != http.StatusRequestEntityTooLarge && status != http.StatusTooManyRequests
}
