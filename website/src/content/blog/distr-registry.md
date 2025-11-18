---
title: 'Distr Launch Week: The Distr Artifact Registry is GA'
description: 'A built-in OCI registry with tag-based access control, audit logs, and support for Helm, Docker & more, designed to simplify self-managed software distribution for both vendors and customers.'
publishDate: 2025-05-22
lastUpdated: 2025-05-22
slug: 'distr-registry'
authors:
  - name: 'Jake Page'
    role: 'Software Distribution Specialist'
    image: '/src/assets/blog/authors/jpage.jpg'
    linkedIn: https://www.linkedin.com/in/jakepage91/
    gitHub: https://github.com/jakepage91
image: '/src/assets/blog/2025-05-22-distr-registry/registry-thumbnail.png'
tags:
  - OCI Registry
  - distr-launch-week
---

Today, we are happy to announce that the Distr [artifact registry](https://distr.sh/docs/product/registry/) is now Generally Available (GA).

Manage and distribute container images, Helm charts, and other OCI artifacts with license-based access control, audit logs, and a visual UI that makes self-managed software distribution pretty darn simple if we do say so ourselves, built for both you and your customers.

<hr>

## Why we built a registry into Distr

Distr was originally built to help vendors manage software distribution in agent based [assisted self-managed](https://distr.sh/docs/use-cases/assisted-self-managed/) environments.
But as end customers increasingly requested full control over their deployments, vendors needed a way to offer a [fully self-managed model](https://distr.sh/docs/use-cases/fully-self-managed/) as well.
Other third-party registries lacked the fine-grained access controls and visibility these scenarios call for, so that's why we built an OCI-compliant registry directly into Distr, enabling both assisted and fully self-managed distribution, with features like tag-based access control and detailed download logs.

## Why is Self-Managed artifact distribution hard?

In case you are unsure whether this is a feature you need, consider this.

Let’s say a new customer is ready to deploy your self-managed solution.
Contracts signed.
Excitement is high.
But then you get a flurry of messages from your end customers along the lines of…

> "We can't access your registry."  
> "Is our firewall blocking the pull?"  
> "Which version are we supposed to use?"

Sound familiar?

Vendors often don’t have a standard way to set up credentials, provide pull instructions, and troubleshoot artifact access issues.
Customers get frustrated.
Deployment stalls.

With Distr's built-in registry, those problems are largely mitigated and planned for in advance.

## Meet the Distr Artifact Registry

Built for software vendors who serve end-customer with different [deployment appetites](https://distr.sh/docs/use-cases/fully-self-managed/), our registry brings OCI-compliant artifact distribution natively integrated into Distr.

Here are a few of its key features:

![registry-diagram](/src/assets/blog/2025-05-22-distr-registry/registry-diagram.png)

### OCI-compliant & format-flexible

The registry supports any OCI artifact, including:

- Docker images
- Helm charts
- WASM modules
- Anything else that follows the OCI spec

### License-based access control

Vendors can restrict access by using Artifacts licenses. Grant or revoke permissions to one or many artifacts at the tag level. Update instantly when entitlements change.

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="app.distr.sh">

![Distr Onboarding Tutorials](/src/assets/blog/2025-05-22-distr-registry/lw-license-dark.png)

</div>

### Visual management

Use any OCI compliant CLI for pushing and pulling, and use the Distr Registry UI to:

- View artifact versions or tags
- See which customer pulled what, and when
- Track deployment status across environments

**General Artifact UI**

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="app.distr.sh">

![General Artifact UI](/src/assets/blog/2025-05-22-distr-registry/lw-artifacts-ui-dark.png)

</div>

**Deployment tracking UI**

![Deployment tracking UI](/src/assets/blog/2025-05-22-distr-registry/lw-artifacts-dark.png)

The Downloads page gives granular artifact consumption logs

**Download audit logs UI**

<div
  className="app-frame mac dark borderless shadow--tl"
  data-url="app.distr.sh">

![Download audit logs UI](/src/assets/blog/2025-05-22-distr-registry/lw-deploy-logs-dark.png)

</div>
## What's next for the Registry?

We’re just getting started, here’s what’s coming soon:

- Integrated CVE scanning for registry artifacts
- Download history query & export
- Visual indicators and download for SBOM & image signature layers

## Give it a try today

The Distr container registry has already distributed thousands of artifact versions to end customers.
Often completely white-labeled, with a custom CNAME DNS record, as part of a vendors software supply distribution stack.

The registry is available now in all Distr accounts. Just head to the Artifacts tab in the dashboard to [get started](https://distr.sh/docs/guides/container-registry/).

- Read the [docs](https://distr.sh/docs/product/registry/)
- Contact us for a [demo](https://cal.glasskube.com/team/gk/demo)

We are excited to hear your thoughts on the registry and open to your feedback to further shape the product to make it as useful as possible for vendors and end-customers alike.
