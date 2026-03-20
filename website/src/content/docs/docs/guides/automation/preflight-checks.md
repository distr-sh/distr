---
title: Pre-Flight Checks
description: Validate target environments before deployment by checking Docker version, CPU, RAM, disk space and more with custom pre-flight check scripts.
slug: docs/guides/preflight-checks
sidebar:
  order: 8
---

Pre-flight checks are validation scripts that run **before** your application is deployed to a customer environment.
They verify that the target system meets the minimum requirements — such as the correct Docker version, sufficient CPU, memory, and disk space — and prevent deployments from proceeding when critical conditions are not met.

## Why Pre-Flight Checks Matter

Deploying software to customer-managed infrastructure means you don't control the environment.
Without pre-flight checks, deployments can fail silently or produce hard-to-diagnose issues:

- **Docker version mismatches** — A customer runs an outdated Docker CLI that doesn't support features your Compose file relies on (e.g., `docker compose` v2 syntax, health check options, or specific network drivers).
- **Insufficient resources** — The target machine doesn't have enough CPU cores, RAM, or free disk space to run your application, leading to OOM kills, slow performance, or failed container starts.
- **Missing dependencies** — Required tools, kernel modules, or system libraries are not installed.
- **Network issues** — The target machine cannot reach required endpoints such as container registries or external APIs.

Pre-flight checks catch these problems **before** deployment begins, giving customers clear, actionable error messages instead of cryptic container failures.

## How It Works in Distr

When pre-flight checks are enabled for your organization, the connect instructions delivered to your customers change:

- **Without pre-flight checks:** Customers receive a `docker compose` command directly.
- **With pre-flight checks:** Customers receive a shell script that:
  1. Runs your pre-flight check scripts to validate the environment.
  2. Reports any failing checks with clear error messages.
  3. Only proceeds with `docker compose up` if all checks pass.

This approach is transparent to customers — they run a single command and get immediate feedback if their environment doesn't meet requirements.

You can configure pre-flight check scripts in the **Organization Settings** under **Agents**.

> **Tip:** Pre-flight check scripts are not limited to validation. You can also use them to install or update software on the target machine — for example, ensuring a specific package is installed or upgrading a dependency to a compatible version before the deployment starts.

> **Note:** Pre-flight check scripts are only executed during the initial connect or a reconnect. A reconnect will not delete any existing deployment — it re-runs the connect scripts and then resumes the deployment.

## Example Pre-Flight Check Scripts

Below are practical shell scripts you can use as pre-flight checks.
Each script exits with a non-zero status and prints an error message if the check fails.

### Minimum Docker CLI Version

Ensures the Docker CLI meets a minimum version requirement. This is useful when your Compose file uses features introduced in a specific Docker version.

```bash
#!/bin/bash
set -e

REQUIRED_VERSION="24.0.0"

if ! command -v docker &> /dev/null; then
  echo "FAIL: Docker is not installed."
  exit 1
fi

DOCKER_VERSION=$(docker version --format '{{.Client.Version}}' 2>/dev/null || echo "0.0.0")

version_ge() {
  printf '%s\n%s' "$2" "$1" | sort -V -C
}

if ! version_ge "$DOCKER_VERSION" "$REQUIRED_VERSION"; then
  echo "FAIL: Docker version $DOCKER_VERSION is below the required minimum $REQUIRED_VERSION."
  exit 1
fi

echo "OK: Docker version $DOCKER_VERSION meets the minimum requirement ($REQUIRED_VERSION)."
```

### Minimum CPU Cores

Checks that the host has enough CPU cores to run your application.

```bash
#!/bin/bash
set -e

REQUIRED_CORES=2

AVAILABLE_CORES=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 0)

if [ "$AVAILABLE_CORES" -lt "$REQUIRED_CORES" ]; then
  echo "FAIL: Only $AVAILABLE_CORES CPU core(s) available, but at least $REQUIRED_CORES required."
  exit 1
fi

echo "OK: $AVAILABLE_CORES CPU core(s) available (minimum: $REQUIRED_CORES)."
```

### Minimum Available RAM

Verifies the system has enough total memory. Adjust the threshold based on your application's requirements.

```bash
#!/bin/bash
set -e

REQUIRED_RAM_MB=4096

if [ -f /proc/meminfo ]; then
  TOTAL_RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
  TOTAL_RAM_MB=$((TOTAL_RAM_KB / 1024))
elif command -v sysctl &> /dev/null; then
  TOTAL_RAM_BYTES=$(sysctl -n hw.memsize 2>/dev/null || echo 0)
  TOTAL_RAM_MB=$((TOTAL_RAM_BYTES / 1024 / 1024))
else
  echo "FAIL: Unable to determine available RAM."
  exit 1
fi

if [ "$TOTAL_RAM_MB" -lt "$REQUIRED_RAM_MB" ]; then
  echo "FAIL: Only ${TOTAL_RAM_MB}MB RAM available, but at least ${REQUIRED_RAM_MB}MB required."
  exit 1
fi

echo "OK: ${TOTAL_RAM_MB}MB RAM available (minimum: ${REQUIRED_RAM_MB}MB)."
```

