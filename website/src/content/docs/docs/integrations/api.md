---
title: Distr API
description: Using the Distr REST API, you can programmatically manage every aspect of your software distribution, from publishing artifacts and creating customers to triggering deployment updates. Using the same API that's behind the Vendor and Customer Portals.

sidebar:
  order: 1
---

Distr exposes a RESTful JSON API that allows you to interact with the platform programmatically.
It is the same API used by the Distr Vendor and Customer Portals.

If you prefer to interact with the API on a higher level of abstraction, you can use our [first-party SDKs](/docs/integrations/sdk/)
that handle authentication and error handling for you, and provide types.

## Base URL

Every Distr instance, no matter if self-managed or the SaaS version, exposes this API under the `/api/v1` path.

## Authentication

To authenticate with the Distr API, you need to use a Personal Access Token.
To create a Personal Access Token, follow the steps outlined in the [Creating a Personal Access Token](/docs/integrations/personal-access-token/) guide.

For each HTTP request to the API, you need to include the Personal Access Token in the `Authorization` header, like shown here:

```shell
curl 'https://app.distr.sh/api/v1/applications' \
  -H "Authorization: AccessToken <your-access-token-here>"
```

This will authenticate you as the user who created the Personal Access Token for the lifetime of this request.
The token needs to be sent with every request to the API.

## Request Limits

The following limits apply to the Distr API:

- 5 requests per second
- 60 requests per minute
- 2000 requests per hour

If these limits are exceeded, the API will respond with a `429 Too Many Requests` status code.

Additionally, the API rejects request bodies larger than 1MiB.
