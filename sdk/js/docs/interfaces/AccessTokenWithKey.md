[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / AccessTokenWithKey

# Interface: AccessTokenWithKey

## Extends

- [`AccessToken`](AccessToken.md)

## Properties

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`AccessToken`](AccessToken.md).[`createdAt`](AccessToken.md#createdat)

---

### expiresAt?

> `optional` **expiresAt?**: `string`

#### Inherited from

[`AccessToken`](AccessToken.md).[`expiresAt`](AccessToken.md#expiresat)

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`AccessToken`](AccessToken.md).[`id`](AccessToken.md#id)

---

### key

> **key**: `string`

---

### keyId

> **keyId**: `string`

The part of the token that identifies it, so that a user can tell which of their tokens a
client is configured with. It stays the same when the secrets are rotated.

#### Inherited from

[`AccessToken`](AccessToken.md).[`keyId`](AccessToken.md#keyid)

---

### label?

> `optional` **label?**: `string`

#### Inherited from

[`AccessToken`](AccessToken.md).[`label`](AccessToken.md#label)

---

### lastUsedAt?

> `optional` **lastUsedAt?**: `string`

#### Inherited from

[`AccessToken`](AccessToken.md).[`lastUsedAt`](AccessToken.md#lastusedat)

---

### secrets

> **secrets**: [`AccessTokenSecret`](AccessTokenSecret.md)[]

#### Inherited from

[`AccessToken`](AccessToken.md).[`secrets`](AccessToken.md#secrets)

---

### userRole?

> `optional` **userRole?**: [`UserRole`](../type-aliases/UserRole.md)

#### Inherited from

[`AccessToken`](AccessToken.md).[`userRole`](AccessToken.md#userrole)
