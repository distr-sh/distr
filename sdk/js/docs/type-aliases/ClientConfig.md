[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / ClientConfig

# Type Alias: ClientConfig

> **ClientConfig** = `object`

## Properties

### apiBase

> **apiBase**: `string`

The base URL of the Distr API ending with /api/v1, e.g. https://app.distr.sh/api/v1.

---

### apiKey

> **apiKey**: `string`

The API key to authenticate with the Distr API.

---

### headers?

> `optional` **headers?**: `Record`\<`string`, `string`\>

Extra headers sent with every request, e.g. the service-token headers of an authenticating proxy in front of a
self-hosted Distr. They cannot override the Authorization header the client sets from [apiKey](#apikey).
