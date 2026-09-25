[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Advisory

# Interface: Advisory

## Extended by

- [`AdvisoryDetail`](AdvisoryDetail.md)

## Properties

### affected?

> `optional` **affected?**: `boolean`

Whether the advisory is still a live problem for the requesting customer or partner: a
deployment of theirs runs an affected version, or they pulled an affected artifact version
without since pulling a patched one. Absent for vendors, who see the status instead.

---

### affectedVersionCount

> **affectedVersionCount**: `number`

---

### createdAt

> **createdAt**: `string`

---

### createdByImageUrl?

> `optional` **createdByImageUrl?**: `string`

---

### createdByUserName?

> `optional` **createdByUserName?**: `string`

Only ever sent to the vendor organization that owns the advisory.

---

### cveId?

> `optional` **cveId?**: `string`

---

### id

> **id**: `string`

---

### patchedVersionCount

> **patchedVersionCount**: `number`

---

### publishedAt?

> `optional` **publishedAt?**: `string`

---

### referenceCount

> **referenceCount**: `number`

---

### resolvedAt?

> `optional` **resolvedAt?**: `string`

Only ever sent to the vendor organization that owns the advisory.

---

### severity

> **severity**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)

---

### status

> **status**: [`AdvisoryStatus`](../type-aliases/AdvisoryStatus.md)

---

### tags

> **tags**: `string`[]

---

### title

> **title**: `string`

---

### updatedAt

> **updatedAt**: `string`
