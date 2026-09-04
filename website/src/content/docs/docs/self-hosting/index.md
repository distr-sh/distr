---
title: Self-Hosting Distr
description: Distr can be easily self-hosted in your own environment to use it as an internal software distribution platform and artifact registry.
sidebar:
  label: Introduction
  order: 0
---

While the easiest way to use Distr is to use our [hosted offering](/onboarding/), self-hosting is also an option.
The free and open source Community Edition is the perfect option to try Distr locally on your own machine.

Distr comes as a statically compiled Go binary, packaged as a container image and has minimal dependencies:

- A PostgreSQL database
- Loki for log processing
- Two S3 compatible object storage buckets for registry blobs and log chunks

Before you get started, review the [System Requirements](/docs/self-hosting/system-requirements/) to make sure your environment is sized correctly.

Check out our [Docker Compose](/docs/self-hosting/docker/) or [Kubernetes](/docs/self-hosting/kubernetes/) deployment options, or find out more information about the inner workings of Distr at [`github.com/distr-sh/distr`](https://github.com/distr-sh/distr/).

## Semantic Versioning

We are using [semantic versioning](https://semver.org/) for the releases of Distr Hub, Distr Agents and Distr SDKs.

## Changelog

See the [changelog](/changelog/) for a list of all releases and the changes they include.

## Self-Hosting a Paid Plan

The guides in this section describe the Community Edition.

Every plan can also be self-hosted, with the same feature set as on Distr Cloud (see [Choosing a Plan](/docs/subscription/)).
A license key unlocks your plan in your own environment, so [get in touch](/contact/) if you are interested.

The reference setups for a paid plan on [AWS](/docs/self-hosting/docker/#distr-on-aws) and
[GCP](/docs/self-hosting/docker/#distr-on-gcp) run the Hub against a managed database and object storage,
either with [Docker Compose](/docs/self-hosting/docker/#running-in-production) or the
[Helm chart](/docs/self-hosting/kubernetes/#running-in-production).
