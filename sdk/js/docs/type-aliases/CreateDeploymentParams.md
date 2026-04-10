[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / CreateDeploymentParams

# Type Alias: CreateDeploymentParams

> **CreateDeploymentParams** = `object`

## Properties

### application

> **application**: `object`

#### id?

> `optional` **id**: `string`

#### versionId?

> `optional` **versionId**: `string`

---

### kubernetesDeployment?

> `optional` **kubernetesDeployment**: `object`

#### releaseName

> **releaseName**: `string`

#### valuesYaml?

> `optional` **valuesYaml**: `string`

---

### target

> **target**: `object`

#### kubernetes?

> `optional` **kubernetes**: `object`

##### kubernetes.namespace

> **namespace**: `string`

##### kubernetes.scope

> **scope**: [`DeploymentTargetScope`](DeploymentTargetScope.md)

#### name

> **name**: `string`

#### type

> **type**: [`DeploymentType`](DeploymentType.md)
