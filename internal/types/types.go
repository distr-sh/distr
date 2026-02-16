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

type SubscriptionType string

func (st SubscriptionType) IsPro() bool {
	return st == SubscriptionTypeTrial || st == SubscriptionTypePro || st == SubscriptionTypeEnterprise
}

const (
	SubscriptionTypeCommunity  SubscriptionType = "community"
	SubscriptionTypeStarter    SubscriptionType = "starter"
	SubscriptionTypePro        SubscriptionType = "pro"
	SubscriptionTypeEnterprise SubscriptionType = "enterprise"
	SubscriptionTypeTrial      SubscriptionType = "trial"
)

var NonProSubscriptionTypes = []SubscriptionType{
	SubscriptionTypeCommunity,
	SubscriptionTypeStarter,
}

var AllSubscriptionTypes = []SubscriptionType{
	SubscriptionTypeCommunity,
	SubscriptionTypeStarter,
	SubscriptionTypePro,
	SubscriptionTypeEnterprise,
	SubscriptionTypeTrial,
}

type Feature string

const (
	FeatureLicensing              Feature = "licensing"
	FeaturePrePostScripts         Feature = "pre_post_scripts"
	FeatureArtifactVersionMutable Feature = "artifact_version_mutable"
)

type DeploymentStatusType string

const (
	DeploymentStatusTypeHealthy     DeploymentStatusType = "healthy"
	DeploymentStatusTypeRunning     DeploymentStatusType = "running"
	DeploymentStatusTypeProgressing DeploymentStatusType = "progressing"
	DeploymentStatusTypeError       DeploymentStatusType = "error"
)

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

type (
	DeploymentType        string
	HelmChartType         string
	DeploymentTargetScope string
	DockerType            string
	Tutorial              string
	FileScope             string
	SubscriptionPeriod    string
)

const (
	DeploymentTypeDocker     DeploymentType = "docker"
	DeploymentTypeKubernetes DeploymentType = "kubernetes"

	HelmChartTypeRepository HelmChartType = "repository"
	HelmChartTypeOCI        HelmChartType = "oci"

	DockerTypeCompose DockerType = "compose"
	DockerTypeSwarm   DockerType = "swarm"

	DeploymentTargetScopeCluster   DeploymentTargetScope = "cluster"
	DeploymentTargetScopeNamespace DeploymentTargetScope = "namespace"

	TutorialBranding      Tutorial  = "branding"
	TutorialAgents        Tutorial  = "agents"
	TutorialRegistry      Tutorial  = "registry"
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
