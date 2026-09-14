[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / CreateAccessTokenRequest

# Interface: CreateAccessTokenRequest

## Properties

### expiresAt?

> `optional` **expiresAt?**: `Date`

The expiration of the secret the token is created with, and a token without one never expires.
It cannot be changed afterwards, so a token is kept alive by adding a secret that expires later.

---

### label?

> `optional` **label?**: `string`

---

### userRole?

> `optional` **userRole?**: [`UserRole`](../type-aliases/UserRole.md)
