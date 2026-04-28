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
  region                      = var.aws_region
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
  default = "http://localhost:4566"
}

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "bucket_name" {
  type    = string
  default = "distr-opentofu-test"
}

resource "aws_s3_bucket" "test" {
  bucket = var.bucket_name
}

output "bucket_arn" {
  value = aws_s3_bucket.test.arn
}

output "bucket_name" {
  value = aws_s3_bucket.test.bucket
}
