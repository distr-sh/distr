---
company: 'Weave'
person:
  name: 'Andrew Churchill'
  role: 'Co-Founder & CTO'
  image: '/src/assets/customers/weave/andrew-churchill.jpeg'
quote: "Weave has a fully self-hosted offering. It's a huge unlock for us, but we almost didn't build it. Distr made such a huge difference in getting us there."
industry: 'AI Engineering Intelligence'
useCase: 'Self-Hosted Deployment'
featured: true
outcome: 'Self-hosted without the engineering tax'
caseStudy:
  logo: '/src/assets/customers/weave/logo.svg'
  pageTitle: 'Weave Case Study'
  pageDescription: 'How Weave delivers a fully self-hosted offering with Distr without it becoming a tax on every engineering decision'
---

## Challenge

[Weave](https://workweave.dev) is an engineering intelligence platform that measures the impact of AI on software teams, analyzing pull requests, reviews and deployments to show organizations exactly where AI is accelerating delivery and where it's adding complexity. With 20,000+ engineers across 500+ organizations, including Fortune 100 companies, many of Weave's customers require the platform to run inside their own infrastructure for security and compliance.

That meant building a fully self-hosted offering, a path Weave nearly avoided altogether.

"**I've talked to other founders who went down the road of offering on-prem and regretted it,**" says Andrew Churchill, Co-Founder & CTO at Weave. "**One told me he wishes he could take it back entirely, even after landing massive logos, because the operational and engineering overhead was way more than they expected. They essentially ended up maintaining a parallel product.**"

The team's concern was threefold:

- Self-hosted becoming a tax on every engineering decision, forcing engineers to reason about two environments for every change
- Losing the visibility and control they have in the cloud, ending up SSHing into random machines that one person set up and nobody remembers how to access
- Self-hosted deployments not keeping pace with SaaS, leaving on-prem customers stuck on stale versions while updates piled up behind manual release work

## Solution

Weave runs as a microservice architecture on Google Cloud Platform. To make self-hosting viable without maintaining a parallel product, the team packaged those same services into a single Docker Compose deployment that customers run on one VM, then used Distr to manage it.

"**A customer spins up a VM, runs a setup command, configures their keys, and from there we can manage almost everything through our platform,**" says Churchill. "**We can see logs, debug issues, ship improvements, and they can control what version they're running. There's no SSHing into random machines that one person set up and nobody remembers how to access.**"

**How they use Distr:**

- **One setup for every self-hosted customer:** Rather than tailoring each deployment to individual customer requirements, Weave ships a single standardized setup. Customers provision a VM and run one setup command. The Distr agent then takes over deployment and reporting.
- **Metrics and logs, without data leaving:** The Distr agent collects metrics and logs to give Weave the visibility they need to debug issues, while no stored customer data ever leaves the customer's environment.
- **Continuous delivery from one platform:** Weave uses the [Distr GitHub Action](/docs/integrations/gh-action/) to continuously push new commits and artifacts to Distr, so every customer fetches the latest version from a single platform, while still deciding for themselves which version they run.

Just as important was keeping self-hosted from slowing the team down. By deploying the same services in both environments, Weave avoided a parallel codebase.

"**In most cases, if something works in the cloud, it works self-hosted too,**" Churchill explains. "**Engineers don't have to think about it unless there's actually a behavioral difference.**"

## Result

Weave shipped a fully self-hosted offering that behaves like a managed product for both their customers and their own engineers, without the operational drag that pushed other founders to regret going on-prem.

- **A real unlock, not a tax:** Self-hosted opened the door to security- and compliance-sensitive customers without becoming a burden on every engineering decision.
- **Self-hosted that keeps pace with SaaS:** Continuous pushes through the GitHub Action mean on-prem customers get new commits and artifacts as fast as cloud users, instead of waiting on manual release cycles.
- **Central management:** Logs, debugging and improvements all flow through one platform instead of ad-hoc server access.
- **Customer autonomy:** Customers stay in control of their environment and the version they run.

"**Weave has a fully self-hosted offering. It's a huge unlock for us, but we almost didn't build it,**" says Churchill. "**Shoutout to the team at Glasskube, it's their product Distr that made such a huge difference in getting us there, and I think more companies exploring self-hosted offerings should know about them.**"
