// Package chartdigest pins the Helm chart of a deployment revision to the manifest digest its tag points to, so
// that the Kubernetes controller deploys exactly that chart and can reuse it across revisions.
package chartdigest

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/distr-sh/distr/internal/db"
	"github.com/distr-sh/distr/internal/distrerr"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/registry/name"
	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/validation"
	"github.com/google/uuid"
)

type chartReference struct {
	host string
	name name.Name
	tag  string
}

// Resolve returns the manifest digest of the chart of an application version, or nil unless the chart is an OCI
// artifact of the organization in this instance's registry. Only the organization's own artifacts are looked up,
// so that a chart URL cannot reveal whether an artifact of another organization exists.
func Resolve(ctx context.Context, organizationID uuid.UUID, version types.ApplicationVersion) (*types.Digest, error) {
	ref := parseChartReference(version)
	if ref == nil || !env.RegistryEnabled() {
		return nil, nil
	}
	if ok, err := isOrganizationRegistryHost(ctx, organizationID, ref.host); err != nil || !ok {
		return nil, err
	}
	digest, err := db.GetArtifactTagDigest(ctx, organizationID, ref.name.OrgName, ref.name.ArtifactName, ref.tag)
	if errors.Is(err, distrerr.ErrNotFound) {
		return nil, nil
	}
	return digest, err
}

func parseChartReference(version types.ApplicationVersion) *chartReference {
	if version.ChartType == nil || *version.ChartType != types.HelmChartTypeOCI ||
		version.ChartUrl == nil || version.ChartVersion == nil {
		return nil
	}
	u, err := url.Parse(*version.ChartUrl)
	if err != nil || u.Scheme != "oci" || u.Host == "" || strings.ContainsAny(u.Path, ":@") {
		return nil
	}
	n, err := name.Parse(strings.Trim(u.Path, "/"))
	if err != nil || n.OrgName == "" || n.ArtifactName == "" {
		return nil
	}
	return &chartReference{host: validation.NormalizeHostname(u.Host), name: *n, tag: *version.ChartVersion}
}

// isOrganizationRegistryHost accepts every host this instance serves the organization's registry on: the instance
// registry host, any custom domain of the organization, since every custom domain serves the registry, and the
// legacy branding registry domain. Unverified custom domains are accepted, since the controller pulls from the
// host in the chart URL either way.
func isOrganizationRegistryHost(ctx context.Context, organizationID uuid.UUID, host string) (bool, error) {
	if host == validation.NormalizeHostname(env.RegistryHost()) {
		return true, nil
	}
	if domain, err := db.GetCustomDomainByDomain(ctx, host); err != nil {
		return false, err
	} else if domain != nil {
		return domain.OrganizationID == organizationID, nil
	}
	if branding, err := db.GetOrganizationBranding(ctx, organizationID); errors.Is(err, distrerr.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return branding.RegistryDomain != nil && validation.NormalizeHostname(*branding.RegistryDomain) == host, nil
	}
}
