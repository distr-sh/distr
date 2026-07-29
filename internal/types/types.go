package types

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opencontainers/go-digest"
)

type UserRole string

const (
	UserRoleReadOnly  UserRole = "read_only"
	UserRoleReadWrite UserRole = "read_write"
	UserRoleAdmin     UserRole = "admin"
)

func ParseUserRole(value string) (UserRole, error) {
	switch value {
	case string(UserRoleReadOnly):
		return UserRoleReadOnly, nil
	case string(UserRoleReadWrite):
		return UserRoleReadWrite, nil
	case string(UserRoleAdmin):
		return UserRoleAdmin, nil
	default:
		return "", errors.New("invalid user role")
	}
}

// Rank orders the role hierarchy: admin > read_write > read_only. It panics
// for unknown roles — every role entering the codebase is validated via
// ParseUserRole / UnmarshalJSON, so an invalid value at this point is a bug.
func (r UserRole) Rank() int {
	switch r {
	case UserRoleReadOnly:
		return 0
	case UserRoleReadWrite:
		return 1
	case UserRoleAdmin:
		return 2
	default:
		panic(fmt.Sprintf("invalid user role: %q", string(r)))
	}
}

// GreaterThan reports whether r is more privileged than other.
func (r UserRole) GreaterThan(other UserRole) bool {
	return r.Rank() > other.Rank()
}

func (ref *UserRole) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	} else if userRole, err := ParseUserRole(value); err != nil {
		return err
	} else {
		*ref = userRole
		return nil
	}
}

type OrderDirection string

const (
	OrderDirectionAsc  OrderDirection = "ASC"
	OrderDirectionDesc OrderDirection = "DESC"
)

// EffectiveOrderDirection determines the SQL ORDER BY direction for log queries.
// In live mode or with only a "before" filter, we want the latest logs first (DESC).
// ASC is only applied when an "after" filter is set, meaning the user wants to paginate forward from a specific point.
func EffectiveOrderDirection(order OrderDirection, hasAfter bool) OrderDirection {
	if order == OrderDirectionAsc && hasAfter {
		return OrderDirectionAsc
	}
	return OrderDirectionDesc
}

type SubscriptionType string

func (st SubscriptionType) IsPro() bool {
	return st == SubscriptionTypeTrial ||
		st == SubscriptionTypePro ||
		st == SubscriptionTypeBusiness ||
		st == SubscriptionTypeEnterprise
}

const (
	SubscriptionTypeCommunity  SubscriptionType = "community"
	SubscriptionTypePro        SubscriptionType = "pro"
	SubscriptionTypeBusiness   SubscriptionType = "business"
	SubscriptionTypeEnterprise SubscriptionType = "enterprise"
	SubscriptionTypeTrial      SubscriptionType = "trial"
)

var NonProSubscriptionTypes = []SubscriptionType{
	SubscriptionTypeCommunity,
}

func AllSubscriptionTypes() []SubscriptionType {
	return []SubscriptionType{
		SubscriptionTypeCommunity,
		SubscriptionTypePro,
		SubscriptionTypeBusiness,
		SubscriptionTypeEnterprise,
		SubscriptionTypeTrial,
	}
}

type Feature string

const (
	FeatureLicensing              Feature = "licensing"
	FeaturePrePostScripts         Feature = "pre_post_scripts"
	FeatureArtifactVersionMutable Feature = "artifact_version_mutable"
	FeatureVendorBilling          Feature = "vendor_billing"
	FeatureDeploymentLogsAfter    Feature = "deployment_logs_after"
	FeaturePartnerManagement      Feature = "partner_management"
	FeatureCustomDomains          Feature = "custom_domains"
)

// ProFeatures is the set of features granted to organizations with a paid (pro) subscription.
var ProFeatures = []Feature{
	FeatureLicensing,
}

// FeaturesForSubscriptionType returns the features granted by a subscription type.
// Subscription reconciliation only ever adds these features, it never removes any:
// manually granted features (e.g. vendor_billing) must survive plan changes, and
// community organizations are stripped of features by ReconcileEditionFeatures instead.
func FeaturesForSubscriptionType(st SubscriptionType) []Feature {
	switch st {
	case SubscriptionTypeCommunity:
		return []Feature{}
	case SubscriptionTypeTrial, SubscriptionTypePro, SubscriptionTypeEnterprise:
		return []Feature{FeatureLicensing}
	case SubscriptionTypeBusiness:
		return []Feature{FeatureLicensing, FeaturePartnerManagement, FeatureCustomDomains}
	default:
		return []Feature{}
	}
}

type DeploymentStatusType string

const (
	DeploymentStatusTypeHealthy     DeploymentStatusType = "healthy"
	DeploymentStatusTypeRunning     DeploymentStatusType = "running"
	DeploymentStatusTypeProgressing DeploymentStatusType = "progressing"
	DeploymentStatusTypeError       DeploymentStatusType = "error"
)

