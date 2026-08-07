[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / CreateUpdateAdvisoryRequest

# Interface: CreateUpdateAdvisoryRequest

## Extended by

- [`CreateAdvisoryRequest`](CreateAdvisoryRequest.md)

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

### fixedApplicationVersionIds

> **fixedApplicationVersionIds**: `string`[]

---

### fixedArtifactVersionIds

> **fixedArtifactVersionIds**: `string`[]

---

### references

> **references**: [`AdvisoryReference`](AdvisoryReference.md)[]

---

### severity

> **severity**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)

---

### tags

> **tags**: `string`[]

---

### title

> **title**: `string`
