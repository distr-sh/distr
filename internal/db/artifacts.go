package db

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/containers/image/v5/manifest"
	"github.com/distr-sh/distr/internal/apierrors"
	internalctx "github.com/distr-sh/distr/internal/context"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/types"
	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	artifactOutputExpr = ` a.id, a.created_at, a.organization_id, a.name, a.image_id `

	artifactDownloadsOutExpr = `
		count(DISTINCT avpl.id) AS downloads_total,
		count(DISTINCT avpl.useraccount_id) FILTER (WHERE avpl.customer_organization_id IS NULL)
			AS downloaded_by_users_count,
		coalesce(array_agg(DISTINCT avpl.useraccount_id)
			FILTER (WHERE avpl.useraccount_id IS NOT NULL
				AND avpl.customer_organization_id IS NULL), ARRAY[]::UUID[])
			AS downloaded_by_users,
		count(DISTINCT avpl.customer_organization_id)
			AS downloaded_by_customer_organizations_count,
		coalesce(array_agg(DISTINCT avpl.customer_organization_id)
			FILTER (WHERE avpl.customer_organization_id IS NOT NULL), ARRAY[]::UUID[])
			AS downloaded_by_customer_organizations `

	artifactWithDownloadsOutputExpr = artifactOutputExpr +
		", o.slug AS organization_slug," +
		artifactDownloadsOutExpr

	artifactVersionOutputExpr = `
		v.id,
		v.created_at,
		v.created_by_useraccount_id,
		v.updated_at,
		v.updated_by_useraccount_id,
		v.name,
		v.manifest_blob_digest,
		v.manifest_blob_size,
		v.manifest_content_type,
		v.manifest_data,
		v.artifact_id `
)

func GetArtifactsByOrgID(ctx context.Context, orgID uuid.UUID) ([]types.ArtifactWithDownloads, error) {
	db := internalctx.GetDb(ctx)
	if artifactRows, err := db.Query(ctx, `
			SELECT `+artifactWithDownloadsOutputExpr+`
			FROM Artifact a
			JOIN Organization o
				ON o.id = a.organization_id
			LEFT JOIN ArtifactVersion av
				ON a.id = av.artifact_id
			LEFT JOIN ArtifactVersionPull avpl
				ON avpl.artifact_version_id = av.id
			WHERE a.organization_id = @orgId
			GROUP BY a.id, a.created_at, a.organization_id, a.name, o.slug
			ORDER BY max(av.created_at) DESC`,
		pgx.NamedArgs{
			"orgId": orgID,
		}); err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	} else if artifacts, err := pgx.CollectRows(
		artifactRows, pgx.RowToStructByName[types.ArtifactWithDownloads],
	); err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	} else {
		return artifacts, nil
	}
}

func GetArtifactsByLicenseOwnerID(ctx context.Context, orgID uuid.UUID, ownerID uuid.UUID) (
	[]types.ArtifactWithDownloads, error,
) {
	db := internalctx.GetDb(ctx)
	if artifactRows, err := db.Query(ctx, `
			SELECT `+artifactWithDownloadsOutputExpr+`
			FROM Artifact a
			JOIN Organization o
				ON o.id = a.organization_id
			LEFT JOIN Organization_UserAccount oua
				ON oua.organization_id = a.organization_id AND oua.customer_organization_id = @ownerId
			LEFT JOIN ArtifactVersion av
				ON a.id = av.artifact_id
			LEFT JOIN ArtifactVersionPull avpl
				ON avpl.artifact_version_id = av.id AND avpl.useraccount_id = oua.user_account_id
			WHERE a.organization_id = @orgId
			AND EXISTS(
				SELECT ala.id
				FROM ArtifactLicense_Artifact ala
				INNER JOIN ArtifactLicense al ON ala.artifact_license_id = al.id
				WHERE al.customer_organization_id = @ownerId AND (al.expires_at IS NULL OR al.expires_at > now())
				AND ala.artifact_id = a.id
			)
			GROUP BY a.id, a.created_at, a.organization_id, a.name, o.slug
			ORDER BY max(av.created_at) DESC`,
		pgx.NamedArgs{
			"orgId":   orgID,
			"ownerId": ownerID,
		}); err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	} else if artifacts, err := pgx.CollectRows(
		artifactRows, pgx.RowToStructByName[types.ArtifactWithDownloads],
	); err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	} else {
		return artifacts, nil
	}
}

