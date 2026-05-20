[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / toBase64

# Function: toBase64()

> **toBase64**(`value`, `field`): `string`

Wraps `btoa` and rethrows the InvalidCharacterError as a typed
[Base64EncodeError](../classes/Base64EncodeError.md) with a descriptive message identifying the field
that failed to encode.

## Parameters

### value

`string`

### field

`string`

## Returns

`string`
