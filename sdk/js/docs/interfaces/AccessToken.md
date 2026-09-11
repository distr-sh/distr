[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AccessToken

# Interface: AccessToken

## Extends

- [`BaseModel`](BaseModel.md)

## Extended by

- [`AccessTokenWithKey`](AccessTokenWithKey.md)

## Properties

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`createdAt`](BaseModel.md#createdat)

---

### expiresAt?

> `optional` **expiresAt?**: `string`

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`id`](BaseModel.md#id)

---

### keyId

> **keyId**: `string`

The part of the token that identifies it, so that a user can tell which of their tokens a
client is configured with. It stays the same when the secrets are rotated.

---

### label?

> `optional` **label?**: `string`

---

### lastUsedAt?

> `optional` **lastUsedAt?**: `string`

---

### secrets

> **secrets**: [`AccessTokenSecret`](AccessTokenSecret.md)[]

---

### userRole?

> `optional` **userRole?**: [`UserRole`](../type-aliases/UserRole.md)