func GetArtifactByID(
	ctx context.Context,
	orgID uuid.UUID,
	artifactID uuid.UUID,
	customerOrgID *uuid.UUID,
) (
	*types.ArtifactWithTaggedVersion,
	error,
) {
	db := internalctx.GetDb(ctx)
	isVendorUser := customerOrgID == nil

	rows, err := db.Query(
		ctx, `
			SELECT `+artifactWithDownloadsOutputExpr+`
			FROM Artifact a
			JOIN Organization o
				ON o.id = a.organization_id
			LEFT JOIN ArtifactVersion av
				ON a.id = av.artifact_id
			LEFT JOIN ArtifactVersionPull avpl
				ON @isVendorUser AND avpl.artifact_version_id = av.id
			LEFT JOIN Organization_UserAccount oua_dl
				ON oua_dl.organization_id = a.organization_id
					AND oua_dl.user_account_id = avpl.useraccount_id
			WHERE a.id = @id AND a.organization_id = @orgId
			GROUP BY a.id, a.created_at, a.organization_id, a.name, o.slug`,
		pgx.NamedArgs{
			"id":           artifactID,
			"orgId":        orgID,
			"isVendorUser": isVendorUser,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifact by ID: %w", err)
	}

	artifact, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.ArtifactWithDownloads])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to collect artifact by ID: %w", err)
	} else if versions, err := GetVersionsForArtifact(ctx, artifact.ID, customerOrgID); err != nil {
		return nil, fmt.Errorf("failed to get artifact versions: %w", err)
	} else if customerOrgID != nil && len(versions) == 0 {
		return nil, apierrors.ErrNotFound
	} else {
		return &types.ArtifactWithTaggedVersion{ArtifactWithDownloads: *artifact, Versions: versions}, nil
	}
}

func GetArtifactByName(ctx context.Context, orgSlug, name string) (*types.Artifact, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+artifactOutputExpr+`
			FROM Artifact a
			JOIN Organization o on o.id = a.organization_id
			WHERE o.slug = @orgSlug AND a.name = @name
			ORDER BY a.name`,
		pgx.NamedArgs{
			"orgSlug": orgSlug,
			"name":    name,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query artifacts: %w", err)
	}
	if a, err := pgx.CollectExactlyOneRow(rows, pgx.RowToAddrOfStructByName[types.Artifact]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get artifact: %w", err)
	} else {
		return a, nil
	}
}

