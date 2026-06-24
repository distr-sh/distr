---
title: Self-Hosting Distr
description: Distr can be easily self-hosted in your own environment to use it as an internal software distribution platform and artifact registry.
sidebar:
  label: Introduction
  order: 0
---

While the easiest way to use Distr is to use our [hosted offering](/onboarding/), self-hosting is also an option.
Distr comes as a statically compiled Go binary, packaged as a container image and has minimal dependencies:

- A PostgreSQL database
- An S3 compatible object storage (only if you want to use the Distr artifacts registry)

Check out our [Docker Compose](/docs/self-hosting/docker/) or [Kubernetes](/docs/self-hosting/kubernetes/) deployment options, or find out more information about the inner workings of Distr at [`github.com/distr-sh/distr`](https://github.com/distr-sh/distr/).

## Semantic Versioning

We are using [semantic versioning](https://semver.org/) for the releases of Distr Hub, Distr Agents and Distr SDKs.

## Changelog

See the [changelog](/changelog/) for a list of all releases and the changes they include.

## Enterprise Support

We also offer enterprise support for self-hosting Distr Pro in your own environment.
If you are interested, make sure [to get in touch](/contact/).
