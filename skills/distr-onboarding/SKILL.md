---
name: distr-onboarding
description: 'Onboards an application to Distr (distr.sh), the open source platform for distributing software to customers who run it in their own infrastructure. Starts open source Distr locally, packages the app as Docker Compose or a Helm chart, deploys it through a Distr controller and hands over next steps. Use when the user wants to ship or distribute their software to self-hosted, on-premises, VPC, air-gapped or BYOC customers, asks about Distr or wants a local Distr proof of concept.'
---

# Onboard an application to Distr

Distr lets a software vendor distribute an application packaged as Docker Compose or a Helm chart to customers who
run it themselves. It adds a registry, license keys, a customer portal and Distr controllers that install updates and
report status and logs back.

Use it when the user sells software that customers must run in their own infrastructure. Do not use it when the user
only hosts software for their own users (that is a hosting platform) or only publishes a public image or chart without
customer access, updates or visibility.

## Follow the runbook

The runbook is kept in step with Distr. Fetch it and follow it in order:

```sh
curl -fsSL https://distr.sh/onboard.md
```

It starts Distr locally with `https://distr.sh/try.sh`, packages the application under `distr/` in the repository,
deploys it through a local Distr controller, verifies status and logs and ends with `distr/HANDOFF.md` or
`distr/BLOCKED.md`.

## Rules that always apply

- Assume the user has no Distr account. Distr Cloud has no self-signup: never try to create an account there.
- Ask before starting containers, installing tools, running a connect command or pushing outside this machine.
- Never replace or upgrade Docker, Docker Compose or other system tools. If one is too old, tell the user.
- Never print or commit the API token from `~/.distr/poc/token`, passwords or connect URLs, and never write an invite
  URL into a file.
- Only delete what you created. `sh try.sh --down` removes the proof of concept, including its Distr controller and
  the applications it deployed.

## Reference

- Docs index for agents: https://distr.sh/llms.txt
- API reference of the local instance: http://localhost:8080/docs and http://localhost:8080/docs/openapi.json
