[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / CreateUpdateAdvisoryRequest

# Interface: CreateUpdateAdvisoryRequest

## Properties

### affectedApplicationVersionIds

> **affectedApplicationVersionIds**: `string`[]

---

### affectedArtifactVersionIds

> **affectedArtifactVersionIds**: `string`[]

---

### cveId?

> `optional` **cveId?**: `string`

Unique per organization, ignoring case. Reusing one that another advisory already
carries is rejected with 409 Conflict.

---

### description

> **description**: `string`

---

### patchedApplicationVersionIds

> **patchedApplicationVersionIds**: `string`[]

---

### patchedArtifactVersionIds

> **patchedArtifactVersionIds**: `string`[]

---

### references

> **references**: [`AdvisoryReference`](AdvisoryReference.md)[]

---

### severity

> **severity**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)

---

### status?

> `optional` **status?**: [`AdvisoryStatus`](../type-aliases/AdvisoryStatus.md)

Defaults to `triage` on create and leaves the status untouched when omitted on update.

---

### tags

> **tags**: `string`[]

---

### title

> **title**: `string`
