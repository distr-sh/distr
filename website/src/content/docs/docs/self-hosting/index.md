---
title: Self-Hosting Distr
description: Distr can be easily self-hosted in your own environment to use it as an internal software distribution platform and artifact registry.
sidebar:
  label: Introduction
  order: 0
---

While the easiest way to use Distr is to use our [hosted offering](/onboarding/), self-hosting is also an option.
The free and open source Community Edition is the perfect option to try Distr locally on your own machine.

Distr comes as a statically compiled Go binary packaged as a container image, and has minimal dependencies:

- A PostgreSQL 18+ database
- Loki for log processing
- Two S3 compatible object storage buckets for registry blobs and log chunks

Before you get started, review the [System Requirements](/docs/self-hosting/system-requirements/) to make sure your environment is sized correctly.

Check out our [Docker Compose](/docs/self-hosting/docker/) or [Kubernetes](/docs/self-hosting/kubernetes/) deployment options, or find out more information about the inner workings of Distr at [`github.com/distr-sh/distr`](https://github.com/distr-sh/distr/).

## Configuration

Whichever way you deploy it, Distr is configured entirely through environment variables, in the `.env` file next to
the Compose file or under `distr.env` in the Helm values. Only `DATABASE_URL`, `JWT_SECRET`,
`DATABASE_ENCRYPTION_KEY`, `DISTR_HOST` and `LOKI_URL` are needed to start it. Everything else either has a default or
belongs to a feature you turn on, such as the OCI registry, OIDC sign-in, outgoing mail, custom domains or the
maintenance jobs.

The [Configuration Reference](/docs/self-hosting/configuration/) lists every variable, grouped by area, and is the
place to look up what a setting does and when it is required.

## Third-Party Software

Distr itself is licensed under the Apache License 2.0. Depending on the Compose stack and the Helm values, a deployment
can also run PostgreSQL, RustFS, Loki and Caddy next to it, each as an unmodified image and as a separate program:
Distr talks to PostgreSQL, RustFS and Loki over the network, and Caddy is a reverse proxy in front of Distr. Their
licenses apply to them alone.

[Grafana Loki](https://github.com/grafana/loki) is licensed under the
[GNU Affero General Public License v3](https://github.com/grafana/loki/blob/main/LICENSE) (AGPL-3.0-only), with the
exceptions listed in [Loki's licensing guide](https://github.com/grafana/loki/blob/main/LICENSING.md). Distr neither
includes nor modifies Loki: the stacks pull the official `grafana/loki` image, which runs as a program of its own, so
the AGPL applies to Loki alone, and its clause on offering software over a network concerns modified versions only. A
Loki instance that you run yourself is licensed under the AGPL as well. To keep Loki out of your stack, point
`LOKI_URL` and the `LOKI_*` authentication variables at a Loki that is operated separately or at a hosted
Loki-compatible service, and set `loki.enabled` to `false` in the Helm chart or remove the `loki` service and the
`loki` entry under `depends_on` of the `distr` service from the Compose stack.

Every bundled component, Go module and npm package is listed with its license in
[`THIRD_PARTY_LICENSES.md`](https://github.com/distr-sh/distr/blob/main/THIRD_PARTY_LICENSES.md), and the attribution
notices those licenses require are collected in [`NOTICE`](https://github.com/distr-sh/distr/blob/main/NOTICE). The
Distr and Distr Controller images contain `NOTICE` and Distr's `LICENSE` in `/usr/share/doc/distr/`, and the Distr image
also contains the full license texts of the Go modules and npm packages in `/usr/share/doc/distr/third-party-licenses.txt`.

## Semantic Versioning

We are using [semantic versioning](https://semver.org/) for the releases of Distr, the Distr Controllers and the Distr SDKs.

## Changelog

See the [changelog](/changelog/) for a list of all releases and the changes they include.

## Self-Hosting a Paid Plan

Every plan can also be self-hosted, with the same feature set as on Distr Cloud (see [Choosing a Plan](/docs/subscription/)).
Paid plans run the Distr Enterprise image, which unlocks the plan its license key was issued for, so [get in touch](/contact/) if you are interested.
How to obtain the image and where to put the license key is described for
[Docker Compose](/docs/self-hosting/docker/#distr-enterprise) and for the
[Helm chart](/docs/self-hosting/kubernetes/#distr-enterprise).

Reference setups for a paid plan ship with the repository, as Compose stacks under
[`deploy/docker`](https://github.com/distr-sh/distr/tree/main/deploy/docker) and as Helm values files
under [`deploy/charts/distr/examples`](https://github.com/distr-sh/distr/tree/main/deploy/charts/distr/examples).
One of each runs everything on a single VM or cluster, and two run Distr against a managed database
and managed object storage on AWS or GCP. The [Docker Compose](/docs/self-hosting/docker/#running-in-production)
and [Kubernetes](/docs/self-hosting/kubernetes/#running-in-production) pages walk through both clouds.