func GetVersionsForArtifact(ctx context.Context, artifactID uuid.UUID, customerOrgID *uuid.UUID) (
	[]types.TaggedArtifactVersion,
	error,
) {
	isVendorUser := customerOrgID == nil

	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(ctx, `
			SELECT
				av.id,
				av.created_at,
				av.manifest_blob_digest,
				av.manifest_content_type,
				av.manifest_data,
				coalesce((
					SELECT array_agg(row (avt.id, avt.name, (
						SELECT ROW(
							count(distinct avplx.id),
							count(DISTINCT avplx.useraccount_id)
								FILTER (WHERE avplx.customer_organization_id IS NULL),
							coalesce(array_agg(DISTINCT avplx.useraccount_id)
								FILTER (WHERE avplx.useraccount_id IS NOT NULL
									AND avplx.customer_organization_id IS NULL), ARRAY[]::UUID[]),
							count(DISTINCT avplx.customer_organization_id),
							coalesce(array_agg(DISTINCT avplx.customer_organization_id)
								FILTER (WHERE avplx.customer_organization_id IS NOT NULL), ARRAY[]::UUID[])
						)
						FROM ArtifactVersionPull avplx
						WHERE @isVendorUser AND avplx.artifact_version_id = avt.id
						)) ORDER BY avt.name
					)
					FROM ArtifactVersion avt
					WHERE avt.manifest_blob_digest = av.manifest_blob_digest
					AND avt.artifact_id = av.artifact_id
					AND avt.name NOT LIKE '%:%'
				), ARRAY []::RECORD[]) AS tags,
				av.manifest_blob_size + coalesce(sum(avp.artifact_blob_size), 0) AS size,
				`+artifactDownloadsOutExpr+`
			FROM ArtifactVersion av
			LEFT JOIN LATERAL (
				WITH RECURSIVE aggregate AS (
					SELECT avp.artifact_version_id as base_av_id,
						   avp.artifact_version_id as related_av_id,
						   avp.artifact_blob_digest,
						   avp.artifact_blob_size
					FROM ArtifactVersionPart avp
						WHERE avp.artifact_version_id = av.id
					UNION ALL
					SELECT aggregate.base_av_id, av1.id, avp.artifact_blob_digest, avp.artifact_blob_size
					FROM aggregate
					JOIN ArtifactVersion av1 ON av1.manifest_blob_digest = aggregate.artifact_blob_digest
					JOIN ArtifactVersionPart avp ON av1.id = avp.artifact_version_id
				)
				SELECT DISTINCT * FROM aggregate
			) avp ON av.id = avp.base_av_id
			LEFT JOIN ArtifactVersionPull avpl ON @isVendorUser AND avpl.artifact_version_id = avp.related_av_id
			LEFT JOIN Artifact a ON a.id = av.artifact_id
			LEFT JOIN Organization_UserAccount oua_dl
				ON oua_dl.organization_id = a.organization_id
					AND oua_dl.user_account_id = avpl.useraccount_id
			WHERE av.artifact_id = @artifactId
			AND av.name LIKE '%:%'
			AND (
				@isVendorUser
				-- only check license if there is at least one license in this organization
				OR NOT EXISTS (
					SELECT al.id
					FROM artifact a
					JOIN ArtifactLicense al ON a.organization_id = al.organization_id
					WHERE a.id = @artifactId
				)
				-- license check
				OR EXISTS (
					-- license for all versions of the artifact
					SELECT *
					FROM ArtifactLicense_Artifact ala
					INNER JOIN ArtifactLicense al ON ala.artifact_license_id = al.id
					WHERE ala.artifact_id = @artifactId AND ala.artifact_version_id IS NULL
					AND al.customer_organization_id = @customerOrgId AND (al.expires_at IS NULL OR al.expires_at > now())
				)
				OR EXISTS (
					-- or license only for specific versions or their parent versions
					WITH RECURSIVE ArtifactVersionAggregate (id, manifest_blob_digest) AS (
						SELECT avx.id, avx.manifest_blob_digest
						FROM ArtifactVersion avx
						WHERE avx.manifest_blob_digest = av.manifest_blob_digest AND avx.artifact_id = @artifactId

						UNION ALL

						SELECT DISTINCT avx.id, avx.manifest_blob_digest
						FROM ArtifactVersion avx
						JOIN ArtifactVersionPart avp ON avx.id = avp.artifact_version_id
						JOIN ArtifactVersionAggregate agg ON avp.artifact_blob_digest = agg.manifest_blob_digest
					)
					SELECT *
					FROM ArtifactVersionAggregate avagg
					INNER JOIN ArtifactLicense_Artifact ala ON ala.artifact_version_id = avagg.id
					INNER JOIN ArtifactLicense al ON ala.artifact_license_id = al.id
					WHERE al.customer_organization_id = @customerOrgId AND (al.expires_at IS NULL OR al.expires_at > now())
					AND ala.artifact_id = @artifactId
				)
			)
			AND EXISTS (
				-- only versions that have a tag
				SELECT avt.id
				FROM ArtifactVersion avt
				WHERE avt.manifest_blob_digest = av.manifest_blob_digest
				AND avt.artifact_id = av.artifact_id
				AND avt.name NOT LIKE '%:%'
			)
			GROUP BY av.id, av.created_at, av.manifest_blob_digest, a.organization_id
			ORDER BY av.created_at DESC
			`,
		pgx.NamedArgs{
			"artifactId":    artifactID,
			"customerOrgId": customerOrgID,
			"isVendorUser":  isVendorUser,
		}); err != nil {
		return nil, err
	} else if versions, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.TaggedArtifactVersion]); err != nil {
		return nil, err
	} else {
		for i, version := range versions {
			version.InferredType = types.ManifestTypeGeneric
			if strings.HasPrefix(version.ManifestContentType, "application/vnd.docker") {
				version.InferredType = types.ManifestTypeContainerImage
			} else if !manifest.MIMETypeIsMultiImage(version.ManifestContentType) && len(version.ManifestData) > 0 {
				parsedManifest, err := manifest.FromBlob(version.ManifestData, version.ManifestContentType)
				if err != nil {
					return nil, err
				}

				if strings.HasPrefix(parsedManifest.ConfigInfo().MediaType, "application/vnd.cncf.helm") ||
					slices.ContainsFunc(parsedManifest.LayerInfos(), func(layer manifest.LayerInfo) bool {
						return strings.HasPrefix(layer.MediaType, "application/vnd.cncf.helm")
					}) {
					version.InferredType = types.ManifestTypeHelmChart
				}
			}
			versions[i] = version
		}
		return versions, nil
	}
}

func GetOrCreateArtifact(ctx context.Context, orgID uuid.UUID, artifactName string) (*types.Artifact, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT `+artifactOutputExpr+`
			FROM Artifact a
			WHERE a.name = @name AND a.organization_id = @orgId`,
		pgx.NamedArgs{
			"name":  artifactName,
			"orgId": orgID,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query artifact: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Artifact]); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			artifact := &types.Artifact{Name: artifactName, OrganizationID: orgID}
			err = CreateArtifact(ctx, artifact)
			return artifact, err
		}
		return nil, fmt.Errorf("could not collect artifact: %w", err)
	} else {
		return &result, nil
	}
}

func CreateArtifact(ctx context.Context, artifact *types.Artifact) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO Artifact AS a (name, organization_id) VALUES (@name, @organizationId) RETURNING `+artifactOutputExpr,
		pgx.NamedArgs{
			"name":           artifact.Name,
			"organizationId": artifact.OrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert Artifact: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.Artifact]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*artifact = result
		return nil
	}
}

