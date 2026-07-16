---
title: Kubernetes
description: Deploy Distr in your Kubernetes cluster using our Helm chart with built-in PostgreSQL and RustFS storage.
sidebar:
  label: Kubernetes
  order: 3
---

Distr is available as a [Helm chart](/glossary/helm-chart/) distributed via ghcr.io.
To install Distr in [Kubernetes](/glossary/kubernetes/), simply run:

```shell
helm upgrade --install --wait --namespace distr --create-namespace \
  distr oci://ghcr.io/distr-sh/charts/distr \
  --set postgresql.enabled=true --set rustfs.enabled=true
```

For a quick testing setup, you don't have to modify the values. However, if you intend to use distr in production, please revisit all available configuration values and adapt them accordingly.
You can find them in the reference [values.yaml](https://artifacthub.io/packages/helm/distr/distr?modal=values) file.

## Log storage (Loki)

The chart includes a bundled [Grafana Loki](https://grafana.com/oss/loki/) instance (enabled by default) that stores deployment and deployment target logs with a 30-day retention.
By default it persists its data in the in-cluster RustFS object storage, so the quick start above works out of the box.

If you use an external S3-compatible object storage instead of the bundled RustFS, point `loki.loki.storage.s3` (and the bucket-provisioning init container under `loki.singleBinary.initContainers`) at it.
To use an externally managed Loki instance, set `loki.enabled=false` and configure `LOKI_URL` (and optionally `LOKI_BEARER_TOKEN` or `LOKI_BASIC_AUTH_*`) in `hub.env`.
