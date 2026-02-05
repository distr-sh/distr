package svc

import (
	"github.com/distr-sh/distr/internal/cleanup"
	"github.com/distr-sh/distr/internal/env"
	"github.com/distr-sh/distr/internal/jobs"
	"github.com/distr-sh/distr/internal/notification"
)

func (r *Registry) GetJobsScheduler() *jobs.Scheduler {
	return r.jobsScheduler
}

func (r *Registry) createJobsScheduler() (*jobs.Scheduler, error) {
	scheduler, err := jobs.NewScheduler(r.GetLogger(), r.GetDbPool(), r.GetMailer(), r.GetTracers().Always())
	if err != nil {
		return nil, err
	}

	if cron := env.CleanupDeploymenRevisionStatusCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentRevisionStatusCleanup",
				cleanup.RunDeploymentRevisionStatusCleanup,
				env.CleanupDeploymenRevisionStatusTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.CleanupDeploymenTargetStatusCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentTargetStatusCleanup",
				cleanup.RunDeploymentTargetStatusCleanup,
				env.CleanupDeploymenTargetStatusTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.CleanupDeploymentTargetMetricsCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentTargetMetricsCleanup",
				cleanup.RunDeploymentTargetMetricsCleanup,
				env.CleanupDeploymentTargetMetricsTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.CleanupDeploymentTargetLogRecordCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentTargetLogRecordCleanup",
				cleanup.RunDeploymentTargetLogRecordCleanup,
				env.CleanupDeploymentTargetLogRecordTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.CleanupDeploymentLogRecordCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentLogRecordCleanup",
				cleanup.RunDeploymentLogRecordCleanup,
				env.CleanupDeploymentLogRecordTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.CleanupOIDCStateCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob("OIDCStateCleanup", cleanup.RunOIDCStateCleanup, env.CleanupOIDCStateCronTimeout()),
		)
		if err != nil {
			return nil, err
		}
	}

	if cron := env.DeploymentStatusNotificationCron(); cron != nil {
		err = scheduler.RegisterCronJob(
			*cron,
			jobs.NewJob(
				"DeploymentStatusNotification",
				notification.RunDeploymentStatusNotifications,
				env.DeploymentStatusNotificationTimeout(),
			),
		)
		if err != nil {
			return nil, err
		}
	}

	return scheduler, nil
}
