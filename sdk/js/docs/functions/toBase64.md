[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / toBase64

# Function: toBase64()

> **toBase64**(`value`, `field`): `string`

Encodes a string as base64. Pre-validates the input against the Latin-1
range that `btoa` accepts; if the input is out of range, throws a typed
[Base64EncodeError](../classes/Base64EncodeError.md) naming the field. Any other errors thrown by
the platform's `btoa` (e.g. it being unavailable) propagate unchanged.

## Parameters

### value

`string`

### field

`string`

## Returns

`string`
