# OpenTofu Agent — Quick Start Example

This doc walks through a minimal end-to-end test of the OpenTofu agent against
LocalStack so you can verify the full flow without touching real cloud
resources.

## Prerequisites

- `docker compose up -d` (postgres, minio, localstack, mailpit)
- Hub running on `:8080` (and the OCI registry on `:8585`)
- `oras`, `tofu`, `aws` CLIs installed

## 1. Sample OpenTofu configuration

Save this as `main.tf`:

```hcl
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  backend "http" {}
}

provider "aws" {
  region                      = "us-east-1"
  access_key                  = "test"
  secret_key                  = "test"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true
  endpoints {
    s3 = var.localstack_endpoint
  }
  s3_use_path_style = true
}

variable "localstack_endpoint" {
  type    = string
  default = "http://host.docker.internal:4566"
}

variable "bucket_name" {
  type    = string
  default = "distr-opentofu-demo"
}

resource "aws_s3_bucket" "demo" {
  bucket = var.bucket_name
}

output "bucket_arn" {
  value = aws_s3_bucket.demo.arn
}
```

## 2. Push the config as an OCI artifact

```bash
# Get a Personal Access Token from Settings → Access Tokens in the UI
PAT="distr-xxxxxxxxxxxxxxxx"

echo "$PAT" | oras login localhost:8585 -u "you@example.com" --password-stdin --insecure

oras push --insecure localhost:8585/your-org-slug/aws-s3-bucket:v0.1.0 \
  main.tf:application/vnd.distr.opentofu.config.v1.tar+gzip
```

## 3. Create the application + version in the Hub UI

1. Enable the OpenTofu feature for your organization (Settings → Organization).
2. Applications → New Application → type `opentofu`, name `aws-s3-bucket`.
3. New Version:
   - Version Name: `v0.1.0`
   - OCI Config Reference: `your-org-slug/aws-s3-bucket`
   - Config Tag: `v0.1.0`

## 4. Create a deployment target and deploy

1. Deployments → Deploy App
2. Select the `aws-s3-bucket` application
3. Target name: `localstack-dev`
4. In OpenTofu Variables (JSON), paste:
   ```json
   {"localstack_endpoint": "http://host.docker.internal:4566", "bucket_name": "distr-opentofu-demo"}
   ```
5. Continue through the wizard to get the `connect` command, run it on the
   target host, and watch the deployment go healthy.

## 5. Verify in LocalStack

```bash
aws --endpoint-url=http://localhost:4566 s3 ls
# 2026-04-27 12:00:00 distr-opentofu-demo
```

You should see the bucket created by the agent. The Terraform state is stored
in the Hub's S3 backend at `state/<deploymentID>` and locked via the
`opentofu_state` table.
