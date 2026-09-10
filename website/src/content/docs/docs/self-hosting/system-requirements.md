---
title: System Requirements
description: Distr is written in Go and highly resource efficient. Learn about the recommended resources for self-hosting the Hub, the registry and the log processing backend, and what to plan for in production.
sidebar:
  label: System Requirements
  order: 1
---

Distr is written in [Go](https://go.dev/) and highly resource efficient.
Our hosted offering serves thousands of requests every second with just two app servers at 30 MB / 50m CPU each and a PostgreSQL database at 1 GB / 200m CPU (excluding read replicas).
This means you can run a self-hosted Distr instance comfortably on modest hardware.

This page covers how much hardware to plan for. The software each deployment method needs is listed with the method itself, on the [Docker Compose](/docs/self-hosting/docker/) and [Kubernetes](/docs/self-hosting/kubernetes/) pages.

:::tip
CPU values on this page use the Kubernetes notation of CPU millicores (`m`), where `1000m` equals one full CPU core. So `50m` means 5% of a single core and `200m` means 20% of a single core.
:::

## Average resource consumption

The following table lists the average CPU and memory per component. These values match the footprint of our staging environments and are a good starting point for a small self-hosted instance. Scale them up based on your request volume and artifact sizes.

| Component               | CPU          | RAM         |
| ----------------------- | ------------ | ----------- |
| Distr                   | 100m         | 128 MB      |
| PostgreSQL (database)   | 250m         | 512 MB      |
| RustFS (object storage) | 100m         | 256 MB      |
| Loki (log processing)   | 100m         | 512 MB      |
| Caddy (reverse proxy)   | 50m          | 64 MB       |
| **Total**               | **~0.6 CPU** | **~1.5 GB** |

&nbsp;

:::note
The average values are per-component footprints for Distr itself and do not include the operating system, Docker or other system services. The workloads also burst beyond these values on certain operations. Loki in particular consumes significantly more CPU and memory while serving log queries and exports over large time ranges.

We therefore recommend provisioning a VM with a minimum of 2 CPUs and 4 GB RAM.
:::

## Persistence

Distr itself does not require any persistent volumes. All state lives in the PostgreSQL database, in the S3-compatible object storage (registry blobs and log chunks) and in the environment configuration.
The registry scratch volume below holds no state and can be thrown away with the container.

## Log processing (Loki)

Deployment and deployment target logs are processed and stored via [Grafana Loki](https://grafana.com/oss/loki/), which is included in all shipped deployment methods (Docker Compose and Helm) in monolithic (single-binary) mode.
Loki persists log chunks and its index in the same S3-compatible object storage as the registry, using a dedicated `loki` bucket, and only needs a small local volume for its write-ahead log and caches.
On Google Cloud it talks to the bucket through the native GCS API instead, as the [GCP examples](/docs/self-hosting/docker/#distr-enterprise-on-gcp) do.
The shipped configuration retains logs for 30 days.

## Registry

The registry buffers uploads while it receives them. Give it a scratch volume (`REGISTRY_SCRATCH_DIR`) so it buffers them to disk, sized for the layers you expect to be pushed at the same time.
Without one, large layer uploads go to RAM and can increase the memory footprint of the Hub considerably.

We also recommend backing the registry with an external S3-compatible object storage like AWS S3.
It is more scalable and durable than a single local RustFS container, and it lets the registry serve layer downloads via pre-signed URLs: instead of streaming the layer through the Hub, the registry answers with an HTTP `307 Temporary Redirect` to a short-lived URL, so clients download layers directly from the object storage.
Pull bandwidth then stays off the Hub, which keeps its CPU and memory footprint low even under heavy pull load.
The redirect is enabled by default and can be turned off with `REGISTRY_S3_ALLOW_REDIRECT`.

## Networking and ports

Distr exposes two HTTP endpoints: the app (web UI and API) and the registry (OCI artifacts). Both are typically served under separate hostnames and put behind a TLS-terminating reverse proxy (Caddy in our Docker Compose setup, an Ingress controller in Kubernetes).

Regardless of how you deploy, make sure the following is in place:

- A public domain name each for the app, the registry and [metrics](/docs/self-hosting/prometheus/), pointed at the public IP of your VM or load balancer.
- Port `443` publicly reachable for HTTPS traffic.
- Port `80` publicly reachable as well if you let the reverse proxy obtain and renew TLS certificates automatically via ACME.

## Production recommendations

For production use, we recommend the following:

- Run PostgreSQL and the object storage as managed services, so you can scale, upgrade and operate them independently of the Hub.
- Run several Hub replicas behind a load balancer so the control plane stays available during upgrades and node failures. Our [Helm chart](/docs/self-hosting/kubernetes/) does this via `replicaCount` and `autoscaling`.
- Trigger the [maintenance jobs](/docs/self-hosting/maintenance/) from outside the Hub instead of using its built-in scheduler, since every replica would otherwise run every job. The Helm chart ships them as Kubernetes `CronJob`s under `cronJobs`.
- Back up the database and the object storage regularly and test your restore procedure. They hold all state there is. Back up `DATABASE_ENCRYPTION_KEY` alongside them but stored separately, since a database backup cannot be restored into a working instance without it.
- Keep the database credentials, `JWT_SECRET`, `DATABASE_ENCRYPTION_KEY` and the object storage keys in a secret manager, such as a Kubernetes `Secret`, Vault or your cloud provider's secret store, rather than in plain-text environment files.
