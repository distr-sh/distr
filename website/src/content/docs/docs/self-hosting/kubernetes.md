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

## Custom domains

Vendor organizations on the Business plan can serve the Distr app and the container registry under
their own domains. Serving those domains requires a proxy in front of the Hub that obtains a
certificate for every domain a vendor registers, so the chart ships an optional
[Caddy](https://caddyserver.com/) deployment with
[on-demand TLS](https://caddyserver.com/docs/automatic-https#on-demand-tls):

```yaml
caddy:
  enabled: true
  acmeEmail: ops@example.com
  # DNS records your vendors point their domains at
  appCnameTarget: cname.example.com
  registryCnameTarget: cname.example.com
```

Point the `appCnameTarget` and `registryCnameTarget` hostnames at the external address of the
`<release>-caddy` `LoadBalancer` Service. Before issuing a certificate, Caddy asks the Hub whether a
domain is registered, using an internal Service that must never be exposed publicly. Setting the two
CNAME targets is what enables the feature in the UI, so both values are required when
`caddy.enabled` is `true`.

Caddy runs as a StatefulSet with two replicas by default, each with its own volume for certificate
storage. Replicas therefore do not share certificates and each one issues its own for a given
domain, which keeps the deployment free of a `ReadWriteMany` volume or a custom Caddy image with a
clustered storage module. Take this into account when raising `caddy.replicaCount`, since every
replica counts against the ACME provider's rate limits. To share storage instead, set
`caddy.storage` to a Caddyfile storage block and use an image that contains the matching module.

## Log processing (Loki)

The chart includes a bundled [Grafana Loki](https://grafana.com/oss/loki/) instance (enabled by default) that stores deployment and deployment target logs with a 30-day retention.
Loki is preconfigured to persist its data in the in-cluster RustFS object storage, so enabling RustFS (as the quick start above does with `--set rustfs.enabled=true`) makes it work out of the box.

If you use an external S3-compatible object storage instead of the bundled RustFS, point `loki.loki.storage.s3` (and the bucket-provisioning init container under `loki.singleBinary.initContainers`) at it.
To use an externally managed Loki instance, set `loki.enabled=false` and configure `LOKI_URL` (and optionally `LOKI_BEARER_TOKEN` or `LOKI_BASIC_AUTH_*`) in `hub.env`.
