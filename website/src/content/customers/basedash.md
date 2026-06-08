---
company: 'Basedash'
person:
  name: 'Derek Reynolds'
  role: 'Product Engineer'
  image: '/src/assets/customers/basedash/derek-reynolds.jpeg'
quote: 'Having a dedicated space for all our self-hosted customers that can manage authenticated registry access is great.'
industry: 'Developer Tools'
useCase: 'Self-Hosted Deployment'
featured: true
outcome: 'One place for every self-hosted customer'
caseStudy:
  logo: '/src/assets/customers/basedash/logo-light.svg'
  pageTitle: 'Basedash Case Study'
  pageDescription: 'How Basedash uses Distr to deliver and manage self-hosted deployments for their customers'
---

## Challenge

[Basedash](https://basedash.com) is AI-native business intelligence: teams use natural language to generate dashboards, reports, insights, and charts in seconds—no SQL required. Many of their customers run Basedash in their own environments for data control and compliance. Supporting those self-hosted deployments used to mean handing out registry credentials manually, keeping spreadsheets of who had access to what, and answering the same setup questions over and over.

"**We needed one place where every self-hosted customer could get in, grab what they need, and deploy without us having to send tokens or run custom scripts,**" says Derek Reynolds, Product Engineer at Basedash.

The team was looking for:

- A single platform where customers could manage their own deployment targets and credentials
- Private container registry with fine-grained access control
- Flexibility: some customers want a fully-managed feeling or self-hosting assistance; others already have their own bespoke setup for deploying self-hosted apps and just want artifact access.

## Solution

Basedash chose Distr to run their self-hosted distribution. Distr supports all deployment use cases out of the box—from fully-managed agent-based deployments where you can push updates directly into customer infrastructure to teams who pull images from the registry themselves—and works for self-hosted customers at every level. Most importantly, it gives Basedash one central place for all their self-managed customers. Distr is the "dedicated space" their team and customers use every day.

**How they use it:**

- **Agent deployment (recommended):** Customers install the Distr agent with a single command from the customer portal. The agent pulls images, runs Docker Compose, and reports status. Basedash gets automatic updates and health visibility without touching customer servers.
- **Container deployment:** Teams that already have a setup for deploying self-hosted apps can pull images from the Distr registry using a PAT. They get full control over when and how to deploy.
- **One place for all self-hosted customers:** Whether a customer uses the agent or pulls images themselves, every self-hosted deployment is visible and manageable from the same platform.

Basedash documents the full flow—agent install, registry auth, and machine specs—in their [self-hosting deploy guide](https://docs.basedash.com/self-hosting/deploy), so customers can get up and running without back-and-forth.

## Result

Basedash's self-hosted offering now runs through Distr. Customers get a clear path to deploy (agent or registry), and the team has a single place to manage self-hosted customers.

- **Less manual work:** No more ad-hoc token generation or credential spreadsheets. The Distr platform handles it.
- **Better security:** Each target has its own credentials. Revoking access or rotating secrets is straightforward.
- **Fewer support loops:** The customer portal gives self-hosted customers a place to generate access tokens and get installation commands on their own—no back-and-forth with the Basedash team.

For a product that ships both as SaaS and self-hosted, having a dedicated space for self-hosted customers has made the whole process simpler for everyone.
