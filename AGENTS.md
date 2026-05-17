# AGENTS.md — Alkalyne-CLI

P2P messaging CLI written in **Go**, using **go-libp2p** for networking, **Bubble Tea** for TUI, and **SQLite** for local storage.

## Stack

| Layer            | Choice                    | Why                                      |
|------------------|---------------------------|------------------------------------------|
| Language         | Go 1.23+                  | Single-binary cross-compilation, mature libp2p |
| P2P networking   | go-libp2p + GossipSub     | Most mature libp2p impl, built-in pubsub |
| TUI framework    | Bubble Tea (Charm)        | Elm-arch, rich ecosystem (Bubbles, Lip Gloss) |
| Local storage    | SQLite (modernc.org/sqlite)| Zero-dep, pure Go, single-file DB        |
| Encryption       | libp2p Noise transport    | Built into go-libp2p, forward secrecy    |
| Identity         | libp2p PeerID (Ed25519)   | Built into go-libp2p                     |

## Project Structure

```
cmd/alkalyne/       — main.go (entrypoint)
internal/
  tui/              — Bubble Tea models, views, updates
  p2p/              — libp2p host setup, pubsub handlers, peer discovery
  mailbox/          — offline message store-and-forward (relay nodes)
  db/               — SQLite schema, queries, migrations
  models/           — shared domain types (Message, Contact, etc.)
  config/           — config file read/write, defaults
pkg/                — public library API (if any)
```

## Modes

```
go run ./cmd/alkalyne            # client mode — TUI for messaging
go run ./cmd/alkalyne daemon     # relay mode — headless store-and-forward node
go run ./cmd/alkalyne relay-setup  # guided TUI wizard to configure a relay
```

## Commands

```
go build ./cmd/alkalyne          # build binary to current dir
go build -o alkalyne ./cmd/alkalyne
go run ./cmd/alkalyne --help
```

### Testing

```
go test ./...                    # all packages
go test ./internal/db/...        # DB tests only (use t.TempDir for DB path)
go test -v -run TestP2PConnect ./internal/p2p/   # single test
```

Tests requiring two running nodes have `//go:build integration` tag. Skip with `go test ./...` (default), run with `go test --tags=integration ./...`.

### Lint & Typecheck

```
golangci-lint run ./...
```

## Conventions

- **Imports:** stdlib -> external -> internal (3 groups, blank line between)
- **Errors:** return `fmt.Errorf("pkg: %w", err)` with `%w`, never `errors.Wrap`
- **Bubble Tea models:** one file per model (`chat_model.go`, `contact_model.go`), `Update` returns `(Model, Cmd)` always
- **DB:** raw SQL via `database/sql`, no ORM. Migrations in `internal/db/migrations/` as numbered `.sql` files
- **PeerID persistence:** Ed25519 key saved to `~/.alkalyne/identity.key` on first run
- **Config:** TOML in `~/.alkalyne/config.toml` (bootstrap peers, listen addrs, data dir)
- **Discovery methods (all serverless):**
  1. **Lobby** — every client auto-joins topic `alkalyne/lobby`. Users see who's online, click to DM. Like IRC.
  2. **Shareable link** — `alkalyne://{peer_id}` copied via `Ctrl+L`, pasted anywhere to add contact.
  3. **Alias registry** — optional on bootstrap peer: register nickname by signing it with your key. No central server, just a bootstrap peer offering it as a service.

## Go Dependencies

```
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
github.com/libp2p/go-libp2p
github.com/libp2p/go-libp2p-pubsub
github.com/libp2p/go-libp2p-circuit
modernc.org/sqlite
github.com/BurntSushi/toml
```

## Gotchas

