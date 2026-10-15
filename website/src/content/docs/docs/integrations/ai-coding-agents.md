---
title: Onboard with an AI Coding Agent
description: Let Claude Code, Codex, Cursor or any other coding agent start a local Distr proof of concept, deploy Hello Distr and your application to a test customer through a Distr controller and hand over the next steps.
sidebar:
  label: AI Coding Agents
  order: 1
---

AI coding agents can onboard your application to Distr on their own.
They start open source Distr on your machine, deploy the Hello Distr example to a test customer, package your application as Docker Compose or a Helm chart, deploy it the same way through a local Distr controller and hand over the next steps.

## Starting the onboarding from your coding agent

Open your application's repository in your coding agent and paste:

```text
Follow https://distr.sh/onboard.md to onboard this application to Distr.
```

The agent asks before it starts containers or installs tools.
When it is done, `deploy/distr/HANDOFF.md` in your repository describes what runs where.

You can also install the Distr skill, which your agent picks up when you ask it to distribute your software to self-hosted customers:

```shell
npx skills add distr-sh/distr --skill distr-onboarding
```

In a checkout of the Distr repository itself, the agent reads the runbook from `website/public/onboard.md` and runs `deploy/docker/quickstart/try.sh` from the checkout instead of fetching either from distr.sh.

## What the coding agent does

1. It runs [`try.sh`](https://distr.sh/try.sh), which checks Docker, Docker Compose, the ports and the free disk space, downloads the images, starts the published open source Distr image as the Docker Compose project `distr-poc` and creates an admin user, an API token and the registry slug `poc`. For Helm it also starts a local minikube cluster.
2. It deploys Hello Distr to a test customer: an application and version, a customer, a deployment target that belongs to the customer and a [Distr controller](/docs/controllers/) on your machine, then verifies the deployment through the API, the Hello Distr health endpoint and the deployment logs.
3. It reads your repository, chooses Docker Compose or Helm, writes the packaging to `deploy/distr/`, proves that it runs, uploads it to Distr and deploys it to the test customer.
4. It checks the deployment status and logs, fixes the packaging if needed and writes `deploy/distr/HANDOFF.md`.

Log in at `http://localhost:8080` as `poc@example.com` to look around; the password is in `~/.distr/poc/credentials`.
If port 8080 is taken, the agent starts Distr on another port with `DISTR_PORT` and tells you where it runs.

The first run downloads about 700 MB of images, including Hello Distr, and can take several minutes.
Manage the proof of concept with these commands:

| What                                  | Command                                         |
| ------------------------------------- | ----------------------------------------------- |
| Start Distr again                     | `sh ~/.distr/poc/compose.sh up -d --pull never` |
| Stop Distr                            | `sh ~/.distr/poc/compose.sh stop`               |
| Show diagnostics with secrets removed | `sh ~/.distr/poc/try.sh --diagnose`             |
| Remove everything                     | `sh ~/.distr/poc/try.sh --down`                 |

`--down` removes Distr, its Distr controllers and the applications they deployed.

## Anonymous progress events

`try.sh` and the helper it installs send anonymous progress events (started, Distr ready, packaged, deployed, completed or blocked) so that we can see whether onboarding works.
They contain no application, repository or personal data.
Set `DO_NOT_TRACK=1` or `DISTR_TELEMETRY=0` to turn them off.

## Next steps after the proof of concept

Once the proof of concept runs, you can keep self-hosting Distr on your own server: the Community Edition stays free, and an Enterprise license adds license keys, RBAC, alerts and support bundles.
Or move to Distr Cloud with a free 30 day trial.
Both start with a short [onboarding](/onboarding/).
When you have access, ask your coding agent to apply `deploy/distr/` to the new instance.

## Testing the onboarding

In a checkout of the Distr repository, `mise run test:onboarding` runs `try.sh` and the Hello Distr customer deployment of the runbook exactly as an agent reads them, verifies the result and removes everything again.
It needs Docker, ports 80 and 443 for Hello Distr and no other Distr proof of concept on the machine.

## Machine-readable docs for agents and tools

- [`llms.txt`](https://distr.sh/llms.txt) lists every docs page as Markdown, and [`llms-full.txt`](https://distr.sh/llms-full.txt) contains all of them in one file.
- Every docs page is also available as Markdown by appending `.md` to its path, for example [`/docs/quickstart.md`](https://distr.sh/docs/quickstart.md).
- The onboarding runbook is [`/onboard.md`](https://distr.sh/onboard.md).
