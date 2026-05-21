package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type ApplicationVersion struct {
	// unfortunately Base nested type doesn't work when ApplicationVersion is a nested row in an SQL query
	ID            uuid.UUID      `db:"id" json:"id"`
	CreatedAt     time.Time      `db:"created_at" json:"createdAt"`
	ArchivedAt    *time.Time     `db:"archived_at" json:"archivedAt,omitempty"`
	Name          string         `db:"name" json:"name"`
	LinkTemplate  string         `db:"link_template" json:"linkTemplate"`
	ApplicationID uuid.UUID      `db:"application_id" json:"applicationId"`
	ChartType     *HelmChartType `db:"chart_type" json:"chartType,omitempty"`
	ChartName     *string        `db:"chart_name" json:"chartName,omitempty"`
	ChartUrl      *string        `db:"chart_url" json:"chartUrl,omitempty"`
	ChartVersion  *string        `db:"chart_version" json:"chartVersion,omitempty"`

	// awful but relevant: the following must be defined after the ChartType, because somehow order matters
	// for pgx at collecting the subrows (relevant at getting application + list of its versions with these
	// array aggregations) – long term it should probably be refactored because this is such a pitfall
	// https://github.com/jackc/pgx/issues/1585#issuecomment-1528810634
	ValuesFileData   []byte `db:"values_file_data" json:"-"`
	TemplateFileData []byte `db:"template_file_data" json:"-"`
	ComposeFileData  []byte `db:"compose_file_data" json:"-"`

	Resources []ApplicationVersionResource `db:"-" json:"resources,omitempty"`
}

func (av ApplicationVersion) ParsedValuesFile() (result map[string]any, err error) {
	if av.ValuesFileData != nil {
		if err = yaml.Unmarshal(av.ValuesFileData, &result); err != nil {
			err = fmt.Errorf("cannot parse ApplicationVersion values file: %w", err)
		}
	}
	return result, err
}

func (av ApplicationVersion) ParsedTemplateFile() (result map[string]any, err error) {
	if av.TemplateFileData != nil {
		if err = yaml.Unmarshal(av.TemplateFileData, &result); err != nil {
			err = fmt.Errorf("cannot parse ApplicationVersion values template: %w", err)
		}
	}
	return result, err
}

func (av ApplicationVersion) ParsedComposeFile() (result map[string]any, err error) {
	if av.ComposeFileData != nil {
		if err = yaml.Unmarshal(av.ComposeFileData, &result); err != nil {
			err = fmt.Errorf("cannot parse ApplicationVersion compose file: %w", err)
		}
	}
	return result, err
}

func (av ApplicationVersion) Validate(deplType DeploymentType) error {
	switch deplType {
	case DeploymentTypeDocker:
		if av.ComposeFileData == nil {
			return errors.New("missing compose file")
		} else if av.ChartType != nil || av.ChartName != nil || av.ChartUrl != nil || av.ChartVersion != nil ||
			av.ValuesFileData != nil {
			return errors.New("unexpected kubernetes specifics in docker application")
		}
	case DeploymentTypeKubernetes:
		if av.ChartType == nil || *av.ChartType == "" ||
			av.ChartUrl == nil || *av.ChartUrl == "" ||
			av.ChartVersion == nil || *av.ChartVersion == "" {
			return errors.New("not all of chart type, url and version are given")
		} else if *av.ChartType == HelmChartTypeRepository && (av.ChartName == nil || *av.ChartName == "") {
			return errors.New("missing chart name")
		} else if av.ComposeFileData != nil {
			return errors.New("unexpected docker file in kubernetes application")
		}
	}
	return nil
}
