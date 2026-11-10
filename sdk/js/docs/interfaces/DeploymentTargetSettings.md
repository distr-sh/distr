[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / DeploymentTargetSettings

# Interface: DeploymentTargetSettings

## Extended by

- [`CreateDeploymentTargetRequest`](CreateDeploymentTargetRequest.md)
- [`UpdateDeploymentTargetRequest`](UpdateDeploymentTargetRequest.md)

## Properties

### automaticUpdatesEnabled?

> `optional` **automaticUpdatesEnabled?**: `boolean`

---

### deploymentLogsAfter?

> `optional` **deploymentLogsAfter?**: `string`

---

### deploymentLogsEnabled

> **deploymentLogsEnabled**: `boolean`

---

### dockerEndpoint?

> `optional` **dockerEndpoint?**: `string`

---

### imageCleanupEnabled

> **imageCleanupEnabled**: `boolean`

---

### metricsEnabled

> **metricsEnabled**: `boolean`

---

### name

> **name**: `string`

---

### nodeSelector?

> `optional` **nodeSelector?**: `Record`\<`string`, `string`\>

---

### podAnnotations?

> `optional` **podAnnotations?**: `Record`\<`string`, `string`\>

---

### resources?

> `optional` **resources?**: [`DeploymentTargetResources`](DeploymentTargetResources.md)

---

### tolerations?

> `optional` **tolerations?**: [`DeploymentTargetToleration`](DeploymentTargetToleration.md)[]
