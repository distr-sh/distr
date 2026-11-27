# Using the Distr API

The Distr API is the REST API that the Distr web interface itself uses. Everything a user can do there can be automated through it.

## Base URL

Every instance serves the API on `/api/v1`: `https://app.distr.sh/api/v1` for Distr Cloud and `https://<host>/api/v1` for a self-hosted instance. Ask the user which instance they use when it is not clear from the context.

## Endpoints

Every instance serves an interactive reference on `/docs` and the OpenAPI specification on `/docs/openapi.json`, for Distr Cloud https://app.distr.sh/docs and https://app.distr.sh/docs/openapi.json. Look up the endpoint, its parameters and its request body in the specification before you call it.

## Authentication

Authenticate every request with a personal access token in the `Authorization` header, prefixed with `AccessToken`:

```shell
curl https://app.distr.sh/api/v1/applications \
  -H "Authorization: AccessToken $DISTR_ACCESS_TOKEN"
```

A request acts as the user who created the token. When the user has no token yet, point them to [Creating a Personal Access Token](https://distr.sh/docs/integrations/personal-access-token/).

## Limits

Each user may send 30 requests per second, 300 per minute and 2000 per hour. Beyond that the API answers `429 Too Many Requests`, so back off and retry instead of sending requests in a tight loop. Request bodies larger than 1 MiB are rejected.

## Further reading

- [Distr API](https://distr.sh/docs/integrations/api/)
