[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AdvisoryImpactedDeployment

# Interface: AdvisoryImpactedDeployment

One deployment that has run an affected application version at some point. `applicationVersion*`
is the most recent affected version it ran and `lastDeployedAt` is when, while
`currentApplicationVersion*` is what it runs today and is what `state` is derived from.

## Properties

### applicationId

> **applicationId**: `string`

---

### applicationName

> **applicationName**: `string`

---

### applicationVersionId

> **applicationVersionId**: `string`

---

### applicationVersionName

> **applicationVersionName**: `string`

---

### currentApplicationVersionId

> **currentApplicationVersionId**: `string`

---

### currentApplicationVersionName

> **currentApplicationVersionName**: `string`

---

### customerOrganizationId?

> `optional` **customerOrganizationId?**: `string`

---

### customerOrganizationName?

> `optional` **customerOrganizationName?**: `string`

---

### deploymentId

> **deploymentId**: `string`

---

### deploymentTargetId

> **deploymentTargetId**: `string`

---

### deploymentTargetName

> **deploymentTargetName**: `string`

---

### lastDeployedAt

> **lastDeployedAt**: `string`

---

### state

> **state**: [`AdvisoryImpactState`](../type-aliases/AdvisoryImpactState.md)
