---
title: Kubernetes
description: Deploy Distr in your Kubernetes cluster using our Helm chart with built-in PostgreSQL and MinIO storage.
sidebar:
  order: 2
---

Distr is available as a [Helm chart](/glossary/helm-chart/) distributed via ghcr.io.
To install Distr in [Kubernetes](/glossary/kubernetes/), simply run:

```shell
helm upgrade --install --wait --namespace distr --create-namespace \
  distr oci://ghcr.io/distr-sh/charts/distr \
  --set postgresql.enabled=true --set minio.enabled=true
```

For a quick testing setup, you don't have to modify the values. However, if you intend to use distr in production, please revisit all available configuration values and adapt them accordingly.
You can find them in the reference [values.yaml](https://artifacthub.io/packages/helm/distr/distr?modal=values) file.
