# Git Workflow — Alkalyne-CLI

## Branch Strategy

```
main              Production-ready, tagged releases
  └─ develop      Integration branch for feature work
       ├─ feat/<name>       New features (from develop)
       ├─ fix/<name>        Bug fixes (from develop or main)
       ├─ refactor/<name>   Code restructuring (from develop)
       └─ docs/<name>       Documentation (from develop)
```

- `main` is protected. Only merge via PR. Always tagged with semver.
- `develop` is the default branch for active work.
- Feature branches branch from `develop`, merge back to `develop`.
- Hotfixes (`fix/critical-*`) branch from `main`, merge to both `main` and `develop`.
- Delete branch after merge.

## Commit Convention

```
<type>: <short description>

<optional body — why, not what>
```

### Types

| Type       | When                          |
|------------|-------------------------------|
| `feat`     | New feature                   |
| `fix`      | Bug fix                       |
| `refactor` | Code change with no behavior change |
| `test`     | Adding or fixing tests        |
| `docs`     | Documentation only            |
| `chore`    | Build, CI, deps, tooling      |
| `perf`     | Performance improvement       |

### Rules

- Subject line: ≤ 72 chars, lowercase after colon, no period
- Body: wrap at 72 chars, explain **why** not what (the diff shows what)
- Reference issues: `Closes #12`, `Related: #34`
- One commit = one logical change. No "fix typo" fixup commits.
- Squash before merging to `develop` or `main`.

### Examples

```
feat: add mailbox relay pickup polling

Clients now poll the relay every 30s for queued messages.
Uses exponential backoff capped at 5min. Closes #88.

fix: reject messages with invalid signatures on relay

Relay was storing signed messages without verification.
Now drops and penalizes sender score.
```

## Pull Request Workflow

1. Create branch from `develop`: `git checkout -b feat/my-feature develop`
2. Make changes, commit following convention
3. Push: `git push -u origin feat/my-feature`
4. Open PR against `develop` with template:

```markdown
## Summary
<1-2 sentences>

## Changes
- <bullet point per logical change>

## Testing
- [ ] `go test ./...` passes
- [ ] `golangci-lint run ./...` passes
- [ ] Manual test: <what you did>

## Closes
Closes #<issue>
```

5. At least one review required (self-merge allowed for solo dev on non-main branches)
6. Squash-merge to `develop`
7. Delete remote branch

## Release Process

```bash
# From develop — bump version, tag, merge to main
git checkout main && git pull
git merge develop --no-ff
git tag -a v0.1.0 -m "v0.1.0 — initial lobby and DM"
git push origin main --tags
```

- Semver: `vMAJOR.MINOR.PATCH`
  - MAJOR: breaking protocol or DB schema changes
  - MINOR: new features, backwards-compatible
  - PATCH: bug fixes, no new features
- Each release tag triggers a build (future: CI publishes binaries)

## Code Review Standards

- Every commit compiles (`go build ./...`)
- Tests pass for affected packages
- No debug code, commented blocks, or `fmt.Print` left in
- Error handling: every returned error is checked or intentionally discarded (use `_ =` with comment)
- Imports follow stdlib → external → internal ordering
- No secrets, keys, or tokens in code

## Phase Tracking

Implementation phases are listed in `engineer-doc.md`. Each phase gets one or more feature branches:

```
Phase 3  →  feat/lobby-pubsub
Phase 4  →  feat/basic-tui
Phase 5  →  feat/dm-persistence
...
```

When a phase is complete and merged, mark it in `engineer-doc.md` and tag a minor release if warranted.

## Implementation Tasks

### CI pipeline

**Branch:** `chore/ci-pipeline`

**Files to create:**
```
.github/workflows/ci.yml
.github/workflows/release.yml
.pre-commit-config.yaml              # optional, for pre-commit hooks
```

**Tasks:**
- CI runs on push/PR to `develop` and `main`:
  1. `go build ./...` compiles
  2. `go test ./...` passes (skip integration by default)
  3. `golangci-lint run ./...` passes
  4. `go vet ./...` passes
- Release workflow on tag push `v*`:
  1. `go build -o alkalyne-${{ matrix.os }}-${{ matrix.arch }} ./cmd/alkalyne`
  2. Matrix: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
  3. Upload binaries as release artifacts

**Deliverables:**
- CI green on every push to `develop`
- Tagged release produces cross-compiled binaries for 5 targets
- No CGo dependencies — cross-compile works without toolchain pain

---

### `.golangci.yml` config (part of Phase 1)

**Branch:** `feat/project-skeleton` (shared with Foundation)

**File:** `.golangci.yml`

**Config:**
```yaml
linters:
  enable:
    - gofmt
    - goimports
    - govet
    - gosimple
    - errcheck
    - ineffassign
    - staticcheck
    - unused
issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

**Deliverables:**
- `golangci-lint run ./...` passes on all code
- `goimports` enforces import ordering (stdlib → external → internal)
