---
title: Service Accounts
description: Create non-personalized service accounts to authenticate CI/CD pipelines and other automation with Distr.
slug: docs/integrations/service-account
sidebar:
  order: 5
---

A **service account** is a non-personalized identity that authenticates automation (CI/CD pipelines, infrastructure
tooling, webhook receivers, scripts) against the Distr API. Unlike a Personal Access Token, a service account is not
tied to any human user, can hold multiple access tokens, and cannot log in to the web interface.

Service accounts are the recommended way to authenticate machines and automation:

- They survive employee turnover — you do not lose access when a user leaves.
- Each service account has its own role, so you can grant least-privilege access per integration.
- A service account can hold several tokens at once, so you can rotate without downtime.
- Tokens authenticate against the same `Authorization: AccessToken distr-…` header as PATs, so SDK and client code does
  not change.

Only **administrators** can create, edit, delete, or manage tokens for service accounts.

## Creating a service account

1. In the sidebar, open **Users**. Below the users table you will see a **Service Accounts** section.
2. Click **Create service account**.
3. Enter a descriptive name (for example `ci-bot`) and choose the role the service account should have within the
   organization. The role controls what the automation is allowed to do — pick the least-privilege role that works.
4. Click **Create**. The new service account appears in the table.

For customer-scoped automation (running inside a customer organization), open that customer's detail page and create
the service account from the **Service Accounts** table there. The token of that service account will only see
resources scoped to that customer.

## Managing access tokens

A service account does not have a token until you create one. Click **Manage tokens** on its row to open the
service-account detail page.

1. Click **Create token**, give the token a label (for example `github-actions`) and an optional expiry date.
2. Click **Create**. The token is shown **once** at the top of the page. Copy it now and store it in your secret store
   (GitHub Secrets, Vault, etc.) — you cannot retrieve it again.
3. Repeat as needed. A service account can hold any number of active tokens.

To revoke a token, click the trash icon on its row. Any automation using that token will receive `401 Unauthorized` on
the next request.

## Using a service account token

Service account tokens authenticate exactly like Personal Access Tokens. Set the `Authorization` header to
`AccessToken distr-…`:

```bash
curl -H "Authorization: AccessToken distr-xxxxxxxxxxxxxx" https://app.distr.sh/api/v1/deployment-targets
```

For OCI registry access (`docker login`, `helm registry login`), use the token as the password and any non-empty string
as the username — same as a PAT.

## Limits

- Service accounts can be created and managed by users with the `admin` role only.
- A service account cannot manage other service accounts, change its own role, or be used to log into the web UI.
- Service accounts do not count toward your organization's user-seat quota.
