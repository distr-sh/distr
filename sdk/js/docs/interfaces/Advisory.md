[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Advisory

# Interface: Advisory

## Extended by

- [`AdvisoryDetail`](AdvisoryDetail.md)

## Properties

### affected?

> `optional` **affected?**: `boolean`

Whether the requesting customer or partner still has a deployment running an affected
version. Absent for vendors, who see the status instead.

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

---

### cveId?

> `optional` **cveId?**: `string`

---

### fixedVersionCount

> **fixedVersionCount**: `number`

---

### id

> **id**: `string`

---

### publishedAt?

> `optional` **publishedAt?**: `string`

---

### referenceCount

> **referenceCount**: `number`

---

### resolvedAt?

> `optional` **resolvedAt?**: `string`

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
