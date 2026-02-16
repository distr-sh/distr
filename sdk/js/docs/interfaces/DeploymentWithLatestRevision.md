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

### ~~applicationId~~

> **applicationId**: `string`

#### Deprecated

Use application.id instead

---

### applicationLicenseId?

> `optional` **applicationLicenseId**: `string`

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

### createdAt?

> `optional` **createdAt**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`createdAt`](Deployment.md#createdat)

---

### deploymentRevisionCreatedAt?

> `optional` **deploymentRevisionCreatedAt**: `string`

---

### deploymentRevisionId?

> `optional` **deploymentRevisionId**: `string`

---

### deploymentTargetId

> **deploymentTargetId**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`deploymentTargetId`](Deployment.md#deploymenttargetid)

---

### dockerType?

> `optional` **dockerType**: [`DockerType`](../type-aliases/DockerType.md)

#### Inherited from

[`Deployment`](Deployment.md).[`dockerType`](Deployment.md#dockertype)

---

### envFileData?

> `optional` **envFileData**: `string`

---

### helmOptions?

> `optional` **helmOptions**: [`HelmOptions`](HelmOptions.md)

---

### id?

> `optional` **id**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`id`](Deployment.md#id)

---

### latestStatus?

> `optional` **latestStatus**: [`DeploymentRevisionStatus`](DeploymentRevisionStatus.md)

---

### logsEnabled

> **logsEnabled**: `boolean`

#### Inherited from

[`Deployment`](Deployment.md).[`logsEnabled`](Deployment.md#logsenabled)

---

### releaseName?

> `optional` **releaseName**: `string`

#### Inherited from

[`Deployment`](Deployment.md).[`releaseName`](Deployment.md#releasename)

---

### valuesYaml?

> `optional` **valuesYaml**: `string`
