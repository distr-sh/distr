---
title: Prometheus Integration
description: Monitor your Distr instance with Prometheus by scraping built-in metrics for deployments, organizations, and Go runtime data.
sidebar:
  order: 6
---

Distr Hub exposes a Prometheus-compatible metrics endpoint that can be scraped by any Prometheus-compatible monitoring system.

## Configuration

The metrics endpoint is configured via environment variables:

| Variable               | Default | Description                                                                        |
| ---------------------- | ------- | ---------------------------------------------------------------------------------- |
| `METRICS_ENABLED`      | `false` | Enable the metrics HTTP server                                                     |
| `METRICS_ADDR`         | `:3000` | Address and port the metrics server listens on                                     |
| `METRICS_BEARER_TOKEN` | —       | If set, requires an `Authorization: Bearer <token>` header on every scrape request |

### Docker Compose

An example configuration can be found in
[`github.com/distr-sh/distr/deploy/docker`](https://github.com/distr-sh/distr/blob/main/deploy/docker/.env):

```dotenv
METRICS_ENABLED=true
# METRICS_ADDR=:3000
# METRICS_BEARER_TOKEN=my-secret-token
```

### Kubernetes (Helm)

When deploying with the Distr Helm chart, the same environment variables can be set via `hub.env` in your `values.yaml`.
The metrics server port is configured separately via `service.metricsPort`.

An example configuration can be found in
[`github.com/distr-sh/distr/deploy/charts/distr`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/values.yaml):

Once enabled, the metrics are available at `GET /metrics` on the configured address (e.g. `http://localhost:3000/metrics`).

## Securing the metrics endpoint

If `METRICS_BEARER_TOKEN` is set, every request to the metrics endpoint must include the token in the `Authorization` header.
Unauthenticated requests will receive a `401 Unauthorized` response.

Example Prometheus scrape configuration with authentication:

```yaml
scrape_configs:
  - job_name: distr
    scheme: http
    static_configs:
      - targets: ['distr-hub:3000']
    authorization:
      type: Bearer
      credentials: my-secret-token
```

## Available metrics

### Distr-specific metrics

These custom metrics are exposed under the `distr` namespace and provide insight into the state of your Distr instance.

| Metric                                      | Type  | Labels                                                                                                                    | Description                                                                                             |
| ------------------------------------------- | ----- | ------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| `distr_organizations_total`                 | Gauge | —                                                                                                                         | Current number of organizations                                                                         |
| `distr_deployment_status`                   | Gauge | `organization`, `customerorganization`, `deploymenttarget`, `deploymentid`, `application`, `applicationversion`, `status` | Whether a deployment is in a given status (`1`) or not (`0`). One time series per deployment per status |
| `distr_deployment_status_timestamp_seconds` | Gauge | `organization`, `customerorganization`, `deploymenttarget`, `deploymentid`, `application`, `applicationversion`           | Unix timestamp of the most recent status update for a deployment                                        |

The `distr_deployment_status` metric uses a one-hot encoding pattern: for each deployment, there is one time series per possible status value (`healthy`, `running`, `progressing`, `error`), where exactly one is set to `1` and the rest to `0`.

### Go runtime and process metrics

In addition to the Distr-specific metrics, the endpoint exposes standard Go runtime and OS process metrics:

- **Go runtime** (`go_*`): goroutine count, GC duration, memory allocation, heap usage, GC configuration
- **Process** (`process_*`): CPU time, memory usage (resident and virtual), open file descriptors, process start time

## Use cases

### Custom alerting for customer deployments

Distr includes built-in [alerts](/docs/agents/alerts/) that notify users by email when deployments become unhealthy or resource usage exceeds thresholds.
For more advanced alerting workflows, the Prometheus metrics endpoint can be combined with [Alertmanager](https://github.com/prometheus/alertmanager) or [Grafana Alerts](https://grafana.com/docs/grafana/latest/alerting/) to build custom error reporting independently from the integrated Distr alerts.

For example, you can define Prometheus alerting rules based on `distr_deployment_status` to trigger notifications via Slack, PagerDuty, or other channels when a customer deployment enters an error state, or to escalate issues that remain unresolved after a certain period.
