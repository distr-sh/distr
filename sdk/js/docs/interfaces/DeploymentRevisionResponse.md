[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / DeploymentRevisionResponse

# Interface: DeploymentRevisionResponse

## Properties

### applicationVersionId

> **applicationVersionId**: `string`

---

### applicationVersionName

> **applicationVersionName**: `string`

---

### createdAt

> **createdAt**: `string`

---

### createdBy?

> `optional` **createdBy?**: [`DeploymentRevisionCreator`](DeploymentRevisionCreator.md)

---

### dockerType?

> `optional` **dockerType?**: [`DockerType`](../type-aliases/DockerType.md)

---

### envFileData?

> `optional` **envFileData?**: `string`

---

### forceRestart

> **forceRestart**: `boolean`

---

### helmOptions?

> `optional` **helmOptions?**: [`HelmOptions`](HelmOptions.md)

---

### id

> **id**: `string`

---

### ignoreRevisionSkew

> **ignoreRevisionSkew**: `boolean`

---

### latestStatus?

> `optional` **latestStatus?**: [`DeploymentRevisionStatus`](DeploymentRevisionStatus.md)

---

### releaseName?

> `optional` **releaseName?**: `string`

---

### trigger

> **trigger**: [`DeploymentRevisionTrigger`](../type-aliases/DeploymentRevisionTrigger.md)

---

### valuesYaml?

> `optional` **valuesYaml?**: `string`
