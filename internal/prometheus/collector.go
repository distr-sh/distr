package prometheus

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/distr-sh/distr/internal/types"
	"github.com/distr-sh/distr/internal/util"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "distr"

type DistrCollector struct {
	organizationsTotal              prometheus.Gauge
	deploymentTargetStatusTimestamp *prometheus.GaugeVec
	deploymentStatus                *prometheus.GaugeVec
	deploymentStatusTimestamp       *prometheus.GaugeVec
}

var _ prometheus.Collector = (*DistrCollector)(nil)

// Collect implements [prometheus.Collector].
func (d *DistrCollector) Collect(c chan<- prometheus.Metric) {
	d.organizationsTotal.Collect(c)
	d.deploymentTargetStatusTimestamp.Collect(c)
	d.deploymentStatus.Collect(c)
	d.deploymentStatusTimestamp.Collect(c)
}

// Describe implements [prometheus.Collector].
func (d *DistrCollector) Describe(c chan<- *prometheus.Desc) {
	d.organizationsTotal.Describe(c)
	d.deploymentTargetStatusTimestamp.Describe(c)
	d.deploymentStatus.Describe(c)
	d.deploymentStatusTimestamp.Describe(c)
}

type InitDataSource interface {
	OrganizationsTotal(context.Context) (int64, error)
	DeploymentTargetStatus(context.Context) ([]types.DeploymentTargetStatusMetricsItem, error)
	DeploymentStatus(context.Context) ([]types.DeploymentStatusMetricsItem, error)
}

func (c *DistrCollector) Initialize(ctx context.Context, src InitDataSource) (retErr error) {
	if count, err := src.OrganizationsTotal(ctx); err != nil {
		retErr = errors.Join(retErr, err)
	} else {
		c.RecordOrganizationsTotal(count)
	}

	if metrics, err := src.DeploymentTargetStatus(ctx); err != nil {
		retErr = errors.Join(retErr, err)
	} else {
		for _, m := range metrics {
			c.HandleDeploymentTargetStatus(m)
		}
	}

	if metrics, err := src.DeploymentStatus(ctx); err != nil {
		retErr = errors.Join(retErr, err)
	} else {
		for _, m := range metrics {
			c.HandleDeploymentStatus(m)
		}
	}

	return
}

func (d *DistrCollector) RecordOrganizationsTotal(value int64) {
	d.organizationsTotal.Set(float64(value))
}

func (d *DistrCollector) IncOrganizationsTotal() {
	d.organizationsTotal.Inc()
}

func (d *DistrCollector) DecOrganizationsTotal() {
	d.organizationsTotal.Dec()
}

type DeploymentTargetStatusLabels struct {
	OrganizationName         string
	CustomerOrganizationName *string
	DeploymentTargetName     string
}

func (l DeploymentTargetStatusLabels) Values() []string {
	return []string{
		l.OrganizationName,
		util.PtrDerefOrDefault(l.CustomerOrganizationName),
		l.DeploymentTargetName,
	}
}

func (d *DistrCollector) RecordDeploymentTargetStatus(l DeploymentTargetStatusLabels, t *time.Time) {
	var v float64
	if t != nil {
		v = float64(t.Unix())
	}
	d.deploymentTargetStatusTimestamp.WithLabelValues(l.Values()...).Set(float64(v))
}

func (c *DistrCollector) HandleDeploymentTargetStatus(item types.DeploymentTargetStatusMetricsItem) {
	c.RecordDeploymentTargetStatus(
		DeploymentTargetStatusLabels{
			OrganizationName:         item.OrganizationName,
			CustomerOrganizationName: item.CustomerOrganizationName,
			DeploymentTargetName:     item.DeploymentTargetName,
		},
		item.DeploymentTargetStatusTimestamp,
	)
}

type DeploymentStatusLabels struct {
	OrganizationName         string
	CustomerOrganizationName *string
	DeploymentTargetName     string
	DeploymentID             uuid.UUID
	ApplicationName          string
	ApplicationVersionName   string
}

func (l DeploymentStatusLabels) Values() []string {
	return []string{
		l.OrganizationName,
		util.PtrDerefOrDefault(l.CustomerOrganizationName),
		l.DeploymentTargetName,
		l.DeploymentID.String(),
		l.ApplicationName,
		l.ApplicationVersionName,
	}
}

func (d *DistrCollector) RecordDeploymentStatus(l DeploymentStatusLabels, t *time.Time, s *types.DeploymentStatusType) {
	var v float64
	if t != nil {
		v = float64(t.Unix())
	}
	d.deploymenStatusTimestamp.WithLabelValues(l.Values()...).Set(v)

	for _, s1 := range types.AllDeploymentStatusTypes {
		var v float64
		if s != nil && *s == s1 {
			v = 1
		}
		d.deploymentStatus.WithLabelValues(append(l.Values(), string(s1))...).Set(v)
	}
}

func (c *DistrCollector) HandleDeploymentStatus(item types.DeploymentStatusMetricsItem) {
	c.RecordDeploymentStatus(
		DeploymentStatusLabels{
			OrganizationName:         item.OrganizationName,
			CustomerOrganizationName: item.CustomerOrganizationName,
			DeploymentTargetName:     item.DeploymentTargetName,
			DeploymentID:             item.DeploymentID,
			ApplicationName:          item.ApplicationName,
			ApplicationVersionName:   item.ApplicationVersionName,
		},
		item.DeploymentStatusTimestamp,
		item.DeploymentStatusType,
	)
}

func NewDistrCollector() *DistrCollector {
	c := &DistrCollector{}

	c.organizationsTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{Namespace: namespace, Name: "organizations_total"},
	)

	c.deploymentTargetStatusTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "deployment_target_status_timestamp_seconds",
			Help:      "Timestamp of latest deployment target status",
		},
		[]string{"organization", "customerorganization", "deploymenttarget"},
	)

	deploymentStatusLabels := []string{
		"organization", "customerorganization", "deploymenttarget", "deploymentid", "application",
		"applicationversion",
	}

	c.deploymenStatusTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "deployment_status_timestamp_seconds",
			Help:      "Timestamp of latest deployment status",
		},
		deploymentStatusLabels,
	)

	c.deploymentStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "deployment_status",
			Help:      "Whether the deployment is in a certain status",
		},
		slices.Concat(deploymentStatusLabels, []string{"status"}),
	)

	return c
}
