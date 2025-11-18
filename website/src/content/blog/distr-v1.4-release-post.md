---
title: 'Distr v1.4 is here!'
description: 'All the details about Distr v1.4'
publishDate: 2025-03-25
lastUpdated: 2025-03-25
slug: 'distr-v1.4-release-post'
authors:
  - name: 'Jake Page'
    role: 'Software Distribution Specialist'
    image: '/src/assets/blog/authors/jpage.jpg'
    linkedIn: https://www.linkedin.com/in/jakepage91/
    gitHub: https://github.com/jakepage91
image: '/src/assets/blog/2025-03-25-distr-1-4-release/thumb.png'
tags:
  - Releases
---

# Distr v1.4 is Live!

Hey there! We listened to your feedback, and we're back with a massive new feature.

This one was inspired by some interesting insights into the real-world use cases a considerable amount of vendors told us they were up against, and it's why we're shipping **v1.4** today. It’s a minor version bump, but it unlocks new possibilities for advanced self-managed distribution.

Each vendor use case we come across is different, so with **v1.4** we aim to give software vendors even more control over **what** they ship and **how** they ship it.

Want a hint? It’s all about **artifacts**, and a whole new way to manage them.

---

## 🚀 What's New: Distr’s Built-in OCI Artifact Registry (Beta)

We are rolling out the **beta version of the Distr OCI Registry**. Initially enabling the feature to users who specifically asked for it, but if you’re interested, just let us know, and we’ll unlock it for you.

### Why it matters

Self-managed deployments often require strict control over what software gets deployed, where, and by whom. That’s where Distr’s new **OCI Artifact Registry** comes in.

With the Distr OCI registry, software vendors can now publish and distribute not just container images, but also **Helm charts, SBOMs, policy modules**, and other **OCI-compliant artifacts**, all managed natively within your Distr account.

### This means software vendors get

- ✅ **Granular access control**: Define exactly which customer can pull which artifacts and versions
- 🧩 **Multi-artifact support**: Distribute Helm charts, images, config bundles, and any artifact that is OCI-compliant
- 🔐 **Software supply chain security**: Store SBOMs and signatures for safer distribution
- 📊 **Visibility**: Track where, how, and by whom artifacts are consumed
- 📦 **Open standards**: Built on the OCI spec and compatible with existing tooling (Docker, ORAS, etc.)

![Diagram showing OCI registry architecture](/src/assets/blog/2025-03-25-distr-1-4-release/diagram-dark.png)

Whether you're distributing a single binary or an entire application stack, the new registry gives you the building blocks to do it **securely**, **reliably**, and in an **OCI-compliant** way, inside your end-customers’ networks.

## Watch the video to see how it works

<div style={{position: 'relative', width: '100%', paddingTop: '56.25%'}}>
  <iframe
    sandbox="allow-same-origin allow-scripts allow-popups allow-forms allow-downloads allow-storage-access-by-user-activation"
    frameBorder="0"
    title="embed"
    loading="lazy"
    src="https://www.youtube.com/embed/gM9WCx61owQ?si=QEccDX-VyV3XMiQI"
    allowFullScreen
    style={{
      position: 'absolute',
      top: 0,
      left: 0,
      width: '100%',
      height: '100%',
      border: 'none',
    }}
  />
</div>

#### 👉 Want to access the OCI Registry feature? Let us know via email at support@glasskube.com!

---

## What’s Coming Next

This is just the start! Here's a peek at what we’re working on for upcoming releases:

- Built-in vulnerability scanning and security checks
- Display audit trails
- UI improvements for better overall usability

Follow us on [LinkedIn](https://www.linkedin.com/company/glasskube) to stay up to date on the latest Distr news.

---

## Join the Conversation

We’d love to hear your thoughts on this release! Share your feedback, ideas, or questions:

- Join the [Discord server](https://discord.gg/STk5Z3nFmT)
- [Book a demo call](https://cal.glasskube.com/team/gk/demo?duration=30)
- Reach out on [LinkedIn](https://www.linkedin.com/company/glasskube) or [Twitter](https://twitter.com/glasskube)

---

## Thank You!

Thank you for being part of the Distr community. We’re excited to continue building the best **Open Source Software Distribution Platform** together.
