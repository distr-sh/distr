[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Deployment

# Interface: Deployment

## Extends

- [`BaseModel`](BaseModel.md)

## Extended by

- [`DeploymentWithLatestRevision`](DeploymentWithLatestRevision.md)

## Properties

### automaticApplicationUpdatesEnabled?

> `optional` **automaticApplicationUpdatesEnabled?**: `boolean`

Whether this deployment is rolled forward to the application's latest version automatically.
Unrelated to DeploymentTarget.automaticUpdatesEnabled, which is the agent updating itself.

---

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`createdAt`](BaseModel.md#createdat)

---

### deploymentTargetId

> **deploymentTargetId**: `string`

---

### dockerType?

> `optional` **dockerType?**: [`DockerType`](../type-aliases/DockerType.md)

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`id`](BaseModel.md#id)

---

### releaseName?

> `optional` **releaseName?**: `string`