func AllDeploymentStatusTypes() []DeploymentStatusType {
	return []DeploymentStatusType{
		DeploymentStatusTypeHealthy,
		DeploymentStatusTypeRunning,
		DeploymentStatusTypeProgressing,
		DeploymentStatusTypeError,
	}
}

var ErrInvalidDeploymentStatusType = errors.New("invalid deployment status type")

func ParseDeploymentStatusType(status string) (DeploymentStatusType, error) {
	switch status {
	case string(DeploymentStatusTypeHealthy):
		return DeploymentStatusTypeHealthy, nil
	case string(DeploymentStatusTypeRunning), "ok":
		return DeploymentStatusTypeRunning, nil
	case string(DeploymentStatusTypeProgressing):
		return DeploymentStatusTypeProgressing, nil
	case string(DeploymentStatusTypeError):
		return DeploymentStatusTypeError, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidDeploymentStatusType, status)
	}
}

func (ref *DeploymentStatusType) UnmarshalJSON(data []byte) error {
	var statusStr string
	if err := json.Unmarshal(data, &statusStr); err != nil {
		return err
	} else if status, err := ParseDeploymentStatusType(statusStr); err != nil {
		return err
	} else {
		*ref = status
		return nil
	}
}

type DeploymentType string

const (
	DeploymentTypeDocker     DeploymentType = "docker"
	DeploymentTypeKubernetes DeploymentType = "kubernetes"
)

var ErrInvalidDeploymentType = errors.New("invalid deployment type")

func ParseDeploymentType(value string) (DeploymentType, error) {
	switch value {
	case string(DeploymentTypeDocker):
		return DeploymentTypeDocker, nil
	case string(DeploymentTypeKubernetes):
		return DeploymentTypeKubernetes, nil
	default:
		return "", fmt.Errorf("%w: %v", ErrInvalidDeploymentType, value)
	}
}

func (ref *DeploymentType) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	} else if deploymentType, err := ParseDeploymentType(value); err != nil {
		return err
	} else {
		*ref = deploymentType
		return nil
	}
}

type (
	HelmChartType         string
	DeploymentTargetScope string
	DockerType            string
	Tutorial              string
	FileScope             string
	SubscriptionPeriod    string
)

const (
	HelmChartTypeRepository HelmChartType = "repository"
	HelmChartTypeOCI        HelmChartType = "oci"

	DockerTypeCompose DockerType = "compose"
	DockerTypeSwarm   DockerType = "swarm"

	DeploymentTargetScopeCluster   DeploymentTargetScope = "cluster"
	DeploymentTargetScopeNamespace DeploymentTargetScope = "namespace"

	TutorialBranding      Tutorial  = "branding"
	TutorialAgents        Tutorial  = "agents"
	TutorialRegistry      Tutorial  = "registry"
	TutorialUsers         Tutorial  = "users"
	FileScopePlatform     FileScope = "platform"
	FileScopeOrganization FileScope = "organization"

	SubscriptionPeriodMonthly SubscriptionPeriod = "monthly"
	SubscriptionPeriodYearly  SubscriptionPeriod = "yearly"
)

type Base struct {
	ID        uuid.UUID `db:"id" json:"id"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
}

type Image struct {
	Image            []byte  `db:"image" json:"image"`
	ImageFileName    *string `db:"image_file_name" json:"imageFileName"`
	ImageContentType *string `db:"image_content_type" json:"imageContentType"`
}

type Digest digest.Digest

var (
	_ sql.Scanner       = util.PtrTo(Digest(""))
	_ pgtype.TextValuer = util.PtrTo(Digest(""))
)

func (target *Digest) Scan(src any) error {
	if srcStr, ok := src.(string); !ok {
		return errors.New("src must be a string")
	} else if h, err := digest.Parse(srcStr); err != nil {
		return err
	} else {
		*target = Digest(h)
		return nil
	}
}

// TextValue implements pgtype.TextValuer.
func (src Digest) TextValue() (pgtype.Text, error) {
	return pgtype.Text{String: string(src), Valid: true}, nil
}

func (h Digest) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(h))
}

type Duration time.Duration

func (d Duration) String() string {
	return time.Duration(d).String()
}

func (d Duration) TextValue() (pgtype.Text, error) {
	return pgtype.Text{String: d.String(), Valid: true}, nil
}

func (d *Duration) Scan(src any) error {
	if srcStr, ok := src.(string); !ok {
		return errors.New("src must be a string")
	} else if h, err := time.ParseDuration(srcStr); err != nil {
		return err
	} else {
		*d = Duration(h)
		return nil
	}
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	} else if p, err := time.ParseDuration(s); err != nil {
		return err
	} else {
		*d = Duration(p)
		return nil
	}
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}
