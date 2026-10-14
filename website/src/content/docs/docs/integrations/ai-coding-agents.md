---
title: Onboard with an AI Coding Agent
description: Let Claude Code, Codex, Cursor or any other coding agent start a local Distr proof of concept, package your application and deploy it through a Distr controller.
sidebar:
  label: AI Coding Agents
  order: 1
---

AI coding agents can onboard your application to Distr on their own.
They start open source Distr on your machine, package your application as Docker Compose or a Helm chart, deploy it through a local Distr controller and hand over the next steps.

## Starting the onboarding from your coding agent

Open your application's repository in your coding agent and paste:

```text
Follow https://distr.sh/onboard.md to onboard this application to Distr.
```

The agent asks before it starts containers or installs tools.
When it is done, `distr/HANDOFF.md` in your repository describes what runs where.

You can also install the Distr skill, which your agent picks up when you ask it to distribute your software to self-hosted customers:

```shell
npx skills add distr-sh/distr --skill distr-onboarding
```

## What the coding agent does

1. It reads your repository and chooses Docker Compose or Helm.
2. It runs [`try.sh`](https://distr.sh/try.sh), which starts the open source Distr quickstart as the Docker Compose project `distr-poc` and creates an admin user, an API token and the registry slug `poc`. For Helm it also starts a local minikube cluster.
3. It writes the packaging to `distr/`, proves that it runs, uploads it to Distr and deploys it through a [Distr controller](/docs/controllers/) on your machine, which installs the application and reports status and logs back to Distr.
4. It checks the deployment status and logs, fixes the packaging if needed and writes `distr/HANDOFF.md`.

Log in at `http://localhost:8080` as `poc@example.com` to look around; the password is in `~/.distr/poc/credentials`.
If port 8080 is taken, the agent starts Distr on another port with `DISTR_PORT` and tells you where it runs.
Stop and remove the proof of concept with `curl -fsSL https://distr.sh/try.sh | sh -s -- --down`.

## Anonymous progress events

`try.sh` and the helper it installs send anonymous progress events (started, Distr ready, packaged, deployed, completed or blocked) so that we can see whether onboarding works.
They contain no application, repository or personal data.
Set `DO_NOT_TRACK=1` or `DISTR_TELEMETRY=0` to turn them off.

## Next steps after the proof of concept

Once the proof of concept runs, you can keep self-hosting Distr on your own server: the Community Edition stays free, and an Enterprise license adds license keys, RBAC, alerts and support bundles.
Or move to Distr Cloud with a free 30 day trial.
Both start with a short [onboarding](/onboarding/).
When you have access, ask your coding agent to apply `distr/` to the new instance.

## Machine-readable docs for agents and tools

- [`llms.txt`](https://distr.sh/llms.txt) lists every docs page as Markdown, and [`llms-full.txt`](https://distr.sh/llms-full.txt) contains all of them in one file.
- Every docs page is also available as Markdown by appending `.md` to its path, for example [`/docs/quickstart.md`](https://distr.sh/docs/quickstart.md).
- The onboarding runbook is [`/onboard.md`](https://distr.sh/onboard.md).