- **Cross-compile:** `GOOS=windows GOARCH=amd64 go build ./cmd/alkalyne` — pure Go SQLite via `modernc.org/sqlite` avoids CGo, so no cross-compile pain. `mattn/go-sqlite3` requires CGo and cross-compiler toolchains — do NOT use it.
- **libp2p circuit relay (NAT):** behind NAT? Enable circuit relay v2. Add well-known relay peers to `config.toml`.
- **Mailbox relay (offline delivery):** separate concept from circuit relay. Mailbox relays are designated store-and-forward nodes. Messages are **E2E-encrypted before reaching the relay** using recipient's Ed25519 public key (libp2p Noise keys). Relay sees encrypted blobs only.
- **Relay-setup wizard:** `go run ./cmd/alkalyne relay-setup` launches a TUI wizard that configures a headless relay daemon on another machine. The wizard generates a `--relay-config` flag for the target machine.
- **Bubble Tea alt-screen:** full-window mode by default. `tea.AltScreen()` on start. `--no-tui` flag for pipe-friendly mode.
- **Discovery:** use `go-libp2p-discovery` with mDNS for LAN, bootstrap peers for WAN. Do not bundle public DHT discovery by default (privacy).
- **Disk:** config/data lives in `~/.alkalyne/`. Respect `XDG_DATA_HOME` and `XDG_CONFIG_HOME` on Linux.

## Implementation Tasks

### Phase 1: Project skeleton (prerequisite for all other workstreams)

**Branch:** `feat/project-skeleton`

**Files to create:**
```
cmd/alkalyne/main.go               # entrypoint, flag parsing, mode dispatch
internal/config/config.go           # TOML read/write, defaults, ~/.alkalyne/
internal/config/config_test.go
internal/models/message.go          # Message struct, NewMessage, Verify
internal/models/contact.go          # Contact struct with status enum
internal/models/config.go           # Config struct for TOML serialisation
internal/models/relay.go            # Relay struct
go.mod                              # module github.com/alkalyne/alkalyne
.golangci.yml                       # lint config
```

**Deliverables:**
- `go build ./cmd/alkalyne` compiles
- `./alkalyne --help` prints usage with all flags
- `./alkalyne` creates `~/.alkalyne/config.toml` with defaults on first run
- `./alkalyne` respects `--data-dir`, `--no-tui`, `--port` flags
- Mode dispatch works: `./alkalyne` → TUI mode, `./alkalyne daemon` → relay mode, `./alkalyne relay-setup` → wizard mode
- `golangci-lint run ./...` passes
- `go test ./internal/models/...` passes

**Depends on:** Nothing (first phase)

**Agent handoff:** After this phase, agents for Phases 2-4 can all start in parallel.

## Enforced Rules

These are not guidelines — they are enforced by automated checks and will fail CI.

### Import boundaries (enforced by `internal/archtest/imports_test.go`)

```
models    → may NOT import any internal/ package
config    → may NOT import db, p2p, tui, mailbox
db        → may NOT import p2p, tui, mailbox, config
p2p       → may NOT import tui, mailbox, db
tui       → may NOT import mailbox
mailbox   → may NOT import tui
```

Any violation of these rules causes `go test ./internal/archtest/...` to fail.

### Linting (enforced by `.golangci.yml`)

All linters are strict and will fail CI:

| Linter | What it catches |
|--------|----------------|
| `dupl` | Duplicated code blocks (>80 lines similar) |
| `gocyclo` | Functions with cyclomatic complexity >15 |
| `gocognit` | Functions with cognitive complexity >20 |
| `goconst` | Repeated strings that should be constants (3+ occurrences) |
| `prealloc` | Slices that could be pre-allocated |
| `unconvert` | Unnecessary type conversions |
| `nakedret` | Naked returns in functions >30 lines |
| `predeclared` | Variable names that shadow Go predeclared identifiers |
| `thelper` | Test helpers not calling `t.Helper()` |

### Pre-commit hook (`.git/hooks/pre-commit` → `scripts/pre-commit.sh`)

Runs before every commit:
1. `gofmt -l -e .` — formatting
2. `go vet ./...` — suspicious constructs
3. `go vet -race ./...` — data races
4. `go test ./internal/archtest/...` — import boundaries
5. `go test ./internal/...` — unit tests
6. `go mod tidy && git diff --exit-code go.mod go.sum` — clean deps
7. `golangci-lint run ./...` — all linters

Install: `ln -sf ../../scripts/pre-commit.sh .git/hooks/pre-commit`

### DRY rules (human review, automated help from `dupl` + `goconst`)

- If you write the same logic twice, extract it. The `dupl` linter flags blocks >80 lines that look similar.
- If you write the same string literal 3+ times, make it a `const`. `goconst` catches this.
- If you write the same error pattern 3+ times, make a `var ErrFoo = errors.New("...")`.
- Never copy-paste test setup — use `t.Helper()` factory functions.
