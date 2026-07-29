[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AdvisoryFilter

# Interface: AdvisoryFilter

Filters for the advisory list. Each field matches an advisory having any of the
given values; omitting a field, or passing an empty array, disables that filter.

## Properties

### severity?

> `optional` **severity?**: [`AdvisorySeverity`](../type-aliases/AdvisorySeverity.md)[]

---

### status?

> `optional` **status?**: [`AdvisoryStatus`](../type-aliases/AdvisoryStatus.md)[]

---

### tag?

> `optional` **tag?**: `string`[]
