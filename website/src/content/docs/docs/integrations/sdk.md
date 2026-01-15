---
title: Distr SDK
description: Integrate Distr into your applications with our TypeScript/JavaScript SDK for programmatic access to deployments, registries, and more.
sidebar:
  order: 2
---

Distr SDKs are first-party libraries that allow you to interact with the Distr platform programmatically.
In the simplest case, you can use them as an API wrapper that performs executing HTTP requests to the [Distr API](/docs/integrations/api/).
For mor high level use cases, they offer services that help you to interact with the Distr platform in a more convenient way, e.g. by providing methods for checking if a deployment is outdated.

Currently, we offer an SDK for TS/JS, which is available on [npm](https://www.npmjs.com/package/@distr-sh/distr-sdk).
More languages are planned to be supported in the future.

## JavaScript SDK

You can install the Distr SDK for JavaScript from [npm](https://npmjs.org/package/@distr-sh/distr-sdk):

```shell
npm install --save @distr-sh/distr-sdk
```

Conceptually, the SDK is divided into two parts:

- A high-level service called `DistrService`, which provides a simplified interface for interacting with the Distr API.
- A low-level client called `Client`, which provides a more direct interface for interacting with the Distr API.

In order to connect to the Distr API, you have to [create a Personal Access Token (PAT)](/docs/integrations/personal-access-token/) in the Distr web interface.
Optionally, you can specify the URL of the Distr API you want to communicate with. It defaults to `https://app.distr.sh/api/v1`.

```typescript
import {DistrService} from '@distr-sh/distr-sdk';
const service = new DistrService({
  // to use your selfhosted instance, set apiBase: 'https://selfhosted-instance.company/api/v1',
  apiKey: '<your-personal-access-token-here>',
});
// do something with the service
```

The [src/examples](https://github.com/distr-sh/distr/tree/main/sdk/js/src/examples) directory contains examples of how to use the SDK.

See the [docs](https://github.com/distr-sh/distr/tree/main/sdk/js/docs/README.md) for more information.

## Feedback

We are always open to hear what is missing or not working in our SDKs. Please let us know in our [GitHub repository](https://github.com/distr-sh/distr).
