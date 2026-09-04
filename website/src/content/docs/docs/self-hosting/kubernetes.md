---
title: Kubernetes
description: Deploy Distr in your Kubernetes cluster with our Helm chart, from a local test install with bundled PostgreSQL and RustFS to production values for EKS on AWS and GKE on GCP.
sidebar:
  label: Kubernetes
  order: 3
---

Distr is available as a [Helm chart](/glossary/helm-chart/) distributed via ghcr.io.
Every setting mentioned on this page is documented in the chart's reference [values.yaml](https://artifacthub.io/packages/helm/distr/distr?modal=values).

Complete values files for the setups below live under [`deploy/charts/distr/examples`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples), one per folder, and match the [Docker Compose examples](/docs/self-hosting/docker/) of the same name.
Every one of them deploys the Hub and [Loki](/docs/self-hosting/configuration/#log-processing-loki) for log processing, and they differ in the edition they run and in what they bring along:

| Example                                                                                                     | Edition    | Includes                                                   | Intended for                                                                  |
| ----------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------- | ----------------------------------------------------------------------------- |
| [`quickstart`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples/quickstart)         | Community  | PostgreSQL, RustFS object storage                          | Trying Distr out on a local [minikube cluster](https://minikube.sigs.k8s.io/) |
| [`community`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples/community)           | Community  | PostgreSQL, RustFS, Ingress with cert-manager              | Community production on a generic cluster                                     |
| [`enterprise`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples/enterprise)         | Enterprise | PostgreSQL, RustFS, Ingress with cert-manager              | Enterprise production on a generic cluster                                    |
| [`enterprise-aws`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples/enterprise-aws) | Enterprise | ALB Ingress (database and S3 are external)                 | Stateless enterprise production on EKS                                        |
| [`enterprise-gcp`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples/enterprise-gcp) | Enterprise | Ingress with cert-manager (Cloud SQL and GCS are external) | Stateless enterprise production on GKE                                        |

## Trying it out locally

To install Distr in [Kubernetes](/glossary/kubernetes/) with its dependencies bundled, run:

```shell
helm upgrade --install --wait --namespace distr --create-namespace \
  distr oci://ghcr.io/distr-sh/charts/distr \
  --set postgresql.enabled=true --set rustfs.enabled=true
```

This deploys the Hub together with PostgreSQL, the RustFS object storage and Loki, so you don't have to modify any values to get a working instance.
Both bundled dependencies keep their data in a single `ReadWriteOnce` volume each and use the default credentials from the values file, which makes them fine for a test cluster and unsuitable for production.
The same install as a values file is [`examples/quickstart/values.yaml`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/quickstart/values.yaml).

For a local cluster with custom domains enabled, see
[`examples/quickstart/custom-domains-values.yaml`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/quickstart/custom-domains-values.yaml).

## Running in production

For production, disable the bundled dependencies and point the chart at managed services instead.
This part is the same on every cloud:

- Leave `postgresql.enabled` at `false` and set `externalDatabase.existingSecret` to a secret holding the connection URI. The alternative, `externalDatabase.uri`, puts the URI into the release values in plain text.
- Use external object storage for both the registry (`REGISTRY_S3_*` in `hub.env`) and Loki (`loki.loki.storage`). Create both buckets up front, set `REGISTRY_S3_CREATE_BUCKET` to `false`, and drop the `create-loki-bucket` init container with `loki.singleBinary.initContainers: []`, which only exists to provision a bucket in the in-cluster RustFS.
- Put `JWT_SECRET`, the license key and the object storage credentials in a `secretKeyRef` or in `hub.envFrom`, not in your `hub.env` values.
- Enable a scratch volume with `hub.scratch.enabled`, so the registry buffers layer uploads on disk instead of in memory.
- Add an Ingress for the two hostnames the chart serves, the app and the registry. Registry pushes have no size limit, so raise or disable the request body limit of your ingress controller on the registry host.

The rest of the defaults are already shaped for production: two Hub replicas with a `PodDisruptionBudget`, and the [maintenance jobs](/docs/self-hosting/maintenance/) as `CronJob`s under `cronJobs` instead of in-process cron, which is what you want as soon as more than one replica runs.
Set `resources` and consider `autoscaling` based on the [System Requirements](/docs/self-hosting/system-requirements/).

Without managed services next door, the [`community`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/community/values.yaml) and [`enterprise`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/enterprise/values.yaml) examples keep PostgreSQL and RustFS in the cluster and put an Ingress with a cert-manager certificate in front, which is the trade-off the self-contained Compose examples make as well.
Each of them keeps its data on a single `ReadWriteOnce` volume.

### Distr Enterprise

The chart defaults to the Community Edition image (`ghcr.io/distr-sh/distr-ce`), which is free and needs no license.
Paid plans run the Enterprise image, available as `ghcr.io/distr-sh/distr-ee` and from our own registry as `registry.distr.sh/enterprise/distr-ee`.
Both need the credentials you received from us, so create a pull secret and reference it in `imagePullSecrets`.

:::tip[Let Distr manage your own instance]
The smoothest way to run a paid plan is to deploy the chart with Distr itself, through a [Kubernetes agent](/docs/agents/kubernetes-agent/) in the target cluster.
The agent then handles the rollout of new Hub versions and injects the license key for you: put `value: '{{ index .LicenseKeys "Distr" }}'` on the `LICENSE_KEY` entry of your Helm values and it is resolved at deploy time from the [license key](/docs/platform/license-keys/) named `Distr`, so the token is never stored in the release.
:::

```yaml
image:
  repository: ghcr.io/distr-sh/distr-ee

imagePullSecrets:
  - name: distr-registry

hub:
  env:
    - name: LICENSE_KEY
      valueFrom:
        secretKeyRef:
          name: distr-license
          key: licenseKey
    # ... the rest of your environment
```

Keep in mind that Helm replaces the `hub.env` list rather than merging it, so every variable you need has to be in the same list.

### Distr Enterprise on AWS

Run the Hub on EKS, the database on RDS for PostgreSQL and both buckets on S3 in the region of the cluster.
The buckets need the same setup as for the [Docker Compose example](/docs/self-hosting/docker/#distr-enterprise-on-aws): private, public access blocked, versioning off so blob cleanup can reclaim storage, and a lifecycle rule that aborts incomplete multipart uploads.

For access, use IRSA rather than static keys. Annotate the Hub's service account with a role that carries the S3 policy and leave the access keys unset, so the AWS SDK picks up the web identity credentials by itself.
Loki needs the same on its own service account.

The full values file, including the Enterprise image and the secrets it expects, is [`examples/enterprise-aws/values.yaml`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/enterprise-aws/values.yaml):

```yaml
serviceAccount:
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/distr-hub

hub:
  env:
    - name: DISTR_HOST
      value: https://distr.example.com
    - name: REGISTRY_ENABLED
      value: 'true'
    - name: REGISTRY_HOST
      value: pkg.example.com
    - name: REGISTRY_S3_BUCKET
      value: example-distr-registry
    - name: REGISTRY_S3_REGION
      value: us-east-2
    - name: REGISTRY_S3_CREATE_BUCKET
      value: 'false'
    - name: REGISTRY_S3_USE_PATH_STYLE
      value: 'false'
    - name: REGISTRY_S3_ALLOW_REDIRECT
      value: 'true'
    - name: LOKI_URL
      value: http://distr-loki:3100
    - name: JWT_SECRET
      valueFrom:
        secretKeyRef:
          name: distr-secrets
          key: jwtSecret
  scratch:
    enabled: true
    size: 50Gi
    storageClassName: gp3

postgresql:
  enabled: false
rustfs:
  enabled: false

externalDatabase:
  existingSecret: distr-database
  existingSecretUriKey: uri

loki:
  serviceAccount:
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/distr-loki
  loki:
    storage:
      type: s3
      bucketNames:
        chunks: example-distr-logs
        ruler: example-distr-logs
        admin: example-distr-logs
      s3:
        region: us-east-2
        endpoint: null
        accessKeyId: null
        secretAccessKey: null
        s3ForcePathStyle: false
  singleBinary:
    initContainers: []

ingress:
  enabled: true
  className: alb
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTPS":443}]'
    alb.ingress.kubernetes.io/certificate-arn: arn:aws:acm:us-east-2:123456789012:certificate/abc123
  hosts:
    - host: distr.example.com
      paths:
        - path: /
          pathType: ImplementationSpecific
          port:
            name: http
    - host: pkg.example.com
      paths:
        - path: /
          pathType: ImplementationSpecific
          port:
            name: artifacts
```

The `null` values in `loki.loki.storage.s3` delete the RustFS endpoint and credentials the chart ships as defaults, which is what makes Loki fall back to the credentials of its service account.
An Application Load Balancer applies no request body limit, so registry pushes work without further configuration.
The [`REGISTRY_S3_ALLOW_REDIRECT`](/docs/self-hosting/configuration/#oci-registry) setting above also offloads layer downloads to pre-signed S3 URLs.

If you prefer static credentials over IRSA, drop the service account annotations and provide `REGISTRY_S3_ACCESS_KEY_ID` / `REGISTRY_S3_SECRET_ACCESS_KEY` from a secret, along with `accessKeyId` and `secretAccessKey` under `loki.loki.storage.s3`.

### Distr Enterprise on GCP

Run the Hub on GKE, the database on Cloud SQL for PostgreSQL with a private IP in the cluster's VPC, and both buckets on Cloud Storage in the region of the cluster.
As in the Compose example, the two buckets are reached in different ways: the registry uses the S3 interoperability API with an HMAC key, while Loki uses the native GCS API and authenticates through Workload Identity.
So bind a Google service account with `roles/storage.objectAdmin` on the logs bucket to Loki's Kubernetes service account, and keep the HMAC key of a service account with access to the registry bucket in a secret.

The full values file, including the Enterprise image and the secrets it expects, is [`examples/enterprise-gcp/values.yaml`](https://github.com/distr-sh/distr/blob/main/deploy/charts/distr/examples/enterprise-gcp/values.yaml):

```yaml
hub:
  env:
    - name: DISTR_HOST
      value: https://distr.example.com
    - name: REGISTRY_ENABLED
      value: 'true'
    - name: REGISTRY_HOST
      value: pkg.example.com
    - name: REGISTRY_S3_BUCKET
      value: example-distr-registry
    - name: REGISTRY_S3_REGION
      value: europe-west3
    - name: REGISTRY_S3_ENDPOINT
      value: https://storage.googleapis.com
    - name: REGISTRY_S3_CREATE_BUCKET
      value: 'false'
    - name: REGISTRY_S3_USE_PATH_STYLE
      value: 'true'
    # The three settings the GCS interoperability API needs
    - name: REGISTRY_S3_REQUEST_CHECKSUM_CALCULATION
      value: 'true'
    - name: REGISTRY_S3_RESPONSE_CHECKSUM_VALIDATION
      value: 'true'
    - name: REGISTRY_RESIGN_FOR_GCP
      value: 'true'
    - name: REGISTRY_S3_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: distr-secrets
          key: hmacAccessKeyId
    - name: REGISTRY_S3_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: distr-secrets
          key: hmacSecretAccessKey
    - name: LOKI_URL
      value: http://distr-loki:3100
    - name: JWT_SECRET
      valueFrom:
        secretKeyRef:
          name: distr-secrets
          key: jwtSecret
  scratch:
    enabled: true
    size: 50Gi
    storageClassName: premium-rwo

postgresql:
  enabled: false
rustfs:
  enabled: false

externalDatabase:
  existingSecret: distr-database
  existingSecretUriKey: uri

loki:
  serviceAccount:
    annotations:
      iam.gke.io/gcp-service-account: distr-loki@example-project.iam.gserviceaccount.com
  loki:
    storage:
      type: gcs
      bucketNames:
        chunks: example-distr-logs
        ruler: example-distr-logs
        admin: example-distr-logs
    schemaConfig:
      configs:
        - from: '2024-01-01'
          store: tsdb
          object_store: gcs
          schema: v13
          index:
            prefix: index_
            period: 24h
    compactor:
      delete_request_store: gcs
  singleBinary:
    initContainers: []
```

Loki picks the object store client per schema period and for delete requests by name, so `schemaConfig` and `compactor` have to name `gcs` as well.
The chart's defaults name `s3`, which would leave Loki looking for an S3 client that the release does not configure.

For ingress, we recommend an ingress controller you configure yourself, such as ingress-nginx with `proxy-body-size: "0"`.
GKE's built-in GCE ingress applies a 30 second backend response timeout, which is short enough to break the upload of a single large layer.
Raising it takes a `BackendConfig` annotation on the Service, and this chart does not render one.

## Custom domains

Vendor organizations on the Business plan can serve the Distr app and the container registry under
their different domains for different customers.
Serving those domains requires a proxy in front of Distr that obtains a
certificate for every domain a vendor registers, so the chart ships an optional
[Caddy](https://caddyserver.com/) deployment with
[on-demand TLS](https://caddyserver.com/docs/automatic-https#on-demand-tls):

```yaml
hub:
  env:
    # DNS record your vendors point their domains at
    - name: CUSTOM_DOMAIN_TARGET
      value: whitelabel.example.com

caddy:
  enabled: true
  acmeEmail: ops@example.com
```

Point that hostname at the external address of the `<release>-caddy` `LoadBalancer` Service. One
target covers every domain a vendor registers, registry domains included, because Caddy routes
registry traffic by the `/v2/` path prefix the OCI distribution API mandates rather than by
hostname. Before issuing a certificate, Caddy asks the Hub whether a domain is registered, using an
internal Service that must never be exposed publicly. `CUSTOM_DOMAIN_TARGET` is what enables the
feature in the UI, so the chart refuses to render a Caddy deployment without it. Helm replaces the
`hub.env` list rather than merging it, so add the variable to the rest of your hub environment
instead of setting it on its own.

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
