---
title: Docker Compose
description: Run Distr with Docker Compose, from an all-in-one stack for trying it out locally to production reference stacks for AWS and GCP with managed PostgreSQL and object storage.
sidebar:
  label: Docker Compose
  order: 2
---

The easiest way to host your own Distr Hub is Docker Compose.
You need Docker Engine 29 or later and the Docker Compose plugin 5.3 or later.

All five Compose stacks under [`deploy/docker`](https://github.com/distr-sh/distr/tree/main/deploy/docker) run the Hub and [Loki](/docs/self-hosting/configuration/#log-processing-loki) for log processing.
They differ in the edition they run and in what they bring along:

| Example                                                                                      | Edition    | Includes                                         | Intended for                           |
| -------------------------------------------------------------------------------------------- | ---------- | ------------------------------------------------ | -------------------------------------- |
| [`quickstart`](https://github.com/distr-sh/distr/tree/main/deploy/docker/quickstart)         | Community  | PostgreSQL, RustFS object storage                | Trying Distr out locally               |
| [`community`](https://github.com/distr-sh/distr/tree/main/deploy/docker/community)           | Community  | PostgreSQL, RustFS, Caddy with https             | Community production on a generic VM   |
| [`enterprise`](https://github.com/distr-sh/distr/tree/main/deploy/docker/enterprise)         | Enterprise | PostgreSQL, RustFS, Caddy with https             | Enterprise production on a generic VM  |
| [`enterprise-aws`](https://github.com/distr-sh/distr/tree/main/deploy/docker/enterprise-aws) | Enterprise | Caddy with https (database and S3 are external)  | Stateless enterprise production on AWS |
| [`enterprise-gcp`](https://github.com/distr-sh/distr/tree/main/deploy/docker/enterprise-gcp) | Enterprise | Caddy with https (database and GCS are external) | Stateless enterprise production on GCP |

## Trying it out locally

First, download and unpack the Distr Docker Compose deployment manifest from the latest release:

```shell
mkdir distr && cd distr && curl -fsSL https://github.com/distr-sh/distr/releases/latest/download/deploy-quickstart.tar | tar -x
```

This command creates a new directory called `distr` containing two files: `docker-compose.yaml` and `.env`.
For a basic setup, you don't have to modify `docker-compose.yaml`, but please open `.env` in your favorite text editor and change the values of `POSTGRES_PASSWORD` and `JWT_SECRET`.
Feel free to also change the value of `DISTR_HOST`, if you intend to make your instance publicly available.
Once you are happy with your configuration, simply start the Hub using Docker Compose:

```shell
docker compose up -d
```

Open [`http://localhost:8080`](http://localhost:8080) to access Distr.

## Running in production

In production, run PostgreSQL and the object storage as managed services so that the Compose stack stays stateless.
That is what the `enterprise-aws` and `enterprise-gcp` stacks do: they run `hub`, `loki` and `caddy` in Docker and nothing else.
All of their configuration lives in the `.env` file next to the Compose file.

Whichever cloud you use, you need the same pieces:

- A VM with at least 2 CPUs and 4 GB RAM (see [System Requirements](/docs/self-hosting/system-requirements/)).
- Two public hostnames, one for the app and one for the registry, both pointing at the VM. Caddy obtains and renews their certificates via ACME, so TCP `80` and `443` have to be reachable from the internet.
- A managed PostgreSQL instance the VM can reach over a private network, with TLS enforced. We test against PostgreSQL 18 with 2 CPUs and 2 GB RAM.
- Two buckets, one for the registry blobs and one for Loki's log chunks.
- Disk space for the scratch volume, where the registry buffers layer uploads instead of holding them in memory. Every stack mounts one into the Hub, so give the VM room for the layers you expect to be pushed at the same time.
- A `JWT_SECRET` from `openssl rand -base64 32`, and a `LICENSE_KEY` if you run a paid plan.

On a single VM, keep running the [maintenance jobs](/docs/self-hosting/maintenance/) inside the Hub process through the `*_CRON` variables in `.env`.
Switch them off and trigger the `cleanup` and `maintenance` subcommands from outside only once you run more than one Hub replica, since every replica would otherwise run every job.

### Distr Enterprise

The `quickstart` and `community` stacks run the Community Edition image (`ghcr.io/distr-sh/distr-ce`), which is free and needs no license.
The three enterprise stacks run the Enterprise image `registry.distr.sh/enterprise/distr-ee` instead.
It comes from our own registry, so run `docker login registry.distr.sh` with the credentials you received from us before you start the stack.

:::tip[Let Distr manage your own instance]
The smoothest way to run a paid plan is to deploy the stack with Distr itself, through a [Docker agent](/docs/agents/docker-agent/) on the target VM.
The agent then handles the rollout of new Distr versions, configures registry credentials and injects the license key for you: the `enterprise`, `enterprise-aws` and `enterprise-gcp` stacks ship `LICENSE_KEY={{ index .LicenseKeys "Distr" }}` in their `.env`, which the agent resolves at deploy time from the [license key](/docs/platform/license-keys/) named `Distr`.
:::

### Distr Enterprise on AWS

We recommend [Amazon Lightsail](https://aws.amazon.com/lightsail/) for the VM.
It bundles the instance, a static IP, a DNS zone, a firewall and optional daily snapshots at a fixed monthly price:

1. Create a Linux instance with a plan that has at least 2 vCPUs and 4 GB RAM, and install Docker with the Compose plugin.
2. Attach a static IP and point both hostnames (app and registry) at it, either in a Lightsail DNS zone or at your registrar.
3. Open TCP `80` and `443` in the instance firewall.

A Lightsail managed database is the simplest choice. It sits in the same Lightsail VPC as the instance, is not publicly reachable unless you ask for it and takes daily backups.
Pick RDS for PostgreSQL when you need Multi-AZ failover, larger instance classes or a read replica, which the Hub can use via `DATABASE_READONLY_URL`.
RDS instances run in the region's default VPC, so a Lightsail instance only reaches them once you enable VPC peering under Account → Advanced in the Lightsail console.

Neither option creates the database Distr expects, so connect to the new instance from the VM and create it yourself:

```shell
psql "postgres://dbmasteruser:<password>@ls-abc123.abcdefg.us-east-2.rds.amazonaws.com:5432/postgres?sslmode=require" \
  -c 'CREATE DATABASE distr'
```

Create the two buckets in the region of the VM:

- Keep them private. Distr always reaches the buckets with credentials, and pull traffic that is offloaded to the object storage uses short-lived pre-signed URLs.
- Leave object versioning off. The [artifact blob cleanup job](/docs/self-hosting/maintenance/) reclaims storage by deleting unreferenced blobs, and versioning would keep them around as noncurrent versions.
- On plain S3, add a lifecycle rule that aborts incomplete multipart uploads after a day. The registry uploads large layers as multipart uploads, and an interrupted push leaves parts behind that you keep paying for.

Buckets created in Lightsail come with their own access keys, listed under Permissions on the bucket, so there is no IAM user or bucket policy to manage.
Copy one key pair per bucket into the matching `REGISTRY_S3_*` and `LOKI_S3_*` variables. Neither of them needs an endpoint, since the regional one follows from `REGISTRY_S3_REGION` and `LOKI_S3_REGION`. Do not add `REGISTRY_S3_ENDPOINT` back with an empty value, which the Hub reads as a custom endpoint and the AWS SDK then rejects.
If you do set one, use the plain regional endpoint (`https://s3.us-east-2.amazonaws.com`) without the bucket in it.

The relevant part of `deploy/docker/enterprise-aws/.env` then looks like this:

```dotenv
LICENSE_KEY="eyJhbGciOi..."

DISTR_APP_HOSTNAME="distr.example.com"
DISTR_REGISTRY_HOSTNAME="pkg.example.com"
CADDY_ACME_EMAIL="ops@example.com"
DISTR_HOST="https://${DISTR_APP_HOSTNAME}"
JWT_SECRET="Zm9vYmFyYmF6..."

DATABASE_URL="postgres://distr:<password>@distr-db.abc123.us-east-2.rds.amazonaws.com:5432/distr?sslmode=require"

REGISTRY_ENABLED=true
REGISTRY_HOST="${DISTR_REGISTRY_HOSTNAME}"
REGISTRY_S3_BUCKET="example-distr-registry"
REGISTRY_S3_REGION="us-east-2"
REGISTRY_S3_ACCESS_KEY_ID="AKIA..."
REGISTRY_S3_SECRET_ACCESS_KEY="..."
REGISTRY_S3_USE_PATH_STYLE=false
REGISTRY_S3_ALLOW_REDIRECT=true
REGISTRY_SCRATCH_DIR="/scratch"

LOKI_URL="http://loki:3100"
LOKI_S3_BUCKET="example-distr-logs"
LOKI_S3_REGION="us-east-2"
LOKI_S3_ENDPOINT=""
LOKI_S3_ACCESS_KEY_ID="AKIA..."
LOKI_S3_SECRET_ACCESS_KEY="..."
LOKI_S3_USE_PATH_STYLE=false
```

`REGISTRY_S3_ALLOW_REDIRECT=true` answers layer downloads with a redirect to a pre-signed S3 URL, which keeps pull bandwidth off the Hub.
Turn it off if your clients cannot reach S3 directly.

### Distr Enterprise on GCP

Use a Compute Engine instance in the same region as the buckets, `e2-medium` (2 vCPU, 4 GB) or larger, with a reserved static external IP and a firewall rule for TCP `80` and `443`.
Attach a dedicated service account to the instance and grant it `roles/storage.objectAdmin` on the two buckets.

For the database, use Cloud SQL for PostgreSQL with a private IP in the same VPC as the instance, plus automated backups and point-in-time recovery.

The stack talks to Google Cloud Storage in two different ways, one per bucket:

- The registry bucket goes through the S3 interoperability (XML) API, because the OCI registry speaks S3. Create an HMAC key for the service account under Cloud Storage → Settings → Interoperability, use it as `REGISTRY_S3_ACCESS_KEY_ID` and `REGISTRY_S3_SECRET_ACCESS_KEY`, and point `REGISTRY_S3_ENDPOINT` at `https://storage.googleapis.com` with path-style addressing. The interoperability API also needs the three compatibility settings the shipped `.env` contains (`REGISTRY_S3_REQUEST_CHECKSUM_CALCULATION`, `REGISTRY_S3_RESPONSE_CHECKSUM_VALIDATION` and `REGISTRY_RESIGN_FOR_GCP`).
- The logs bucket goes through the native GCS API, so Loki only needs its name in `LOKI_GCS_BUCKET_NAME`.

The Compose stack passes no storage credentials to Loki.
Loki falls back to Google's application default credentials, which on Compute Engine are the credentials of the service account attached to the instance, fetched from the metadata server (see Loki's [storage configuration](https://grafana.com/docs/loki/latest/configure/storage/)).
Whatever you grant that service account is therefore what Loki can do with the logs bucket, and on an instance without one it cannot write any chunks.

Create both buckets with uniform bucket-level access and public access prevention, in a single region rather than multi-region, and add a lifecycle rule on the registry bucket that aborts incomplete multipart uploads.

The relevant part of `deploy/docker/enterprise-gcp/.env`:

```dotenv
LICENSE_KEY="eyJhbGciOi..."

DISTR_APP_HOSTNAME="distr.example.com"
DISTR_REGISTRY_HOSTNAME="pkg.example.com"
CADDY_ACME_EMAIL="ops@example.com"
DISTR_HOST="https://${DISTR_APP_HOSTNAME}"
JWT_SECRET="Zm9vYmFyYmF6..."

DATABASE_URL="postgres://distr:<password>@10.42.0.3:5432/distr?sslmode=require"

REGISTRY_ENABLED=true
REGISTRY_HOST="${DISTR_REGISTRY_HOSTNAME}"
REGISTRY_S3_BUCKET="example-distr-registry"
REGISTRY_S3_REGION="europe-west3"
REGISTRY_S3_ENDPOINT="https://storage.googleapis.com"
REGISTRY_S3_ACCESS_KEY_ID="GOOG1E..."
REGISTRY_S3_SECRET_ACCESS_KEY="..."
REGISTRY_S3_USE_PATH_STYLE=true
REGISTRY_S3_ALLOW_REDIRECT=true
REGISTRY_S3_REQUEST_CHECKSUM_CALCULATION=true
REGISTRY_S3_RESPONSE_CHECKSUM_VALIDATION=true
REGISTRY_RESIGN_FOR_GCP=true
REGISTRY_SCRATCH_DIR="/scratch"

LOKI_URL="http://loki:3100"
LOKI_GCS_BUCKET_NAME="example-distr-logs"
```
