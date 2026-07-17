---
title: System Requirements
description: Distr is written in Go and highly resource efficient. Learn about the required software and recommended resources for self-hosting the Hub, the registry, and the all-in-one Docker Compose setup.
sidebar:
  label: System Requirements
  order: 1
---

Distr is written in [Go](https://go.dev/) and highly resource efficient.
Our hosted offering serves thousands of requests every second with just two app servers at **30 MB / 50m CPU** each and a PostgreSQL database at **1 GB / 200m CPU** (excluding read replicas).
This means you can run a self-hosted Distr instance comfortably on modest hardware.

:::tip
CPU values on this page use the Kubernetes notation of CPU millicores (`m`), where `1000m` equals one full CPU core. So `50m` means 5% of a single core and `200m` means 20% of a single core.
:::

## Required software

To run the all-in-one [Docker Compose](/docs/self-hosting/docker/) setup you need:

- **Docker Engine** ≥ v29
- **Docker Compose** ≥ v5.3 (the `docker compose` plugin; the log storage setup uses `pre_start` lifecycle hooks introduced in v5.3.0)
- **curl** (to download the deployment manifest)

## Average resource consumption

The following table lists the average CPU and memory per component. These values match the footprint of our staging environments and are a good starting point for a small self-hosted instance. Scale them up based on your request volume and artifact sizes.

| Component               | CPU          | RAM         |
| ----------------------- | ------------ | ----------- |
| Distr                   | 100m         | 128 MB      |
| PostgreSQL (database)   | 250m         | 512 MB      |
| RustFS (object storage) | 100m         | 256 MB      |
| Loki (log storage)      | 100m         | 512 MB      |
| Caddy (reverse proxy)   | 50m          | 64 MB       |
| **Total**               | **~0.6 CPU** | **~1.5 GB** |

&nbsp;

:::note
The average values are per-component footprints for Distr itself and do not include the operating system, Docker, or other system services. The workloads will also burst beyond these values on certain operations — Loki in particular consumes significantly more CPU and memory while serving log queries and exports over large time ranges.

We therefore recommend provisioning a VM with a **minimum of 2 CPUs and 4 GB RAM**.
:::

## Persistence

Distr itself does not require any persistent volumes. All state is stored in the PostgreSQL database, the S3-compatible object storage (registry blobs and log chunks), and the environment configuration.

## Log storage (Loki)

Deployment and deployment target logs are stored in [Grafana Loki](https://grafana.com/oss/loki/), which is included in all shipped deployment methods (Docker Compose and Helm) in monolithic (single-binary) mode.
Loki persists log chunks and its index in the same S3-compatible object storage as the registry, using a dedicated `loki` bucket, and only needs a small local volume for its write-ahead log and caches.
The shipped configuration retains logs for 30 days.

## Registry

For the optional registry, a scratch volume is recommended (sized by concurrently uploading image sizes).
OCI registry uploads are buffered while they are being received: with a scratch volume in place, Distr buffers the upload to disk instead of holding it in memory.
Without it, large layer uploads are buffered to RAM, which can significantly increase the memory footprint of the Hub.

We also highly recommend backing the registry with an external S3-compatible object storage like AWS S3.
On top of being more scalable and durable than a single local RustFS container, it lets the registry serve blob (layer) downloads via **pre-signed URLs**: instead of streaming the layer through the Hub, the registry responds with an HTTP `307 Temporary Redirect` to a short-lived pre-signed URL, so clients download layers directly from the object storage.
This offloads pull bandwidth from the Hub and keeps its CPU and memory footprint low even under heavy pull load.
This behavior is enabled by default and can be controlled via `REGISTRY_S3_ALLOW_REDIRECT`.

## Networking & ports

Distr exposes two HTTP endpoints: the app (web UI and API) and the registry (OCI artifacts). Both are typically served under separate hostnames and put behind a TLS-terminating reverse proxy (Caddy in our Docker Compose setup, an Ingress controller in Kubernetes).

Regardless of how you deploy, make sure the following is in place:

- **Public domain names** for the app, registry and [metrics](/docs/self-hosting/prometheus/) configured and pointed to the public IP of your VM (or load balancer).
- **Port `443`** publicly reachable for HTTPS traffic.
- **Port `80`** publicly reachable as well if you let the reverse proxy (e.g. Caddy) obtain and renew TLS certificates automatically via ACME / Let's Encrypt.

## Production recommendations

For production use, we recommend the following:

- **Externally managed PostgreSQL and object storage.** This lets you scale, upgrade, and operate these stateful components independently of the Distr Hub, and keeps the Hub itself stateless and easy to upgrade.
- **A highly available (HA) architecture.** Run multiple Hub replicas behind a load balancer so the control plane stays available during upgrades and node failures. Our [Helm chart](/docs/self-hosting/kubernetes/) supports this via `replicaCount` and `autoscaling`.
- **An external job trigger** for maintenance jobs instead of the Hub's built-in scheduler. The Helm chart ships these as Kubernetes `CronJob`s (see `cronJobs` in the chart values), which is the recommended way to run [maintenance jobs](/docs/self-hosting/maintenance/) in production.
- **Regular backups** of your PostgreSQL database and object storage. The Distr Hub itself is stateless, so all persistent state lives in these two components. Back them up regularly and test your restore procedure.
- **External secret management.** Store sensitive configuration such as the database credentials, `JWT_SECRET`, and object storage keys in a dedicated secret manager (e.g. a Kubernetes `Secret`, Vault, or your cloud provider's secret store) instead of plain-text environment files.
