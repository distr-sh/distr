---
title: 'Distr v1.1 is Here!'
description: 'All the details about the inaugural release of Distr v1.0'
publishDate: 2025-02-17
lastUpdated: 2025-02-17
slug: 'distr-v1-release-post'
authors:
  - name: 'Jake Page'
    role: 'Software Distribution Specialist'
    image: '/src/assets/blog/authors/jpage.jpg'
    linkedIn: https://www.linkedin.com/in/jakepage91/
    gitHub: https://github.com/jakepage91
image: '/src/assets/blog/2025-02-17-distr-1-1-release/blog-thumb.png'
tags:
  - Releases
---

The inaugural release of [Distr](https://github.com/glasskube/distr) v1.0 shipped just a little over a week ago, and we're already back with a minor update that packs major features! These updates expand Distr's functionality in valuable ways, making it an even more powerful tool for your software distribution needs.

### 1. Feature: GitHub Release Action

This action creates a new version of a [Distr](https://github.com/glasskube/distr) application, fully automating the software release lifecycle.

- **Why it matters:** Say goodbye to manual steps! With this action, you can ship new docker and helm bases application versions automatically from end to end, making them instantly available on Distr for deployment. It's all about tying the loose ends of your CI/CD workflow, saving you time and effort.

- **How to use it:** Ready to dive in? Check out the [action documentation](https://github.com/marketplace/actions/distr-create-version#usage) for more information to help you get started.

Here is an example of how to use it to create a new version of a Docker based application

![docker-based-image](/src/assets/blog/2025-02-17-distr-1-1-release/docker-github-action.png)

### 2. Feature: Docker environment variable Management

This feature enables and simplifies the management of environment variables in your Docker Compose setup, ensuring a smoother and more secure deployment process.

- **Why it matters:** Managing environment variables can be tricky, especially in complex applications. With this feature, you can centralize and simplify env management, reducing errors and making it easier to maintain consistency across different environments.

- **How to get started:** Check out the [guide](https://distr.sh/docs/guides/docker-env-var-template/) on how to manage environment variables in Docker Compose files.

![env-management](/src/assets/blog/2025-02-17-distr-1-1-release/docker-env-management.png)

Have questions or feedback? Share your thoughts on [Discord](https://discord.gg/STk5Z3nFmT) or join request a [demo call](https://cal.glasskube.com/team/gk/demo?duration=30) here!

### Upgrading to v1.1

Our hosted version of Distr was already upgraded to the latest v1.1 release.

To take advantage of all new features in Distr v1.1, here's what you need to do **if you're using the previous version of the agents**:

- If you're running an older agent version, click update agent via the deployment web UI to ensure compatibility with the latest improvements.

If you are self-hosting Distr make sure to check out our [changelog](https://github.com/glasskube/distr/blob/main/CHANGELOG.md).

### What's Coming Next

We're just getting started! Based on user feedback and feature requests, we are currently working on licensing and private repository support. This is how it will look like:

- Each license is uniquely named within its organization and can optionally include an expiration date (agent can revoke access to software is if there is now renewal)
- Licenses are tied to a specific application and can apply either to all versions or only to selected ones.
- Deployments can be linked to a license. If the license is activated, customers see only applications and versions they have a license for
- For organizations using private container registries, licenses can also include Docker login credentials—such as a registry URL, username, and password—allowing the agent to authenticate and pull images securely.

Would love to get your thoughts on the licensing feature and hear what else you would like us to ship next.

### Join the Conversation

We'd love to hear your thoughts on this release! Share your feedback, ideas, or questions:

- Join the [Discord server](https://discord.gg/STk5Z3nFmT)
- [Book a demo call](https://cal.glasskube.com/team/gk/demo?duration=30)
- Reach out on [LinkedIn](https://www.linkedin.com/company/glasskube) or Twitter

### Thank You!

Thank you for being part of the Distr community. We're excited to continue building the best Open Source Software Distribution Platform together.
