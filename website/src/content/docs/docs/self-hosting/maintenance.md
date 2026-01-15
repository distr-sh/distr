---
title: Maintenance Jobs
description: Configure automated cleanup jobs to prune metrics and statuses from your Distr database.
slug: docs/self-hosting/maintenance
sidebar:
  order: 5
---

## Database Cleanup Jobs

Distr includes built-in cli tasks for database pruning to prevent unbounded growth of metrics and status entries.

The cleanup jobs can be executed manually.

```shell
distr cleanup DeploymentLogRecord
```

For production deployments we recommend scheduling these jobs automatically, either using the built-in job scheduler for single instance deployments
or using Kubernetes CronJobs for high-availability deployments.

### Automated job scheduling for Single Instance deployments

If you only have one instance of Distr running (e.g., using Docker Compose), you can use the integrated job scheduling.

The internal scheduling can be configured via environment variables.

An example configuration file can be found on
[`github.com/distr-sh/distr/deploy/docker`](https://github.com/distr-sh/distr/blob/main/deploy/docker/.env):

```dotenv
# Cron interval for cleaning deployment revision statuses older than STATUS_ENTRIES_MAX_AGE
CLEANUP_DEPLOYMENT_REVISION_STATUS_CRON="*/5 * * * *"
# Cron interval for cleaning deployment target statuses older than STATUS_ENTRIES_MAX_AGE
CLEANUP_DEPLOYMENT_TARGET_STATUS_CRON="*/5 * * * *"
# Cron interval for cleaning metrics older than METRICS_ENTRIES_MAX_AGE
CLEANUP_DEPLOYMENT_TARGET_METRICS_CRON="*/5 * * * *"
# Cron interval for cleaning log entries older than LOG_RECORD_ENTRIES_MAX_COUNT
CLEANUP_DEPLOYMENT_LOG_RECORD_CRON="*/5 * * * *"
```

If these variables are not set, no cleanup jobs are scheduled.

### Automated job scheduling for High-Availability deployments

For high-availability deployments with multiple instances of Distr (e.g., using Kubernetes), the built-in job scheduling
is not suitable, as it would lead to multiple instances trying to perform the same cleanup tasks concurrently.

Therefore, we recommend using [CronJobs](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/) in Kubernetes to handle the cleanup tasks.
But the concept can apply to any HA setup where the cleanup jobs are triggered by

These jobs can also be configured via our Helm Chart.

An example configuration can be found in
[`github.com/distr-sh/distr/deploy/charts/distr`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/values.yaml):

```yaml
cronJobs:
  - name: deployment-log-record-cleanup
    labels:
      distr.sh/job: deployment-log-record-cleanup
    args: [cleanup, DeploymentLogRecord]
  - name: deployment-revision-status-cleanup
    labels:
      distr.sh/job: deployment-revision-status-cleanup
    args: [cleanup, DeploymentRevisionStatus]
  - name: deployment-target-metrics-cleanup
    labels:
      distr.sh/job: deployment-target-metrics-cleanup
    args: [cleanup, DeploymentTargetMetrics]
  - name: deployment-target-status-cleanup
    labels:
      distr.sh/job: deployment-target-status-cleanup
    args: [cleanup, DeploymentTargetStatus]
```
