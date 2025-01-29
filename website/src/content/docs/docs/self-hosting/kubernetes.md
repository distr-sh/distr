---
title: Kubernetes
description: Self-hosting Distr with Kubernetes
sidebar:
  order: 2
---

Distr is available as a Helm chart distributed via ghcr.io.
To install Distr in Kubernetes, simply run:

```shell
helm upgrade --install --wait --namespace distr --create-namespace \
  distr oci://ghcr.io/glasskube/charts/distr --version 1.0.0 \
  --set postgresql.enabled=true
```

For all available configuration values, please consult the reference
[values.yaml](https://github.com/glasskube/distr/blob/main/deploy/charts/distr/values.yaml) file.
