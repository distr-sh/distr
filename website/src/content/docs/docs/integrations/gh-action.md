---
title: GitHub Actions
description: Automatically create new application versions in Distr whenever you push a release to GitHub, supporting both Docker and Helm applications.
sidebar:
  order: 5
---

## distr-create-version-action

Distr offers the [distr-create-version-action](https://github.com/distr-sh/distr-create-version-action) GitHub Action that allows you to automatically create new versions of your application in Distr every time you push a new release.
It supports both Docker and Helm applications.

### Key Features

- **Automatic Version Creation** - Create new application versions in Distr on every release
- **Automatic Deployment Updates** - Optionally update all customer deployments to the new version
- **Docker Support** - Upload Docker Compose files with environment variable templates
- **Helm Support** - Reference Helm charts from OCI or traditional repositories
- **Multi-Organization** - Deploy to multiple Distr instances or organizations in parallel

### Quick Example

```yaml
name: Push Distr Application Version

on:
  push:
    tags:
      - '*'

jobs:
  push-to-distr:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: distr-sh/distr-create-version-action@v1
        with:
          api-token: ${{ secrets.DISTR_API_TOKEN }}
          application-id: ${{ vars.DISTR_APPLICATION_ID }}
          version-name: ${{ github.ref_name }}
          compose-file: ${{ github.workspace }}/docker-compose.yaml
          update-deployments: true
```

### Complete Setup Guide

For a comprehensive step-by-step guide including:

- Creating your application in Distr
- Setting up Personal Access Tokens
- Configuring GitHub secrets and variables
- Setting up automatic deployment updates
- Troubleshooting and advanced scenarios

See the **[Automatic Deployments from GitHub](/docs/guides/automatic-deployments-from-github/)** guide.

### Additional Resources

- [GitHub Action README](https://github.com/distr-sh/distr-create-version-action/blob/main/README.md) - Complete action documentation
- [hello-distr Example](https://github.com/distr-sh/hello-distr) - Real-world implementation example
- [Personal Access Tokens](/docs/integrations/personal-access-token/) - How to create API tokens
