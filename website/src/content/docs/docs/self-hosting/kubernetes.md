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

Caddy stores the certificates it obtains on a persistent volume, which the chart keeps when the
release is uninstalled so that a reinstall does not have to reissue a certificate for every custom
domain.

The default is a single replica, because Caddy instances form a cluster only when they use the same
storage. Sharing storage is what lets them coordinate issuance, share certificates and solve each
other's ACME challenges — and the last part is not optional behind a load balancer, which routes the
certificate authority's validation request to an arbitrary replica. To run more than one replica,
give all of them the same storage in one of two ways:

- Set `caddy.persistence.accessModes` to `[ReadWriteMany]` with a storage class that supports it
  (EFS, Filestore, Azure Files, CephFS, …), or point `caddy.persistence.existingClaim` at such a
  claim.
- Set `caddy.storage` to a Caddyfile storage block for a distributed backend (Redis, Consul, …) and
  `caddy.image.repository` to an image built with that storage module, since Caddy has no runtime
  plugins.

The chart refuses to render a `caddy.replicaCount` above 1 without either of them.

To try custom domains on a local cluster, where no certificate authority can validate an ACME
challenge for a domain that does not resolve publicly, see
[`github.com/distr-sh/distr/deploy/minikube`](https://github.com/distr-sh/distr/blob/main/deploy/minikube/custom-domains-values.yaml).
It runs everything in-cluster, including PostgreSQL and RustFS, and is meant for local testing only.

## Log processing (Loki)

The chart includes a bundled [Grafana Loki](https://grafana.com/oss/loki/) instance (enabled by default) that stores deployment and deployment target logs with a 30-day retention.
Loki is preconfigured to persist its data in the in-cluster RustFS object storage, so enabling RustFS (as the quick start above does with `--set rustfs.enabled=true`) makes it work out of the box.

If you use an external S3-compatible object storage instead of the bundled RustFS, point `loki.loki.storage.s3` (and the bucket-provisioning init container under `loki.singleBinary.initContainers`) at it.
To use an externally managed Loki instance, set `loki.enabled=false` and configure `LOKI_URL` (and optionally `LOKI_BEARER_TOKEN` or `LOKI_BASIC_AUTH_*`) in `hub.env`.
