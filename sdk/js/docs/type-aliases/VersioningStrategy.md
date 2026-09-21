[**@distr-sh/distr-sdk**](../README.md)

---

[@distr-sh/distr-sdk](../README.md) / VersioningStrategy

# Type Alias: VersioningStrategy

> **VersioningStrategy** = `"semver"` \| `"chronological"` \| `"legacy"`

How the latest version of an application is determined.

- 'semver' orders versions by semantic versioning, which every version name must follow.
- 'chronological' orders versions by their creation date.
- 'legacy' orders by semantic versioning where every name follows it and by creation date
  otherwise. It applies to applications created before a strategy existed and cannot be set.