func HasAnyArtifactLicense(ctx context.Context, orgID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	var hasLicenses bool
	err := db.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM ArtifactLicense al
			WHERE al.organization_id = @orgId
		)`,
		pgx.NamedArgs{"orgId": orgID},
	).Scan(&hasLicenses)
	if err != nil {
		return false, fmt.Errorf("could not check for licenses: %w", err)
	}
	return hasLicenses, nil
}

func CheckLicenseForArtifact(
	ctx context.Context,
	orgName, name, reference string,
	customerOrganizationID uuid.UUID,
	orgID uuid.UUID,
) error {
	hasLicenses, err := HasAnyArtifactLicense(ctx, orgID)
	if err != nil {
		return err
	} else if !hasLicenses {
		return nil
	}

	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`WITH RECURSIVE ArtifactVersionAggregate (id, artifact_id, manifest_blob_digest) AS (
			SELECT av.id, av.artifact_id, av.manifest_blob_digest
				FROM Artifact a
				JOIN ArtifactVersion av ON a.id = av.artifact_id
				JOIN ArtifactVersion avx ON a.id = avx.artifact_id AND avx.manifest_blob_digest = av.manifest_blob_digest
				JOIN Organization o ON o.id = a.organization_id
				WHERE o.slug = @orgName
				AND a.name = @name
				AND (avx.name = @reference OR avx.manifest_blob_digest = @reference)
			UNION ALL
			SELECT DISTINCT av.id, av.artifact_id, av.manifest_blob_digest
				FROM ArtifactVersion av
				JOIN ArtifactVersionPart avp ON av.id = avp.artifact_version_id
				JOIN ArtifactVersionAggregate agg ON avp.artifact_blob_digest = agg.manifest_blob_digest
		)
		SELECT exists(
			SELECT *
				FROM ArtifactVersionAggregate av
				JOIN ArtifactLicense_Artifact ala
					ON av.artifact_id = ala.artifact_id
						AND (ala.artifact_version_id IS NULL OR ala.artifact_version_id = av.id)
				JOIN ArtifactLicense al ON ala.artifact_license_id = al.id
				WHERE al.customer_organization_id = @customerOrganizationId
					AND (al.expires_at IS NULL OR al.expires_at > now())
		)`,
		pgx.NamedArgs{
			"orgName":                orgName,
			"name":                   name,
			"reference":              reference,
			"customerOrganizationId": customerOrganizationID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return fmt.Errorf("could not query ArtifactVersion: %w", err)
	} else if !exists {
		return apierrors.ErrForbidden
	}
	return nil
}

func CheckLicenseForArtifactBlob(ctx context.Context, digest string,
	customerOrganizationID uuid.UUID,
	orgID uuid.UUID,
) error {
	hasLicenses, err := HasAnyArtifactLicense(ctx, orgID)
	if err != nil {
		return err
	} else if !hasLicenses {
		return nil
	}

	db := internalctx.GetDb(ctx)

	rows, err := db.Query(
		ctx,
		`WITH RECURSIVE ArtifactVersionAggregate (id, artifact_id, manifest_blob_digest) AS (
			SELECT av.id, av.artifact_id, av.manifest_blob_digest
				FROM ArtifactVersion av
				JOIN ArtifactVersionPart avp ON av.id = avp.artifact_version_id
				WHERE avp.artifact_blob_digest = @digest
			UNION ALL
			SELECT DISTINCT av.id, av.artifact_id, av.manifest_blob_digest
				FROM ArtifactVersion av
				JOIN ArtifactVersionPart avp ON av.id = avp.artifact_version_id
				JOIN ArtifactVersionAggregate agg ON avp.artifact_blob_digest = agg.manifest_blob_digest
		)
		SELECT exists(
			SELECT *
				FROM ArtifactVersionAggregate av
				JOIN ArtifactLicense_Artifact ala
					ON av.artifact_id = ala.artifact_id
						AND (ala.artifact_version_id IS NULL OR ala.artifact_version_id = av.id)
				JOIN ArtifactLicense al ON ala.artifact_license_id = al.id
				WHERE al.customer_organization_id = @customerOrganizationId
					AND (al.expires_at IS NULL OR al.expires_at > now())
		)`,
		pgx.NamedArgs{"digest": digest, "customerOrganizationId": customerOrganizationID},
	)
	if err != nil {
		return fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return fmt.Errorf("could not query ArtifactVersion: %w", err)
	} else if !exists {
		return apierrors.ErrForbidden
	}
	return nil
}

func GetArtifactVersion(ctx context.Context, orgName, name, reference string) (*types.ArtifactVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`SELECT`+artifactVersionOutputExpr+`
		FROM Artifact a
		JOIN Organization o ON o.id = a.organization_id
		LEFT JOIN ArtifactVersion v ON a.id = v.artifact_id
		WHERE o.slug = @orgName
			AND a.name = @name
			AND v.name = @reference`,
		pgx.NamedArgs{"orgName": orgName, "name": name, "reference": reference},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactVersion])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	return &result, nil
}

func CreateArtifactVersion(ctx context.Context, av *types.ArtifactVersion) error {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(
		ctx,
		`INSERT INTO ArtifactVersion AS av (
            name,
			created_by_useraccount_id,
			manifest_blob_digest,
			manifest_blob_size,
			manifest_content_type,
			manifest_data,
			artifact_id
        ) VALUES (
        	@name, @createdById, @manifestBlobDigest, @manifestBlobSize, @manifestContentType, @manifestData,
			@artifactId
        ) RETURNING *`,
		pgx.NamedArgs{
			"name":                av.Name,
			"createdById":         av.CreatedByUserAccountID,
			"manifestBlobDigest":  av.ManifestBlobDigest,
			"manifestBlobSize":    av.ManifestBlobSize,
			"manifestContentType": av.ManifestContentType,
			"manifestData":        av.ManifestData,
			"artifactId":          av.ArtifactID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not insert ArtifactVersion: %w", err)
	}
	if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactVersion]); err != nil {
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pgerrcode.UniqueViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
		return err
	} else {
		*av = result
		return nil
	}
}

func CreateArtifactVersionPart(ctx context.Context, avp *types.ArtifactVersionPart) error {
	db := internalctx.GetDb(ctx)
	if rows, err := db.Query(
		ctx,
		`INSERT INTO ArtifactVersionPart AS avp (
        	artifact_version_id, artifact_blob_digest, artifact_blob_size
        ) VALUES (@versionId, @blobDigest, @blobSize)
		ON CONFLICT (artifact_version_id, artifact_blob_digest)
			DO UPDATE SET
				artifact_version_id = @versionId,
				artifact_blob_digest = @blobDigest,
				artifact_blob_size = @blobSize
		RETURNING *`,
		pgx.NamedArgs{
			"versionId":  avp.ArtifactVersionID,
			"blobDigest": avp.ArtifactBlobDigest,
			"blobSize":   avp.ArtifactBlobSize,
		},
	); err != nil {
		return fmt.Errorf("could not insert ArtifactVersionPart: %w", err)
	} else if result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactVersionPart]); err != nil {
		return err
	} else {
		*avp = result
		return nil
	}
}

func CreateArtifactPullLogEntry(
	ctx context.Context,
	versionID,
	userID uuid.UUID,
	remoteAddress string,
	customerOrgID *uuid.UUID,
) error {
	db := internalctx.GetDb(ctx)
	remoteAddressPtr := &remoteAddress
	if remoteAddress == "" {
		remoteAddressPtr = nil
	}
	_, err := db.Exec(
		ctx,
		`INSERT INTO ArtifactVersionPull (
			artifact_version_id,
			useraccount_id,
			remote_address,
			customer_organization_id
		)
		VALUES (
			@versionId,
			@userId,
			@remoteAddress,
			@customerOrgId
		)`,
		pgx.NamedArgs{
			"versionId":     versionID,
			"userId":        userID,
			"remoteAddress": remoteAddressPtr,
			"customerOrgId": customerOrgID,
		},
	)
	if err != nil {
		return fmt.Errorf("could not create artifact pull log entry: %w", err)
	}
	return nil
}

func EnsureArtifactTagLimitForInsert(ctx context.Context, orgID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT count(av.name) + 1 < coalesce(
			o.artifact_tag_limit,
			CASE WHEN @defaultLimit > 0 THEN @defaultLimit ELSE @maxLimit END
		)
		FROM ArtifactVersion av
		JOIN Artifact a on av.artifact_id = a.id
		JOIN Organization o ON a.organization_id = o.id
		WHERE o.id = @orgId AND av.name NOT LIKE '%:%'
		GROUP BY o.id;`,
		pgx.NamedArgs{
			"orgId":        orgID,
			"defaultLimit": env.ArtifactTagsDefaultLimitPerOrg(),
			"maxLimit":     math.MaxInt32,
		},
	)
	if err != nil {
		return false, fmt.Errorf("could not check quota: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[struct{ Ok bool }])
	// If there are no rows, the organization has no tags yet, and the limit is not exceeded.
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	} else if err != nil {
		return false, fmt.Errorf("could not check quota: %w", err)
	} else {
		return result.Ok, nil
	}
}

