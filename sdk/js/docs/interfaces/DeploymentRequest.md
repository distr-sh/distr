[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / DeploymentRequest

# Interface: DeploymentRequest

## Properties

### applicationEntitlementId?

> `optional` **applicationEntitlementId?**: `string`

---

### applicationVersionId

> **applicationVersionId**: `string`

---

### automaticApplicationUpdatesEnabled?

> `optional` **automaticApplicationUpdatesEnabled?**: `boolean`

Leaves an existing deployment's setting alone when absent.

---

### deploymentId?

> `optional` **deploymentId?**: `string`

---

### deploymentTargetId

> **deploymentTargetId**: `string`

---

### dockerType?

> `optional` **dockerType?**: [`DockerType`](../type-aliases/DockerType.md)

---

### envFileData?

> `optional` **envFileData?**: `string`

---

### forceRestart?

> `optional` **forceRestart?**: `boolean`

---

### helmOptions?

> `optional` **helmOptions?**: [`HelmOptions`](HelmOptions.md)

---

### ignoreRevisionSkew?

> `optional` **ignoreRevisionSkew?**: `boolean`

---

### releaseName?

> `optional` **releaseName?**: `string`

---

### valuesYaml?

> `optional` **valuesYaml?**: `string`
