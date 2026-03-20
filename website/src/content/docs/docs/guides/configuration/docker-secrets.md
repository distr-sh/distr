---
title: Configure Docker Secrets
description: Configure secrets in Docker Compose to securely store and manage sensitive information such as passwords, API keys, and certificates.
slug: docs/guides/docker-secrets
sidebar:
  order: 5
---

If your application requires access to sensitive information such as passwords, API keys, or certificates, you can use Docker secrets to securely pass them to your application without potentially leaking them via environment variables.

## How to use Docker secrets

To configure secrets in Docker Compose, you can use the `secrets` section of the `compose.yaml` file. For example:

```yaml
services:
  backend:
    image: my-application-backend
    secrets:
      - api-key.txt
    environment:
      API_KEY_PATH: /run/secrets/api-key.txt

secrets:
  api-key.txt:
    environment: API_KEY
```

In this example, the `backend` service is configured to use the `api-key.txt` secret.
The secret is defined in the `secrets` section of the `compose.yaml` file and is mounted as a file called `/run/secrets/api-key.txt` in the container.
It is populated from the environment variable `API_KEY`, which can be set in the `.env` file:

```shell
API_KEY="secret_value"
```

For more details about `.env` files in Docker Compose, see our guide on [Docker Environment Variables](/docs/guides/docker-env-var-template).

## Using Distr Secrets

You can also use [Distr secrets](/docs/guides/secrets) in your Compose file by referencing them in your `.env` file:

```shell
API_KEY="{{ .Secrets.API_KEY }}"
```

The `.env` file syntax used by Docker does not support values spanning multiple lines.
To work around this limitation, if your secret contains newline characters, Distr escapes them automatically.
However, be aware that Docker Compose only supports such shell escape sequences in double quoted values.

To learn more about secrets in Distr, check out our [dedicated guide](/docs/guides/secrets).
