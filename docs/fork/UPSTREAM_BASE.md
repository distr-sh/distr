# Upstream Base

This fork baseline was recorded for PR-000 from `docs/roadmaps/DISTR_COMMUNITY_FORK_MASTER_PLAN.md`.

## Upstream

- Repository: `https://github.com/distr-sh/distr.git`
- Default branch: `main`
- Commit SHA: `b49fb27eb6270d7a71eed82b12e47eec1217c4cf`
- Detected version/tag: `2.24.1`
- Recorded date: `2026-06-20`
- Local checkout: `C:\Users\pc\Desktop\repository\distr-octopus-community`
- Roadmap SHA256: `4650858F21F9C8729D12754F14409C2C051F56FAE54847CE3B349C130B013F90`

## Toolchain

The upstream `mise.toml` declares:

- Go `1.26.4`
- Node.js `26`
- pnpm `11.7.0`
- golangci-lint `2.12.2`
- helm-docs `1.14.2`
- watchexec `2.5.1`
- stripe `1.42.13`
- git-lfs `3.7.1`
- Delve `1.26.3`

On Windows, `mise` had to be run through Git Bash because `mise.toml` sources `hack/current-commit.sh` with `/bin/bash`.

## Baseline Commands

```shell
git ls-remote --symref https://github.com/distr-sh/distr.git HEAD
git clone https://github.com/distr-sh/distr.git C:\Users\pc\Desktop\repository\distr-octopus-community
git rev-parse HEAD
git describe --tags --always --dirty
```

Result:

```text
HEAD branch: main
HEAD SHA: b49fb27eb6270d7a71eed82b12e47eec1217c4cf
Detected tag/version: 2.24.1
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise install"
```

Result:

```text
Passed. mise installed the declared Go, Node.js, pnpm, golangci-lint, helm-docs, watchexec, stripe, git-lfs, and Delve tools.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise run test"
```

Result:

```text
Passed after freeing local disk space. The first attempt failed during Go compilation because the default Go temp work directory under C:\tmp ran out of disk space.
```

Fallback command used during the low-disk investigation:

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && GOTMPDIR=/c/tmp/go-single-work /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise exec -- go test -p 1 ./..."
```

Result:

```text
Passed. All Go packages completed successfully with package parallelism reduced to 1.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise exec -- pnpm install"
```

Result:

```text
Passed after freeing local disk space. The first attempt failed with ENOSPC while installing root frontend dependencies.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise run lint:migrations"
```

Result:

```text
Failed on Windows because the task tried to run `hack/validate-migrations.sh` through a Windows command shell.
```

Direct Git Bash validation command:

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && bash hack/validate-migrations.sh"
```

Result:

```text
Passed. All migration files from 0 through 107 are properly paired.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise run build:agent:docker"
```

Result:

```text
Passed.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise run build:agent:kubernetes"
```

Result:

```text
Passed.
```

```shell
C:\Program Files\Git\bin\bash.exe -lc "cd /c/Users/pc/Desktop/repository/distr-octopus-community && /c/Users/pc/AppData/Local/Microsoft/WinGet/Packages/jdx.mise_Microsoft.Winget.Source_8wekyb3d8bbwe/mise/bin/mise run build:hub:community"
```

Result:

```text
Failed in the Angular community frontend build after SDK dependencies, SDK build/docs, root dependencies, and Go tidy completed.

Angular compiler errors are in existing upstream files:
- `frontend/ui/src/app/deployments/deployment-target-card/deployment-target-card.component.html`
- `frontend/ui/src/app/deployments/deployment-target-card/deployment-target-card.component.ts`

Primary error shape:
- `TS2339: Property 'version' does not exist on type 'never'.`
- `TS2339: Property 'sections' does not exist on type 'never'.`
- `TS7006: Parameter implicitly has an 'any' type.`

The build script generated `frontend/ui/src/data/agent-changelog.json` with `{"releases":[]}` before the Angular compiler failure.
```

## Baseline Status

- Go tests pass with the standard `mise run test` task.
- Docker and Kubernetes agent builds pass.
- Migration pairing validation passes when the shell script is run directly through Git Bash.
- Root frontend dependency installation passes with `pnpm install`.
- The community hub build is not green at this upstream commit because the Angular community frontend build fails in existing deployment-target changelog typing/template code.
- No functional code, database migration, API endpoint, UI route, or agent protocol change is included in PR-000.
- The root `AGENTS.md` requires Codex to read the master plan before making roadmap or fork feature changes.
