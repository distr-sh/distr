---
title: Feature Flags
description: Enable or disable feature flags for you self-hosted Distr instance.
sidebar:
  order: 4
---

Distr uses a primitive approach on feature management.
As long as a feature is not general available or we don't think that is should be activated by default, it is hidden behind a feature flag.
Feature flags are just an array in the postgres database.
All feature types can be found in [`types.go`](https://github.com/glasskube/distr/blob/main/internal/types/types.go).

If you want to enable a feature flag, you can do this by running the following command in your postgres database:

```sql
 update organization set features ='{licensing}';
```
