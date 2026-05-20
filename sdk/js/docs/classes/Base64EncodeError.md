[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Base64EncodeError

# Class: Base64EncodeError

Thrown by [toBase64](../functions/toBase64.md) when the input contains characters that the
platform's `btoa` cannot encode (codepoints outside the Latin-1 range,
i.e. > 0xFF).

Common culprits: the emdash "—" (U+2014), smart quotes (" " ' '), and
other typographic punctuation that text editors silently insert.

## Extends

- `Error`

## Constructors

### Constructor

> **new Base64EncodeError**(`field`): `Base64EncodeError`

#### Parameters

##### field

`string`

#### Returns

`Base64EncodeError`

#### Overrides

`Error.constructor`

## Properties

### cause?

> `optional` **cause?**: `unknown`

#### Inherited from

`Error.cause`

---

### field

> `readonly` **field**: `string`

---

### message

> **message**: `string`

#### Inherited from

`Error.message`

---

### name

> **name**: `string`

#### Inherited from

`Error.name`

---

### stack?

> `optional` **stack?**: `string`

#### Inherited from

`Error.stack`