func GetArtifactVersionPullFilterOptions(
	ctx context.Context,
	orgID uuid.UUID,
) (*types.ArtifactVersionPullFilterOptions, error) {
	db := internalctx.GetDb(ctx)
	result := &types.ArtifactVersionPullFilterOptions{}

	pullBaseJoin := ` FROM ArtifactVersionPull p
		JOIN ArtifactVersion v ON v.id = p.artifact_version_id
		JOIN Artifact a ON a.id = v.artifact_id`
	args := pgx.NamedArgs{"orgId": orgID}

	// Customer organizations
	rows, err := db.Query(ctx,
		`SELECT co.id, co.name`+pullBaseJoin+`
			JOIN CustomerOrganization co ON co.id = p.customer_organization_id
		WHERE a.organization_id = @orgId
		GROUP BY co.id, co.name
		ORDER BY co.name`, args)
	if err != nil {
		return nil, fmt.Errorf("could not query customer organizations for filter options: %w", err)
	}
	if result.CustomerOrganizations, err = pgx.CollectRows(rows, pgx.RowToStructByPos[types.FilterOption]); err != nil {
		return nil, fmt.Errorf("could not scan customer organizations for filter options: %w", err)
	}

	// User accounts
	rows, err = db.Query(ctx,
		`SELECT u.id, COALESCE(NULLIF(u.name, ''), u.email) AS name`+pullBaseJoin+`
			JOIN UserAccount u ON u.id = p.useraccount_id
		WHERE a.organization_id = @orgId
		GROUP BY u.id, u.name, u.email
		ORDER BY name`, args)
	if err != nil {
		return nil, fmt.Errorf("could not query user accounts for filter options: %w", err)
	}
	if result.UserAccounts, err = pgx.CollectRows(rows, pgx.RowToStructByPos[types.FilterOption]); err != nil {
		return nil, fmt.Errorf("could not scan user accounts for filter options: %w", err)
	}

	// Remote addresses
	rows, err = db.Query(ctx,
		`SELECT p.remote_address`+pullBaseJoin+`
		WHERE a.organization_id = @orgId
			AND p.remote_address IS NOT NULL
		GROUP BY p.remote_address
		ORDER BY p.remote_address`, args)
	if err != nil {
		return nil, fmt.Errorf("could not query remote addresses for filter options: %w", err)
	}
	if result.RemoteAddresses, err = pgx.CollectRows(rows, pgx.RowTo[string]); err != nil {
		return nil, fmt.Errorf("could not scan remote addresses for filter options: %w", err)
	}

	// Artifacts
	rows, err = db.Query(ctx,
		`SELECT a.id, a.name`+pullBaseJoin+`
		WHERE a.organization_id = @orgId
		GROUP BY a.id, a.name
		ORDER BY a.name`, args)
	if err != nil {
		return nil, fmt.Errorf("could not query artifacts for filter options: %w", err)
	}
	if result.Artifacts, err = pgx.CollectRows(rows, pgx.RowToStructByPos[types.FilterOption]); err != nil {
		return nil, fmt.Errorf("could not scan artifacts for filter options: %w", err)
	}

	return result, nil
}

