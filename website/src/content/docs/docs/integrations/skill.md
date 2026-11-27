---
title: Distr Skill
description: Install the Distr skill so that coding agents like Claude Code, Cursor, Codex and GitHub Copilot know how to use the Distr API. It is updated with every Distr release.
slug: docs/integrations/skill
sidebar:
  label: Skill
  order: 3
---

The Distr skill teaches coding agents how to work with Distr: where to find the current documentation and how to call the [Distr API](/docs/integrations/api/).
It follows the [Agent Skills](https://agentskills.io) format, so Claude Code, Cursor, Codex, GitHub Copilot, Gemini CLI and most other agents can load it.

The skill is released together with Distr.
Every Distr release publishes a new version of it, and the installation methods below pick that version up when you update.

## Install with the skills CLI

The [skills CLI](https://github.com/vercel-labs/skills) installs the skill for every agent it detects on your machine:

```shell
npx skills add https://distr.sh --skill distr --global
```

It reads the skill from the discovery index on `https://distr.sh/.well-known/agent-skills/index.json` and checks the digest of the archive before it installs anything.
Omit `--global` to install the skill into the current project instead.

To update the skill, run:

```shell
npx skills update distr --global
```

## Install with the GitHub CLI

The [GitHub CLI](https://cli.github.com/manual/gh_skill_install) installs the skill from the latest Distr release on GitHub:

```shell
gh skill install distr-sh/distr skills/distr --agent claude-code --scope user
```

Set `--agent` to the agent you use, for example `claude-code`, `cursor`, `codex` or `github-copilot`, and repeat the command for every further agent.
Without `--agent`, the GitHub CLI asks for the agents in a terminal and installs for GitHub Copilot alone everywhere else.

To update the skill, run:

```shell
gh skill update distr
```

## Install as a Claude Code plugin

Add the Distr marketplace to Claude Code and install the plugin from it:

```shell
claude plugin marketplace add https://distr.sh/.well-known/agent-skills/marketplace.json
claude plugin install distr@distr
```

To update the plugin, run `claude plugin marketplace update distr` followed by `claude plugin update distr@distr`, or enable automatic updates for the marketplace in the `/plugin` menu of Claude Code.

## Keep the skill up to date with topgrade

[topgrade](https://github.com/topgrade-rs/topgrade) updates skills along with everything else on your machine.
It runs the update command of each installation method above in its own step:

| Installation method | topgrade step         | Command it runs                   |
| ------------------- | --------------------- | --------------------------------- |
| skills CLI          | `skills`              | `npx skills update --global`      |
| GitHub CLI          | `github_cli_skills`   | `gh skill update --all`           |
| Claude Code plugin  | `claude_code_plugins` | `claude plugin update` per plugin |

The steps are enabled by default and skip themselves when the tool they need is not installed.
The GitHub CLI step requires topgrade 17.14 or later.
topgrade only updates skills installed with `--global`, since it does not run inside your projects.
