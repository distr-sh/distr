[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AdvisoryDetail

# Interface: AdvisoryDetail

## Extends

- [`Advisory`](Advisory.md)

## Properties

### affected?

> `optional` **affected?**: `boolean`

Whether the requesting customer or partner still has a deployment running an affected
version. Absent for vendors, who see the status instead.

#### Inherited from

[`Advisory`](Advisory.md).[`affected`](Advisory.md#affected)

---

### affectedVersionCount

> **affectedVersionCount**: `number`

#### Inherited from

[`Advisory`](Advisory.md).[`affectedVersionCount`](Advisory.md#affectedversioncount)

---

### applicationVersions

> **applicationVersions**: [`AdvisoryApplicationVersion`](AdvisoryApplicationVersion.md)[]

---

### artifactVersions

> **artifactVersions**: [`AdvisoryArtifactVersion`](AdvisoryArtifactVersion.md)[]

---

### createdAt

> **createdAt**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`createdAt`](Advisory.md#createdat)

---

### createdByImageUrl?

> `optional` **createdByImageUrl?**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`createdByImageUrl`](Advisory.md#createdbyimageurl)

---

### createdByUserName?

> `optional` **createdByUserName?**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`createdByUserName`](Advisory.md#createdbyusername)

---

### cveId?

> `optional` **cveId?**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`cveId`](Advisory.md#cveid)

---

### description

> **description**: `string`

---

### events

> **events**: [`AdvisoryEvent`](AdvisoryEvent.md)[]

---

### fixedVersionCount

> **fixedVersionCount**: `number`

#### Inherited from

[`Advisory`](Advisory.md).[`fixedVersionCount`](Advisory.md#fixedversioncount)

---

### id

> **id**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`id`](Advisory.md#id)

---

### publishedAt?

> `optional` **publishedAt?**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`publishedAt`](Advisory.md#publishedat)

---

### referenceCount

> **referenceCount**: `number`

#### Inherited from

[`Advisory`](Advisory.md).[`referenceCount`](Advisory.md#referencecount)

---

### references

> **references**: [`AdvisoryReference`](AdvisoryReference.md)[]

---

### resolvedAt?

> `optional` **resolvedAt?**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`resolvedAt`](Advisory.md#resolvedat)

---

### severity

> **severity**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)

#### Inherited from

[`Advisory`](Advisory.md).[`severity`](Advisory.md#severity)

---

### status

> **status**: [`AdvisoryStatus`](../type-aliases/AdvisoryStatus.md)

#### Inherited from

[`Advisory`](Advisory.md).[`status`](Advisory.md#status)

---

### tags

> **tags**: `string`[]

#### Inherited from

[`Advisory`](Advisory.md).[`tags`](Advisory.md#tags)

---

### title

> **title**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`title`](Advisory.md#title)

---

### updatedAt

> **updatedAt**: `string`

#### Inherited from

[`Advisory`](Advisory.md).[`updatedAt`](Advisory.md#updatedat)