func GetArtifactVersionPullVersionOptions(
	ctx context.Context,
	orgID uuid.UUID,
	artifactID uuid.UUID,
) ([]types.FilterOption, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx,
		`SELECT v.id, v.name
		FROM ArtifactVersionPull p
			JOIN ArtifactVersion v ON v.id = p.artifact_version_id
			JOIN Artifact a ON a.id = v.artifact_id
		WHERE a.organization_id = @orgId
			AND a.id = @artifactId
		GROUP BY v.id, v.name
		ORDER BY v.name`,
		pgx.NamedArgs{"orgId": orgID, "artifactId": artifactID})
	if err != nil {
		return nil, fmt.Errorf("could not query artifact version options: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.FilterOption])
	if err != nil {
		return nil, fmt.Errorf("could not scan artifact version options: %w", err)
	}
	return result, nil
}

func GetArtifactVersionPulls(
	ctx context.Context,
	filter types.ArtifactVersionPullFilter,
) ([]types.ArtifactVersionPull, error) {
	db := internalctx.GetDb(ctx)

	conditions := []string{
		"a.organization_id = @orgId",
		"p.created_at < @before",
	}
	args := pgx.NamedArgs{
		"orgId":  filter.OrgID,
		"before": filter.Before,
		"count":  filter.Count,
	}

	if !filter.After.IsZero() {
		conditions = append(conditions, "p.created_at > @after")
		args["after"] = filter.After
	}
	if filter.CustomerOrganizationID != nil {
		conditions = append(conditions, "p.customer_organization_id = @customerOrgId")
		args["customerOrgId"] = *filter.CustomerOrganizationID
	}
	if filter.UserAccountID != nil {
		conditions = append(conditions, "p.useraccount_id = @userAccountId")
		args["userAccountId"] = *filter.UserAccountID
	}
	if filter.RemoteAddress != nil {
		conditions = append(conditions, "p.remote_address = @remoteAddress")
		args["remoteAddress"] = *filter.RemoteAddress
	}
	if filter.ArtifactID != nil {
		conditions = append(conditions, "a.id = @artifactId")
		args["artifactId"] = *filter.ArtifactID
	}
	if filter.ArtifactVersionID != nil {
		conditions = append(conditions, "v.id = @artifactVersionId")
		args["artifactVersionId"] = *filter.ArtifactVersionID
	}

	query := `SELECT
			p.created_at,
			p.remote_address,
			CASE WHEN u.id IS NOT NULL THEN (` + userAccountOutputExpr + `) ELSE NULL END,
			CASE WHEN co.id IS NOT NULL THEN (` + customerOrganizationOutputExpr + `) ELSE NULL END,
			(` + artifactOutputExpr + `),
			(` + artifactVersionOutputExpr + `)
		FROM ArtifactVersionPull p
			LEFT JOIN UserAccount u ON u.id = p.useraccount_id
			LEFT JOIN CustomerOrganization co ON co.id = p.customer_organization_id
			JOIN ArtifactVersion v ON v.id = p.artifact_version_id
			JOIN Artifact a ON a.id = v.artifact_id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY p.created_at DESC
		LIMIT @count`

	rows, err := db.Query(ctx, query, args)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactVersionPulls: %w", err)
	}
	result, err := pgx.CollectRows(rows, pgx.RowToStructByPos[types.ArtifactVersionPull])
	if err != nil {
		return nil, fmt.Errorf("could not scan ArtifactVersionPulls: %w", err)
	}
	return result, nil
}

