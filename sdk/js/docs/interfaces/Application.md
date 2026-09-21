[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / Application

# Interface: Application

## Extends

- [`BaseModel`](BaseModel.md).[`Named`](Named.md)

## Properties

### allowAutomaticUpdates?

> `optional` **allowAutomaticUpdates?**: `boolean`

Whether deployments of this application may have automatic updates enabled.

---

### createdAt?

> `optional` **createdAt?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`createdAt`](BaseModel.md#createdat)

---

### id?

> `optional` **id?**: `string`

#### Inherited from

[`BaseModel`](BaseModel.md).[`id`](BaseModel.md#id)

---

### imageId?

> `optional` **imageId?**: `string`

---

### imageUrl?

> `optional` **imageUrl?**: `string`

---

### name?

> `optional` **name?**: `string`

#### Inherited from

[`Named`](Named.md).[`name`](Named.md#name)

---

### type

> **type**: [`DeploymentType`](../type-aliases/DeploymentType.md)

---

### versioningStrategy?

> `optional` **versioningStrategy?**: [`VersioningStrategy`](../type-aliases/VersioningStrategy.md)

---

### versions?

> `optional` **versions?**: [`ApplicationVersion`](ApplicationVersion.md)[]
