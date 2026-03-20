---
title: Docker Compose
description: Deploy Distr in minutes using Docker Compose with minimal configuration and automatic database setup.
sidebar:
  order: 1
---

The easiest way to get started hosting your own Distr Hub instance is with Docker Compose.
For this, you need a working installation of Docker, as well as the Docker Compose plugin.

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
