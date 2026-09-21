[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / DeploymentWithLatestRevision

# Interface: DeploymentWithLatestRevision

## Extends

- [`Deployment`](Deployment.md)

## Properties

### application

> **application**: [`Application`](Application.md)

---

### applicationEntitlementId?

> `optional` **applicationEntitlementId?**: `string`

---

### ~~applicationId~~

> **applicationId**: `string`

#### Deprecated

Use application.id instead

---

### applicationLink

> **applicationLink**: `string`

---

### ~~applicationName~~

> **applicationName**: `string`

#### Deprecated

Use application.name instead

---

### applicationVersionId

> **applicationVersionId**: `string`

---

### applicationVersionName

> **applicationVersionName**: `string`

---

### automaticApplicationUpdatesEnabled?

> `optional` **automaticApplicationUpdatesEnabled?**: `boolean`

Whether this deployment is rolled forward to the application's latest version automatically.
Unrelated to DeploymentTarget.automaticUpdatesEnabled, which is the agent updating itself.

#### Inherited from

[`Deployment`](Deployment.md).[`automaticApplicationUpdatesEnabled`](Deployment.md#automaticapplicationupdatesenabled)

---

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`createdAt`](Deployment.md#createdat)

---

### currentDeploymentRevisionId?

> `optional` **currentDeploymentRevisionId?**: `string`

The revision an agent last reported as applied, which differs from deploymentRevisionId while a
newer revision is being rolled out or has failed.

---

### currentStatus?

> `optional` **currentStatus?**: [`DeploymentRevisionStatus`](DeploymentRevisionStatus.md)

---

### deploymentRevisionCreatedAt?

> `optional` **deploymentRevisionCreatedAt?**: `string`

---

### deploymentRevisionId?

> `optional` **deploymentRevisionId?**: `string`

---

### deploymentTargetId

> **deploymentTargetId**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`deploymentTargetId`](Deployment.md#deploymenttargetid)

---

### dockerType?

> `optional` **dockerType?**: [`DockerType`](../type-aliases/DockerType.md)

#### Inherited from

[`Deployment`](Deployment.md).[`dockerType`](Deployment.md#dockertype)

---

### envFileData?

> `optional` **envFileData?**: `string`

---

### helmOptions?

> `optional` **helmOptions?**: [`HelmOptions`](HelmOptions.md)

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`id`](Deployment.md#id)

---

### latestStatus?

> `optional` **latestStatus?**: [`DeploymentRevisionStatus`](DeploymentRevisionStatus.md)

---

### releaseName?

> `optional` **releaseName?**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`releaseName`](Deployment.md#releasename)

---

### valuesYaml?

> `optional` **valuesYaml?**: `string`
