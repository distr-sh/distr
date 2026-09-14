[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AccessTokenSecret

# Interface: AccessTokenSecret

## Properties

### createdAt

> **createdAt**: `string`

---

### expiresAt?

> `optional` **expiresAt?**: `string`

Fixed when the secret is created. A token is kept alive by adding a secret that expires later,
not by moving an expiration that something in circulation already relies on.

---

### lastUsedAt?

> `optional` **lastUsedAt?**: `string`

---

### slot

> **slot**: [`AccessTokenSecretSlot`](../type-aliases/AccessTokenSecretSlot.md)
