---
title: Docker Compose
description: Deploy Distr in minutes using Docker Compose with minimal configuration and automatic database setup.
sidebar:
  label: Docker Compose
  order: 2
---

The easiest way to get started hosting your own Distr Hub instance is with Docker Compose.
For this, you need a working installation of Docker, as well as the Docker Compose plugin, **ideally version 5.3.0 or later** (see below for details and workarounds).

First, download and unpack the Distr Docker Compose deployment manifest from the latest release:

```shell
mkdir distr && cd distr && curl -fsSL https://github.com/distr-sh/distr/releases/latest/download/deploy-docker.tar.bz2 | tar -jx
```

This command creates a new directory called `distr` containing two files: `docker-compose.yaml` and `.env`.
For a basic setup, you don't have to modify `docker-compose.yaml`, but please open `.env` in your favorite text editor and change the values of `POSTGRES_PASSWORD` and `JWT_SECRET`.
Feel free to also change the value of `DISTR_HOST`, if you intend to make your instance publicly available.
Once you are happy with your configuration, simply start the Hub using Docker Compose:

```shell
docker compose up -d
```

> If you are using the legacy standalone distribution of Docker Compose, you may need to use `docker-compose up -d` instead.

## Older Docker Compose versions

The shipped `docker-compose.yaml` provisions the object storage bucket for the Loki log processing backend with a `pre_start` lifecycle hook on the `loki` service, which requires Docker Compose ≥ 5.3.0.
If you cannot upgrade Docker Compose yet, replace the `pre_start` hook with a one-shot service and let the `loki` service depend on its successful completion:

```yaml
services:
  create-loki-bucket:
    image: 'rclone/rclone:1.74.4'
    restart: 'no'
    environment:
      RCLONE_CONFIG_STORAGE_TYPE: s3
      RCLONE_CONFIG_STORAGE_PROVIDER: Other
      RCLONE_CONFIG_STORAGE_ENDPOINT: 'http://storage:9000'
      RCLONE_CONFIG_STORAGE_ACCESS_KEY_ID: '${REGISTRY_S3_ACCESS_KEY_ID}'
      RCLONE_CONFIG_STORAGE_SECRET_ACCESS_KEY: '${REGISTRY_S3_SECRET_ACCESS_KEY}'
    command: ['mkdir', 'storage:loki']
    depends_on:
      storage:
        condition: service_healthy

  loki:
    # ... keep the shipped service definition, but remove the pre_start section
    # and replace depends_on with:
    depends_on:
      create-loki-bucket:
        condition: service_completed_successfully
```
