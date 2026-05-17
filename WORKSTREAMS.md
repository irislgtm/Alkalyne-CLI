# Workstreams — Parallel Agentic Development

## Overview

Five parallel workstreams derived from the six spec files. Each workstream has its own agent, spec file, and branch prefix. Foundation (WS-A) must finish first; WS-B through WS-E then run in parallel.

```
Phase 1 ──→ Foundation (WS-A)
                │
        ┌───────┼───────┬───────┐
        ▼       ▼       ▼       ▼
     Network   UI     Security  CI
     (WS-B)  (WS-C)   (WS-D)  (WS-E)
```

## Workstream assignments

| WS | Name | Spec File | Branch Prefix | Agent |
|---|---|---|---|---|
| A | **Foundation** | `AGENTS.md` | `feat/` | Agent A |
| B | **Networking + DB** | `engineer-doc.md` | `feat/` | Agent B |
| C | **UI** | `designer-doc.md` | `feat/` | Agent C |
| D | **Security** | `security-doc.md` | `sec/` | Agent D |
| E | **CI/CD** | `git-workflow.md` | `chore/` | Agent E |

## Phase → Workstream mapping

```
Phase   Feature                    WS   Branch
──────────────────────────────────────────────────────
1       Project skeleton           A    feat/project-skeleton
2       Identity + p2p host        B    feat/identity-p2p
3       LAN + lobby pubsub         B    feat/lobby-pubsub
4       Minimal TUI                C    feat/basic-tui
5       DM conversations           C    feat/dm-conversations
6       WAN bootstrap              B    feat/wan-bootstrap
7       Message signing            B    feat/message-signing
8       Invite link                C    feat/invite-link
9       Alias registry             B    feat/alias-registry
10      Mailbox relay daemon       B    feat/mailbox-daemon
11      Mailbox client             B    feat/mailbox-client
12      TUI relay management       C    feat/tui-relays
13      Delivery receipts          B    feat/delivery-receipts
14      Search + contacts          C    feat/search-ui
15-16   Polish                     C    feat/polish
```

### Security (WS-D) cross-cuts

Security workstream has its own branches that review and harden each phase as it lands:

```
Phase 2   →  sec/identity-hardening
Phase 7   →  sec/message-signing
Phase 9   →  sec/alias-registry
Phase 10-11 → sec/mailbox-encryption
Phase 13  →  sec/delivery-receipts
Global    →  sec/file-permissions
```

### CI (WS-E) is a single branch

```
All phases →  chore/ci-pipeline  (can be done any time after Phase 1)
```

## Dependency graph

```
WS-A: Phase 1 (skeleton)
 │
 ├──► WS-B: Phase 2 (identity) → Phase 3 (pubsub)
 │       │                        │
 │       │                        ├──► Phase 6 (WAN)
 │       │                        ├──► Phase 7 (signing) → Phase 9 (alias)
 │       │                        └──► Phase 10 (relay) → Phase 11 (client) → Phase 13 (receipts)
 │
 ├──► WS-C: Phase 4 (TUI) → Phase 5 (DM view)
 │       │                    │
 │       │                    ├──► Phase 8 (invite link)
 │       │                    ├──► Phase 12 (relay UI)    ← depends on Phase 10
 │       │                    └──► Phase 14 (search UI)
 │
 ├──► WS-D: can start reviewing Phase 2+ as they land
 │
 └──► WS-E: CI pipeline — needs Phase 1 go.mod to exist, otherwise independent
```

## Coordination rules

1. **WS-A must finish first.** Phases 2-14 wait for the project skeleton.
2. **WS-B and WS-C start in parallel** after WS-A completes.
3. **WS-D reviews each phase as its branch is opened for PR.** Security agent does NOT need to wait for a phase to be merged — review the branch directly.
4. **WS-E is independent** after Phase 1. CI config doesn't depend on application code.
5. **Branches merge to `develop`**, not directly to `main`. See `git-workflow.md` for PR process.
6. **When WS-B and WS-C both need the same thing** (e.g. internal/db schema), the agent that gets there first implements it. The other agent imports from `internal/db/` rather than reimplementing.
7. **Interface contracts** between workstreams must be stable before parallel work starts. The key interfaces are:
   - `internal/db/` — queries for messages, contacts, conversations (used by both B and C)
   - `internal/p2p/pubsub.go` — `Subscribe(topic)`, `Publish(topic, msg)`, message event channel (used by C)
   - `internal/models/` — shared structs (used by everyone)

## File ownership

| File | Owner |
|---|---|
| `cmd/alkalyne/main.go` | WS-A (then modified by WS-B for daemon mode) |
| `internal/config/` | WS-A |
| `internal/models/` | WS-A (extended by WS-B) |
| `internal/db/` | WS-B (imported by WS-C, WS-D) |
| `internal/p2p/` | WS-B (imported by WS-C) |
| `internal/mailbox/` | WS-B |
| `internal/tui/` | WS-C |
| `pkg/` | WS-B (alias registry) |
| `~/.alkalyne/` (runtime) | Everyone's code reads/writes this |

## Merging sequence

Recommended order for merging feature branches to `develop`:

```
Round 1: feat/project-skeleton
         feat/identity-p2p
Round 2: feat/lobby-pubsub
         feat/basic-tui
Round 3: feat/dm-conversations
         feat/message-signing
         feat/wan-bootstrap
         chore/ci-pipeline
Round 4: feat/invite-link
         feat/alias-registry
Round 5: feat/mailbox-daemon
         feat/tui-relays
Round 6: feat/mailbox-client
Round 7: feat/delivery-receipts
         feat/search-ui
Round 8: feat/polish
```

Security branches (`sec/*`) merge alongside or immediately after their corresponding feature branch.
