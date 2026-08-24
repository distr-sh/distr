[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AdvisoryDetail

# Interface: AdvisoryDetail

## Extends

- [`Advisory`](Advisory.md)

## Properties

### affected?

> `optional` **affected?**: `boolean`

Whether the advisory is still a live problem for the requesting customer or partner: a
deployment of theirs runs an affected version, or they pulled an affected artifact version
without since pulling one that carries the fix. Absent for vendors, who see the status
instead.

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

Only ever sent to the vendor organization that owns the advisory.

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

The vendor-internal timeline. Empty for customer and partner users.

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