### Minimum Free Disk Space

Checks that the target mount point (typically `/` or `/var/lib/docker`) has enough free disk space.

```bash
#!/bin/bash
set -e

REQUIRED_DISK_GB=20
MOUNT_POINT="/"

AVAILABLE_KB=$(df --output=avail "$MOUNT_POINT" 2>/dev/null | tail -1)
AVAILABLE_GB=$((AVAILABLE_KB / 1024 / 1024))

if [ "$AVAILABLE_GB" -lt "$REQUIRED_DISK_GB" ]; then
  echo "FAIL: Only ${AVAILABLE_GB}GB free disk space on $MOUNT_POINT, but at least ${REQUIRED_DISK_GB}GB required."
  exit 1
fi

echo "OK: ${AVAILABLE_GB}GB free disk space on $MOUNT_POINT (minimum: ${REQUIRED_DISK_GB}GB)."
```

## Combining Checks Into a Single Script

You can combine multiple checks into one pre-flight script.
The script runs all checks and reports all failures at once, so customers can fix everything in a single pass.

```bash
#!/bin/bash

ERRORS=0

# --- Docker version check ---
REQUIRED_DOCKER_VERSION="24.0.0"
if ! command -v docker &> /dev/null; then
  echo "FAIL: Docker is not installed."
  ERRORS=$((ERRORS + 1))
else
  DOCKER_VERSION=$(docker version --format '{{.Client.Version}}' 2>/dev/null || echo "0.0.0")
  if ! printf '%s\n%s' "$REQUIRED_DOCKER_VERSION" "$DOCKER_VERSION" | sort -V -C; then
    echo "FAIL: Docker version $DOCKER_VERSION is below the required minimum $REQUIRED_DOCKER_VERSION."
    ERRORS=$((ERRORS + 1))
  else
    echo "OK: Docker version $DOCKER_VERSION"
  fi
fi

# --- CPU check ---
REQUIRED_CORES=2
AVAILABLE_CORES=$(nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null || echo 0)
if [ "$AVAILABLE_CORES" -lt "$REQUIRED_CORES" ]; then
  echo "FAIL: Only $AVAILABLE_CORES CPU core(s), need at least $REQUIRED_CORES."
  ERRORS=$((ERRORS + 1))
else
  echo "OK: $AVAILABLE_CORES CPU core(s)"
fi

# --- RAM check ---
REQUIRED_RAM_MB=4096
if [ -f /proc/meminfo ]; then
  TOTAL_RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
  TOTAL_RAM_MB=$((TOTAL_RAM_KB / 1024))
else
  TOTAL_RAM_MB=0
fi
if [ "$TOTAL_RAM_MB" -lt "$REQUIRED_RAM_MB" ]; then
  echo "FAIL: Only ${TOTAL_RAM_MB}MB RAM, need at least ${REQUIRED_RAM_MB}MB."
  ERRORS=$((ERRORS + 1))
else
  echo "OK: ${TOTAL_RAM_MB}MB RAM"
fi

# --- Disk space check ---
REQUIRED_DISK_GB=20
AVAILABLE_KB=$(df --output=avail / 2>/dev/null | tail -1)
AVAILABLE_GB=$((AVAILABLE_KB / 1024 / 1024))
if [ "$AVAILABLE_GB" -lt "$REQUIRED_DISK_GB" ]; then
  echo "FAIL: Only ${AVAILABLE_GB}GB free disk space, need at least ${REQUIRED_DISK_GB}GB."
  ERRORS=$((ERRORS + 1))
else
  echo "OK: ${AVAILABLE_GB}GB free disk space"
fi

# --- Summary ---
echo ""
if [ "$ERRORS" -gt 0 ]; then
  echo "Pre-flight checks failed with $ERRORS error(s). Please resolve the issues above before deploying."
  exit 1
fi

echo "All pre-flight checks passed."
```

## Writing Your Own Checks

When writing custom pre-flight check scripts, follow these conventions:

- **Exit with a non-zero status** on failure — this signals Distr to abort the deployment.
- **Print clear messages** prefixed with `FAIL:` or `OK:` so customers understand what went wrong.
- **Keep scripts idempotent and safe** — if they install or update software or otherwise change system state, they must be safe to run multiple times and avoid destructive side effects.
- **Test on the target OS** — Linux distributions may differ in available commands (e.g., `nproc` vs `sysctl`). Include fallbacks where possible.
- **Set reasonable thresholds** — document your minimums and explain why they are required.

## Next Steps

- **[Deployment Agents](/docs/product/agents/)** — Learn how Distr agents manage deployments on customer infrastructure
- **[Automatic Deployments from GitHub](/docs/guides/automatic-deployments-from-github/)** — Automate your CI/CD pipeline with the Distr GitHub Action
