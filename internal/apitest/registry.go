package apitest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/distr-sh/distr/api"
	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/errcode"
)

// registryUsername is what an OCI client sends as the username, which the registry ignores.
const registryUsername = "distr"

// Registry is a client sending requests to the OCI registry with password the way an OCI client
// logs in, or without credentials if password is empty. The password is the key of a personal
// access token or the token of a controller.
func (h *Harness) Registry(password string) *Client {
	c := &Client{h: h, baseURL: h.registry.URL}
	if password != "" {
		c.authorization = "Basic " + base64.StdEncoding.EncodeToString([]byte(registryUsername+":"+password))
	}
	return c
}

// SetSlug gives the organization of u the slug its artifacts are named after in the registry.
func (u *User) SetSlug(slug string) {
	u.h.t.Helper()
	org := Expect[api.OrganizationResponse](u.Get("/api/v1/organization"), http.StatusOK)
	u.Put("/api/v1/organization", api.CreateUpdateOrganizationRequest{Name: org.Name, Slug: &slug}).
		ExpectStatus(http.StatusOK)
}

// CreateAccessToken creates a personal access token for u and returns its key.
func (u *User) CreateAccessToken() string {
	u.h.t.Helper()
	return Expect[api.AccessTokenWithKey](
		u.Post(AccessTokens, api.CreateAccessTokenRequest{Label: new("registry")}),
		http.StatusCreated,
	).Key
}

func (h *Harness) repository(name, password string) *remote.Repository {
	h.t.Helper()
	host := h.registryHost()
	repo, err := remote.NewRepository(host + "/" + name)
	if err != nil {
		h.t.Fatalf("invalid repository name %q: %v", name, err)
	}
	repo.PlainHTTP = true
	// An auth.Client without a credential answers a challenge with an error of its own rather than the
	// registry's 401, which RegistryStatus could not report.
	if password == "" {
		repo.Client = h.registry.Client()
	} else {
		repo.Client = &auth.Client{
			Client:     h.registry.Client(),
			Credential: auth.StaticCredential(host, auth.Credential{Username: registryUsername, Password: password}),
		}
	}
	return repo
}

func (h *Harness) registryHost() string {
	u, err := url.Parse(h.registry.URL)
	if err != nil {
		h.t.Fatalf("registry URL %q is invalid: %v", h.registry.URL, err)
	}
	return u.Host
}

// PushImage pushes an image with one layer holding layer as name:tag, the way an OCI client
// does, and returns the digest of its manifest. The name is the organization slug followed by the
// artifact name. Images pushed with the same layer have the same digest.
func (h *Harness) PushImage(password, name, tag, layer string) (digest.Digest, error) {
	h.t.Helper()
	ctx := h.t.Context()
	store := memory.New()
	config, err := pushBlob(ctx, store, ocispec.MediaTypeImageConfig, map[string]any{
		"architecture": "amd64",
		"os":           "linux",
		"rootfs":       map[string]any{"type": "layers", "diff_ids": []string{}},
	})
	if err != nil {
		return "", err
	}
	layerDesc := content.NewDescriptorFromBytes(ocispec.MediaTypeImageLayer, []byte(layer))
	if err := store.Push(ctx, layerDesc, bytes.NewReader([]byte(layer))); err != nil {
		return "", err
	}
	manifest, err := oras.PackManifest(ctx, store, oras.PackManifestVersion1_1, "", oras.PackManifestOptions{
		ConfigDescriptor: &config,
		Layers:           []ocispec.Descriptor{layerDesc},
		// PackManifest stamps the current time otherwise, which gives every push a digest of its own.
		ManifestAnnotations: map[string]string{ocispec.AnnotationCreated: "2026-01-01T00:00:00Z"},
	})
	if err != nil {
		return "", err
	}
	if err := store.Tag(ctx, manifest, tag); err != nil {
		return "", err
	}
	if _, err := oras.Copy(ctx, store, tag, h.repository(name, password), tag, oras.DefaultCopyOptions); err != nil {
		return "", err
	}
	return manifest.Digest, nil
}

func pushBlob(ctx context.Context, store *memory.Store, mediaType string, value any) (ocispec.Descriptor, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	desc := content.NewDescriptorFromBytes(mediaType, data)
	return desc, store.Push(ctx, desc, bytes.NewReader(data))
}

// PullImage pulls name:reference and every blob it references, the way an OCI client does.
func (h *Harness) PullImage(password, name, reference string) error {
	h.t.Helper()
	_, err := oras.Copy(h.t.Context(), h.repository(name, password), reference, memory.New(), reference,
		oras.DefaultCopyOptions)
	return err
}

// RegistryStatus is the status of the registry response err reports, or 0 if err did not come
// from a response.
func RegistryStatus(err error) int {
	if response, ok := errors.AsType[*errcode.ErrorResponse](err); ok {
		return response.StatusCode
	}
	return 0
}
