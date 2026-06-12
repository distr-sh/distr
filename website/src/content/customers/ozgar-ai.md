---
company: 'Ozgar AI'
person:
  name: 'Daniel Kasen'
  role: 'Chief Engineer for Customer Success'
  image: '/src/assets/customers/ozgar-ai/daniel-kasen.jpeg'
quote: 'We went from hands-on Docker setup calls to an install flow that can be running in minutes.'
industry: 'Enterprise AI'
useCase: 'On-Prem Deployment'
featured: true
outcome: 'From guided setup to scalable delivery'
caseStudy:
  logo: '/src/assets/customers/ozgar-ai/logo-black.svg'
  logoLight: '/src/assets/customers/ozgar-ai/logo-black.svg'
  logoDark: '/src/assets/customers/ozgar-ai/logo-white.svg'
  pageTitle: 'Ozgar AI Case Study'
  pageDescription: 'How Ozgar AI uses Distr to move from setup calls to repeatable on-prem installs and ongoing support.'
---

## Challenge

[Ozgar AI](https://ozgar.ai) helps teams and AI tools understand complex applications through trusted, source-linked context. Built for IBM i, IBM z, and other complex enterprise environments, Ozgar connects code, data, documentation, jobs, and business logic into a unified knowledge layer that improves understanding, accelerates change, and reduces risk. When teams are navigating hundreds of thousands of lines of code across multiple services, Ozgar AI helps engineers explore unfamiliar codebases, trace dependencies and generate documentation, without spending days reading through source files.

Because Ozgar AI works directly with source code, the customers who want it most are often the ones who cannot send that code anywhere. Enterprise engineering teams with internal security requirements, compliance obligations and large proprietary codebases need the platform to run inside their own infrastructure. From the start, the deployment experience was going to be part of the product experience.

In the early rollout, getting a customer live meant sending a link and API key, scheduling a call and walking through Docker setup step by step. That worked for early users with strong technical teams, but it introduced a ceiling. Every new enterprise customer was its own project: different environments, different IT configurations, different levels of Docker familiarity. The team was the connective tissue across all of them.

"**We went from hands-on Docker setup calls to an install flow that can be running in minutes,**" says Daniel Kasen, Chief Engineer for Customer Success at Ozgar AI.

For a team focused on product velocity, that time had to go somewhere else. The team needed a path that could:

- Turn onboarding into a repeatable install flow
- Keep self-hosted deployments visible after install
- Reduce custom support work across customers
- Give customers a self-serve path to deploy without relying on the team for every install

## Solution

Ozgar AI adopted Distr to standardize how it delivers its on-prem offering. Instead of treating each install as a custom project, Distr gives Ozgar AI one workflow for creating deployment targets, connecting agents, shipping updates and checking customer environments after rollout.

The team packages Ozgar AI as a Docker Compose deployment and ships it through Distr. When a new enterprise customer is ready to go live, they log into the Distr customer portal, configure the application and set their environment variables through the UI. From there, they get a single setup command that handles the full installation. The Distr agent connects their environment back to Ozgar AI's platform, and the team has the visibility it needs from that point on without ever requiring direct access to the customer's infrastructure.

**How they use Distr:**

- **Guided onboarding via the customer portal:** Customers configure their application and environment variables through the Distr Customer Portal, then run a single command that sets everything up. No scheduled call required.
- **Deployment agents:** Distr agents connect customer environments back to Ozgar AI's deployment workflow, so the team can manage releases without needing direct access to the customer's infrastructure.
- **Logs and health visibility:** The team can see whether deployments are healthy and inspect logs when something needs attention, without asking the customer to dig through their own infrastructure.
- **Support bundles:** When a customer needs help, support data can be collected through Distr instead of relying on long back-and-forth debugging sessions across environment boundaries.
- **Versioned updates:** Ozgar AI can keep deployments current with a consistent release process while customers stay in control of the version they run.

Distr gave Ozgar AI the operational tooling out of the box, from deployment agents to logs and support bundles, so the team could keep supporting self-hosted deployments without turning every account into a custom operations project.

## Result

The shift was most visible in what no longer had to happen: setup calls scheduled around environment differences, debugging sessions that required customers to dig through their own servers, manual tracking of which version each customer was running.

- **Up and running in minutes, supported at scale:** The guided install path is repeatable across customers, so onboarding no longer depends on a bespoke call for every deployment.
- **Full visibility without direct access:** Deployment agents, logs, support bundles and release controls give the team what it needs to operate and support every customer environment.
- **On-prem delivery that scales:** The same platform covers the first install, updates and ongoing support, so the customer base can grow without operational overhead growing with it.

The result is a self-hosted offering that behaves like a managed product: customers get a clean, guided install path and can run Ozgar AI in their environment, while the team retains the visibility it needs to keep every deployment healthy.
