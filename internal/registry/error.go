// Copyright 2018 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/distr-sh/distr/internal/apierrors"
	"github.com/distr-sh/distr/internal/auth"
	"github.com/distr-sh/distr/internal/registry/authz"
	registryerror "github.com/distr-sh/distr/internal/registry/error"
)

type regError struct {
	Status  int
	Code    string
	Message string
	Header  http.Header
	Error   error
}

func (r *regError) Write(resp http.ResponseWriter) error {
	for key, values := range r.Header {
		for _, value := range values {
			resp.Header().Add(key, value)
		}
	}
	resp.WriteHeader(r.Status)

	type err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	type wrap struct {
		Errors []err `json:"errors"`
	}
	return json.NewEncoder(resp).Encode(wrap{
		Errors: []err{
			{
				Code:    r.Code,
				Message: r.Message,
			},
		},
	})
}

// regErrInternal returns an internal server error.
func regErrInternal(err error) *regError {
	return &regError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL_SERVER_ERROR",
		Message: err.Error(),
		Error:   err,
	}
}

func regErrManifestInvalid(err error) *regError {
	return &regError{
		Status:  http.StatusBadRequest,
		Code:    "MANIFEST_INVALID",
		Message: err.Error(),
		Error:   err,
	}
}

var regErrBlobUnknown = &regError{
	Status:  http.StatusNotFound,
	Code:    "BLOB_UNKNOWN",
	Message: "Unknown blob",
}

var regErrUnsupported = &regError{
	Status:  http.StatusMethodNotAllowed,
	Code:    "UNSUPPORTED",
	Message: "Unsupported operation",
}

var regErrDigestMismatch = &regError{
	Status:  http.StatusBadRequest,
	Code:    "DIGEST_INVALID",
	Message: "digest does not match contents",
}

var regErrDigestInvalid = &regError{
	Status:  http.StatusBadRequest,
	Code:    "DIGEST_INVALID",
	Message: "invalid digest",
}

var regErrNameInvalid = &regError{
	Status:  http.StatusBadRequest,
	Code:    "NAME_INVALID",
	Message: "invalid name",
}

var regErrManifestUnknown = &regError{
	Status:  http.StatusNotFound,
	Code:    "MANIFEST_UNKNOWN",
	Message: "Unknown manifest",
}

var regErrNameUnknown = &regError{
	Status:  http.StatusNotFound,
	Code:    "NAME_UNKNOWN",
	Message: "Unknown name",
}

var regErrMethodUnknown = &regError{
	Status:  http.StatusBadRequest,
	Code:    "METHOD_UNKNOWN",
	Message: "We don't understand your method + url",
}

func regErrDenied(message string) *regError {
	return &regError{
		Status:  http.StatusForbidden,
		Code:    "DENIED",
		Message: message,
	}
}

var regErrTooManyRequests = &regError{
	Status:  http.StatusTooManyRequests,
	Code:    "TOOMANYREQUESTS",
	Message: "anonymous pull rate limit exceeded, authenticate to raise it",
}

var regErrUnauthorized = &regError{
	Status:  http.StatusUnauthorized,
	Code:    "UNAUTHORIZED",
	Message: "authentication required",
	Header:  auth.ArtifactsAuthenticateHeader,
}

// regErrAuthz maps an authorization error of a route that names an artifact.
func regErrAuthz(err error) *regError {
	switch {
	case errors.Is(err, authz.ErrAuthenticationRequired):
		return regErrUnauthorized
	case errors.Is(err, authz.ErrAccessDenied):
		return regErrDenied(err.Error())
	case errors.Is(err, registryerror.ErrInvalidArtifactName):
		return regErrNameInvalid
	default:
		return regErrInternal(err)
	}
}

// regErrBlobAuthz is regErrAuthz for the blob routes, where a digest the caller may not see is
// reported as an unknown blob.
func regErrBlobAuthz(err error) *regError {
	if errors.Is(err, apierrors.ErrNotFound) {
		return regErrBlobUnknown
	}
	return regErrAuthz(err)
}

var regErrDeniedQuotaExceeded = &regError{
	Status:  http.StatusForbidden,
	Code:    "DENIED",
	Message: "You have exhausted your organizations tag quota",
}

var regErrTagAlreadyExists = &regError{
	Status:  http.StatusConflict,
	Code:    "UNSUPPORTED",
	Message: "this tag already exists and cannot be overwritten",
}

var regErrConflict = &regError{
	Status:  http.StatusConflict,
	Code:    "DENIED",
	Message: "operation cannot be completed due to a conflict",
}

var regErrBadRequest = &regError{
	Status:  http.StatusBadRequest,
	Code:    "UNSUPPORTED",
	Message: "the request is invalid or not supported",
}