func UpdateArtifactImage(ctx context.Context, artifact *types.ArtifactWithTaggedVersion, imageID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	row := db.QueryRow(ctx,
		`UPDATE Artifact SET image_id = @imageId WHERE id = @id RETURNING image_id`,
		pgx.NamedArgs{"imageId": imageID, "id": artifact.ID},
	)
	if err := row.Scan(&artifact.ImageID); err != nil {
		return fmt.Errorf("could not save image id to artifact: %w", err)
	}
	return nil
}

func ArtifactIsReferencedInLicenses(ctx context.Context, artifactID uuid.UUID) (bool, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT count(ala.id) > 0
		FROM ArtifactLicense_Artifact ala
		WHERE ala.artifact_id = @artifactId`,
		pgx.NamedArgs{"artifactId": artifactID},
	)
	if err != nil {
		return false, err
	}
	exists, err := pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetArtifactVersionByTag retrieves an artifact version by its tag name
func GetArtifactVersionByTag(
	ctx context.Context,
	artifactID uuid.UUID,
	tagName string,
) (*types.ArtifactVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+artifactVersionOutputExpr+`
		FROM ArtifactVersion v
		WHERE v.artifact_id = @artifactId
		AND v.name = @tagName`,
		pgx.NamedArgs{
			"artifactId": artifactID,
			"tagName":    tagName,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	result, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[types.ArtifactVersion])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apierrors.ErrNotFound
		}
		return nil, fmt.Errorf("could not query ArtifactVersion: %w", err)
	}
	return &result, nil
}

// GetArtifactVersionsByDigest retrieves all artifact versions with the same manifest_blob_digest
func GetArtifactVersionsByDigest(
	ctx context.Context,
	artifactID uuid.UUID,
	digest string,
) ([]types.ArtifactVersion, error) {
	db := internalctx.GetDb(ctx)
	rows, err := db.Query(ctx, `
		SELECT `+artifactVersionOutputExpr+`
		FROM ArtifactVersion v
		WHERE v.artifact_id = @artifactId
		AND v.manifest_blob_digest = @digest
		ORDER BY v.created_at DESC`,
		pgx.NamedArgs{
			"artifactId": artifactID,
			"digest":     digest,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("could not query ArtifactVersions: %w", err)
	}
	results, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.ArtifactVersion])
	if err != nil {
		return nil, fmt.Errorf("could not collect ArtifactVersions: %w", err)
	}
	return results, nil
}

// CheckArtifactVersionDeletionForLicenses performs comprehensive license validation before deleting a tag
// It checks if the deletion would break license access by verifying:
// 1. If the SHA version is referenced in licenses
// 2. If there are other non-SHA tags pointing to the same digest
// 3. If there are all-versions licenses (without artifact_version_id)
// Returns error if deletion should be prevented due to license references
func CheckArtifactVersionDeletionForLicenses(
	ctx context.Context,
	artifactID uuid.UUID,
	version *types.ArtifactVersion,
	versionsWithSameDigest []types.ArtifactVersion,
) error {
	db := internalctx.GetDb(ctx)

	// Find the SHA version (where name = manifest_blob_digest, i.e., starts with "sha256:")
	var shaVersion *types.ArtifactVersion
	for i := range versionsWithSameDigest {
		if versionsWithSameDigest[i].Name == string(versionsWithSameDigest[i].ManifestBlobDigest) {
			shaVersion = &versionsWithSameDigest[i]
			break
		}
	}

	// If there's no SHA version, we can't have license references to it
	if shaVersion == nil {
		// Still check for all-versions licenses
		return checkAllVersionsLicense(ctx, artifactID)
	}

	// Check if the SHA version is referenced in any license
	var isReferencedCount int64
	err := db.QueryRow(ctx, `
		SELECT count(*)
		FROM ArtifactLicense_Artifact ala
		WHERE ala.artifact_version_id = @shaVersionId`,
		pgx.NamedArgs{
			"shaVersionId": shaVersion.ID,
		},
	).Scan(&isReferencedCount)
	if err != nil {
		return fmt.Errorf("could not check license references: %w", err)
	}

	// If SHA version is referenced in licenses
	if isReferencedCount > 0 {
		// Count other non-SHA tags pointing to the same digest (excluding the tag being deleted)
		otherNonSHATags := 0
		for _, v := range versionsWithSameDigest {
			// Count non-SHA tags (names that don't contain ":")
			if v.Name != version.Name && !isDigestName(v.Name) {
				otherNonSHATags++
			}
		}

		// If there are no other non-SHA tags, deletion should fail
		if otherNonSHATags == 0 {
			return apierrors.NewBadRequest(
				"cannot delete tag: the manifest digest is referenced in one or more licenses " +
					"and this is the last non-SHA tag pointing to it",
			)
		}
	}

	// Check for all-versions licenses
	return checkAllVersionsLicense(ctx, artifactID)
}

// checkAllVersionsLicense checks if there's a license referencing the artifact without any artifact_version_id
func checkAllVersionsLicense(ctx context.Context, artifactID uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	var hasAllVersionsLicense bool
	err := db.QueryRow(ctx, `
		SELECT count(*) > 0
		FROM ArtifactLicense_Artifact ala
		WHERE ala.artifact_id = @artifactId
		AND ala.artifact_version_id IS NULL`,
		pgx.NamedArgs{
			"artifactId": artifactID,
		},
	).Scan(&hasAllVersionsLicense)
	if err != nil {
		return fmt.Errorf("could not check all-versions license: %w", err)
	}

	if hasAllVersionsLicense {
		return apierrors.NewBadRequest("cannot delete tag: there is an all-versions license for this artifact")
	}

	return nil
}

// isDigestName checks if a version name is a digest (contains ":")
func isDigestName(name string) bool {
	return len(name) > 0 && strings.Contains(name, ":")
}

func DeleteArtifactWithID(ctx context.Context, id uuid.UUID) error {
	db := internalctx.GetDb(ctx)
	cmd, err := db.Exec(ctx, `DELETE FROM Artifact WHERE id = @id`, pgx.NamedArgs{"id": id})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete Artifact: %w", err)
	}

	return nil
}

func IsLastTagOfArtifact(ctx context.Context, artifactID uuid.UUID, tagName string) (bool, error) {
	db := internalctx.GetDb(ctx)

	// Count all non-SHA tags for this artifact
	// Tags are ArtifactVersion records where name does NOT contain a colon
	var tagCount int64
	err := db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ArtifactVersion
		WHERE artifact_id = @artifactId
		AND name NOT LIKE '%:%'`,
		pgx.NamedArgs{
			"artifactId": artifactID,
		}).Scan(&tagCount)
	if err != nil {
		return false, fmt.Errorf("could not count tags: %w", err)
	}

	// If there is only 1 tag remaining, and we're trying to delete it, prevent deletion
	return tagCount == 1, nil
}

func DeleteArtifactVersion(ctx context.Context, artifactID uuid.UUID, tagName string) error {
	db := internalctx.GetDb(ctx)

	// Delete only the tag, not the version SHA
	// Tags are ArtifactVersion records where name does NOT contain a colon
	cmd, err := db.Exec(ctx, `
		DELETE FROM ArtifactVersion
		WHERE artifact_id = @artifactId
		AND name = @tagName
		AND name NOT LIKE '%:%'`,
		pgx.NamedArgs{
			"artifactId": artifactID,
			"tagName":    tagName,
		})
	if err != nil {
		if pgerr := (*pgconn.PgError)(nil); errors.As(err, &pgerr) && pgerr.Code == pgerrcode.ForeignKeyViolation {
			err = fmt.Errorf("%w: %w", apierrors.ErrConflict, err)
		}
	} else if cmd.RowsAffected() == 0 {
		err = apierrors.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("could not delete tag: %w", err)
	}

	return nil
}
