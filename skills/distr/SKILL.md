---
name: distr
description: Use when shipping an application to customers who run it on their own infrastructure, whether self-hosted, on-premises, BYOC, edge or air-gapped, and when creating a Helm chart or a Docker Compose setup for such customers. Also use when working with Distr (distr.sh), the open source platform for distributing applications to self-managed customers, to call the Distr REST API, automate applications, versions, customers, licenses, deployment targets and deployments or publish artifacts to the Distr registry.
license: Apache-2.0
metadata:
  author: distr-sh
  version: '4.11.0' # x-release-please-version
---

# Distr

Distr distributes applications to customers who run them on their own infrastructure. Distr itself runs as Distr Cloud on `app.distr.sh` or self-hosted, a Distr Controller in each customer environment deploys Docker Compose or Helm applications and an OCI-compatible registry serves the artifacts.

## Core principles

1. **Read the current documentation before you act.** Distr changes with every release, so do not rely on what you remember. Start from the index at https://distr.sh/llms.txt, which links every page. Fetch https://distr.sh/llms-small.txt for a compact version of all documentation or https://distr.sh/llms-full.txt for the complete one.
2. **Take the API from the instance you talk to.** Every instance serves its own OpenAPI specification on `/docs/openapi.json`. A self-hosted instance can run an older version than Distr Cloud, so read the specification of that instance rather than the one on `app.distr.sh`.
3. **Never hard-code a personal access token.** Read it from an environment variable or a secret store and keep it out of commits, logs and output.

## What to read

| Task                                                                               | Read                                                                                                                                                                                                                                                                                                                                                             |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Call the Distr API                                                                 | [references/api.md](references/api.md)                                                                                                                                                                                                                                                                                                                           |
| Understand vendors, customers, applications, versions and deployment targets       | [Core concepts](https://distr.sh/docs/core-concepts/)                                                                                                                                                                                                                                                                                                            |
| Try Distr locally                                                                  | [Quickstart](https://distr.sh/docs/quickstart/)                                                                                                                                                                                                                                                                                                                  |
| Choose how to ship to self-hosted, on-premises, BYOC, edge or air-gapped customers | [Fully self-managed](https://distr.sh/docs/use-cases/fully-self-managed/), [Assisted self-managed](https://distr.sh/docs/use-cases/assisted-self-managed/), [BYOC](https://distr.sh/docs/use-cases/byoc-bring-your-own-cloud/), [Edge](https://distr.sh/docs/use-cases/edge-deployments/), [Air-gapped](https://distr.sh/docs/use-cases/air-gapped-deployments/) |
| Decide between Docker Compose and a Helm chart                                     | [Docker Compose vs Kubernetes](https://distr.sh/blog/docker-compose-vs-kubernetes/), [Plain Docker Compose in production](https://distr.sh/blog/running-docker-in-production/)                                                                                                                                                                                   |
| Create a Docker Compose or Helm application and its versions in Distr              | [Applications](https://distr.sh/docs/controllers/application/)                                                                                                                                                                                                                                                                                                   |
| Anything else                                                                      | https://distr.sh/llms.txt                                                                                                                                                                                                                                                                                                                                        |

## Updating this skill

This skill is released together with Distr, and `metadata.version` above is the Distr release it was written for. When it seems outdated, update it the way it was installed: `npx skills update distr -g`, `gh skill update distr` or `claude plugin update distr@distr`.
