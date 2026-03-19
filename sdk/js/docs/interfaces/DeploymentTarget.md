[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / DeploymentTarget

# Interface: DeploymentTarget

## Extends

- [`BaseModel`](BaseModel.md).[`Named`](Named.md)

## Properties

### agentVersion?

> `optional` **agentVersion?**: [`AgentVersion`](AgentVersion.md)

---

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`createdAt`](BaseModel.md#createdat)

---

### currentStatus?

> `optional` **currentStatus?**: [`DeploymentTargetStatus`](DeploymentTargetStatus.md)

---

### customerOrganization?

> `optional` **customerOrganization?**: [`CustomerOrganization`](CustomerOrganization.md)

---

### deployments

> **deployments**: [`DeploymentWithLatestRevision`](DeploymentWithLatestRevision.md)[]

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`id`](BaseModel.md#id)

---

### imageCleanupEnabled

> **imageCleanupEnabled**: `boolean`

---

### metricsEnabled

> **metricsEnabled**: `boolean`

---

### name

> **name**: `string`

#### Overrides

[`Named`](Named.md).[`name`](Named.md#name)

---

### namespace?

> `optional` **namespace?**: `string`

---

### reportedAgentVersionId?

> `optional` **reportedAgentVersionId?**: `string`

---

### resources?

> `optional` **resources?**: [`DeploymentTargetResources`](DeploymentTargetResources.md)

---

### scope?

> `optional` **scope?**: [`DeploymentTargetScope`](../type-aliases/DeploymentTargetScope.md)

---

### type

> **type**: [`DeploymentType`](../type-aliases/DeploymentType.md)
