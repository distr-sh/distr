[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / PatchAccessTokenRequest

# Interface: PatchAccessTokenRequest

Supports partial updates: an omitted field is left unchanged, an explicit null clears it, which
means no label, no expiry and the role of the user.

## Properties

### expiresAt?

> `optional` **expiresAt?**: `Date` \| `null`

---

### label?

> `optional` **label?**: `string` \| `null`

---

### userRole?

> `optional` **userRole?**: [`UserRole`](../type-aliases/UserRole.md) \| `null`
