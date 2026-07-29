[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / CreateAdvisoryRequest

# Interface: CreateAdvisoryRequest

## Extends

- [`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md)

## Properties

### affectedApplicationVersionIds

> **affectedApplicationVersionIds**: `string`[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`affectedApplicationVersionIds`](CreateUpdateAdvisoryRequest.md#affectedapplicationversionids)

---

### affectedArtifactVersionIds

> **affectedArtifactVersionIds**: `string`[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`affectedArtifactVersionIds`](CreateUpdateAdvisoryRequest.md#affectedartifactversionids)

---

### cveId?

> `optional` **cveId?**: `string`

Unique per organization, ignoring case. Reusing one that another advisory already
carries is rejected with 409 Conflict.

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`cveId`](CreateUpdateAdvisoryRequest.md#cveid)

---

### description

> **description**: `string`

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`description`](CreateUpdateAdvisoryRequest.md#description)

---

### fixedApplicationVersionIds

> **fixedApplicationVersionIds**: `string`[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`fixedApplicationVersionIds`](CreateUpdateAdvisoryRequest.md#fixedapplicationversionids)

---

### fixedArtifactVersionIds

> **fixedArtifactVersionIds**: `string`[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`fixedArtifactVersionIds`](CreateUpdateAdvisoryRequest.md#fixedartifactversionids)

---

### references

> **references**: [`AdvisoryReference`](AdvisoryReference.md)[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`references`](CreateUpdateAdvisoryRequest.md#references)

---

### severity

> **severity**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`severity`](CreateUpdateAdvisoryRequest.md#severity)

---

### status?

> `optional` **status?**: [`InitialAdvisoryStatus`](../type-aliases/InitialAdvisoryStatus.md)

Status the advisory starts in. Defaults to `triage`, where issues reported through
the API wait to be assessed. Pass `draft` for an advisory you are writing yourself.

---

### tags

> **tags**: `string`[]

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`tags`](CreateUpdateAdvisoryRequest.md#tags)

---

### title

> **title**: `string`

#### Inherited from

[`CreateUpdateAdvisoryRequest`](CreateUpdateAdvisoryRequest.md).[`title`](CreateUpdateAdvisoryRequest.md#title)
