[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Client

# Class: Client

The low-level Distr API client. Each method represents on API endpoint.

## Constructors

### Constructor

> **new Client**(`config`): `Client`

#### Parameters

##### config

`ConditionalPartial`\<[`ClientConfig`](../type-aliases/ClientConfig.md), `"apiBase"`\>

#### Returns

`Client`

## Methods

### createAccessForDeploymentTarget()

> **createAccessForDeploymentTarget**(`deploymentTargetId`): `Promise`\<[`DeploymentTargetAccessResponse`](../interfaces/DeploymentTargetAccessResponse.md)\>

#### Parameters

##### deploymentTargetId

`string`

#### Returns

`Promise`\<[`DeploymentTargetAccessResponse`](../interfaces/DeploymentTargetAccessResponse.md)\>

---

### createAdvisory()

> **createAdvisory**(`request`): `Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

#### Parameters

##### request

[`CreateUpdateAdvisoryRequest`](../interfaces/CreateUpdateAdvisoryRequest.md)

#### Returns

`Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

---

### createAdvisoryComment()

> **createAdvisoryComment**(`advisoryId`, `request`): `Promise`\<[`AdvisoryEvent`](../interfaces/AdvisoryEvent.md)\>

#### Parameters

##### advisoryId

`string`

##### request

[`CreateAdvisoryCommentRequest`](../interfaces/CreateAdvisoryCommentRequest.md)

#### Returns

`Promise`\<[`AdvisoryEvent`](../interfaces/AdvisoryEvent.md)\>

---

### createApplication()

> **createApplication**(`application`): `Promise`\<[`Application`](../interfaces/Application.md)\>

#### Parameters

##### application

[`Application`](../interfaces/Application.md)

#### Returns

`Promise`\<[`Application`](../interfaces/Application.md)\>

---

### createApplicationVersion()

> **createApplicationVersion**(`applicationId`, `version`, `files?`): `Promise`\<[`ApplicationVersion`](../interfaces/ApplicationVersion.md)\>

#### Parameters

##### applicationId

`string`

##### version

[`ApplicationVersion`](../interfaces/ApplicationVersion.md)

##### files?

[`ApplicationVersionFiles`](../type-aliases/ApplicationVersionFiles.md)

#### Returns

`Promise`\<[`ApplicationVersion`](../interfaces/ApplicationVersion.md)\>

---

### createDeploymentTarget()

> **createDeploymentTarget**(`deploymentTarget`): `Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)\>

#### Parameters

##### deploymentTarget

[`DeploymentTarget`](../interfaces/DeploymentTarget.md)

#### Returns

`Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)\>

---

### createOrUpdateDeployment()

> **createOrUpdateDeployment**(`deploymentRequest`): `Promise`\<[`DeploymentRequest`](../interfaces/DeploymentRequest.md)\>

#### Parameters

##### deploymentRequest

[`DeploymentRequest`](../interfaces/DeploymentRequest.md)

#### Returns

`Promise`\<[`DeploymentRequest`](../interfaces/DeploymentRequest.md)\>

---

### getAdvisories()

> **getAdvisories**(`filter?`): `Promise`\<[`Advisory`](../interfaces/Advisory.md)[]\>

#### Parameters

##### filter?

[`AdvisoryFilter`](../interfaces/AdvisoryFilter.md) = `{}`

#### Returns

`Promise`\<[`Advisory`](../interfaces/Advisory.md)[]\>

---

### getAdvisory()

> **getAdvisory**(`advisoryId`): `Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

#### Parameters

##### advisoryId

`string`

#### Returns

`Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

---

### getAdvisoryImpact()

> **getAdvisoryImpact**(`advisoryId`): `Promise`\<[`AdvisoryImpact`](../interfaces/AdvisoryImpact.md)\>

#### Parameters

##### advisoryId

`string`

#### Returns

`Promise`\<[`AdvisoryImpact`](../interfaces/AdvisoryImpact.md)\>

---

### getAdvisoryTags()

> **getAdvisoryTags**(): `Promise`\<`string`[]\>

Spans undisclosed advisories and is therefore available to the vendor organization only.

#### Returns

`Promise`\<`string`[]\>

---

### getApplication()

> **getApplication**(`applicationId`): `Promise`\<[`Application`](../interfaces/Application.md)\>

#### Parameters

##### applicationId

`string`

#### Returns

`Promise`\<[`Application`](../interfaces/Application.md)\>

---

### getApplications()

> **getApplications**(): `Promise`\<[`Application`](../interfaces/Application.md)[]\>

#### Returns

`Promise`\<[`Application`](../interfaces/Application.md)[]\>

---

### getApplicationVersionResources()

> **getApplicationVersionResources**(`applicationId`, `versionId`): `Promise`\<[`ApplicationVersionResource`](../interfaces/ApplicationVersionResource.md)[]\>

#### Parameters

##### applicationId

`string`

##### versionId

`string`

#### Returns

`Promise`\<[`ApplicationVersionResource`](../interfaces/ApplicationVersionResource.md)[]\>

---

### getDeploymentTarget()

> **getDeploymentTarget**(`deploymentTargetId`): `Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)\>

#### Parameters

##### deploymentTargetId

`string`

#### Returns

`Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)\>

---

### getDeploymentTargets()

> **getDeploymentTargets**(): `Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)[]\>

#### Returns

`Promise`\<[`DeploymentTarget`](../interfaces/DeploymentTarget.md)[]\>

---

### patchAdvisory()

> **patchAdvisory**(`advisoryId`, `request`): `Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

#### Parameters

##### advisoryId

`string`

##### request

[`PatchAdvisoryRequest`](../interfaces/PatchAdvisoryRequest.md)

#### Returns

`Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

---

### updateAdvisory()

> **updateAdvisory**(`advisoryId`, `request`): `Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

#### Parameters

##### advisoryId

`string`

##### request

[`CreateUpdateAdvisoryRequest`](../interfaces/CreateUpdateAdvisoryRequest.md)

#### Returns

`Promise`\<[`AdvisoryDetail`](../interfaces/AdvisoryDetail.md)\>

---

### updateApplication()

> **updateApplication**(`application`): `Promise`\<[`Application`](../interfaces/Application.md)\>

#### Parameters

##### application

[`Application`](../interfaces/Application.md)

#### Returns

`Promise`\<[`Application`](../interfaces/Application.md)\>
