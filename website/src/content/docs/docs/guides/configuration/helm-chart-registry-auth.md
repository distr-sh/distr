---
title: Configuring a Helm Chart to Support Distr Artifacts
description: Configure your Helm charts to support authenticated image pulls from the Distr registry by correctly using imagePullSecrets which enables agent-based credential injection.
slug: docs/guides/helm-chart-registry-auth
sidebar:
  order: 6
---

Distr is an [OCI artifact registry](/glossary/oci-container-artifact-registry/) that is compliant with the OCI distribution specification.
As such, it can be used to host container images, [Helm charts](/glossary/helm-chart/) and many other types of artifact.
Because Distr requires authentication to access the registry API, but our Agents are able to inject relevant authentication credentials where they are needed.

A common use-case is to host both a Helm chart and related container images on Distr.
In this case the agent has to create a `Secret` resource in the same namespace as the helm release and supply that as an image pull secret to workloads created as part of the Helm release.
To take advantage of this feature, you have to include one or more `imagePullSecrets` or `pullSecrets` entries in the values.yaml field of your application version where you want the agent to inject the secret name. For example:

```yaml
imagePullSecrets:
  - name: foo
myAppServer:
  imagePullSecrets: []
myAppDb:
  imagePullSecrets: {}
image:
  pullSecrets: []
```

will be transformed into

```yaml
imagePullSecrets:
  - name: foo
  - name: sh.distr.agent.v1.xxxxxx.pull
myAppServer:
  imagePullSecrets:
    name: sh.distr.agent.v1.xxxxxx.pull
myAppDb:
  # not changed because it is not an array
  imagePullSecrets: {}
image:
  pullSecrets:
    name: sh.distr.agent.v1.xxxxxx.pull
```
