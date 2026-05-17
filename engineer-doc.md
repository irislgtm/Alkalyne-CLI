# Alkalyne-CLI — Software Engineer Document

Recommended stack: **Go + go-libp2p + Bubble Tea + SQLite** (see AGENTS.md for rationale).

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                     cmd/alkalyne/main.go                 │
│                     Parse flags, init subsystems         │
└──────┬──────────────┬──────────────┬──────────┬─────────┘
       │              │              │          │
  ┌────▼────┐   ┌────▼────┐   ┌────▼────┐ ┌───▼──────┐
  │internal/│   │internal/│   │internal/│ │internal/ │
  │  p2p    │   │  tui    │   │mailbox  │ │   db     │
  │         │   │         │   │         │ │          │
  │ Host    │   │ AppModel│   │ Offline │ │ Open/init│
  │ PubSub  │   │Contact  │   │ store-  │ │ Migrate  │
  │ Discov. │   │ Chat    │   │ forward │ │ CRUD all │
  │ Identity│   │ Input   │   │ relay   │ │          │
  └────┬────┘   └────┬────┘   │ pickup  │ └────┬─────┘
       │             │        └────┬────┘      │
       └──────┬──────┴──────────┬──┘           │
              │                 │              │
        ┌─────▼──────┐          │              │
        │internal/   │◄─────────┘──────────────┘
        │models      │
        │            │
        │ Message    │
        │ Contact    │
        │ Config     │
        └────────────┘
```

## Package Responsibilities

### `cmd/alkalyne/main.go`
- Parse flags (`--port`, `--no-tui`, `--data-dir`, `--bootstrap`)
- Determine mode: `client` (TUI), `daemon` (headless relay), or `relay-setup` (wizard)
- Initialise config (read/write `~/.alkalyne/config.toml`)
- Create/load Ed25519 identity key
- Instantiate libp2p host
- Open DB, run migrations
- Start TUI (Bubble Tea) or daemon relay mode
- Handle graceful shutdown (SIGINT/SIGTERM → close host → close DB)

### `internal/p2p/`
- `host.go` — create libp2p host with Noise transport, private key, listen addrs
- `pubsub.go` — create GossipSub, join/leave topics, handle message events. Auto-join `alkalyne/lobby` on startup. Track lobby presence via connected peer list on topic.
- `discovery.go` — mDNS for LAN, bootstrap peer connector with backoff
- `identity.go` — load or create Ed25519 key, wrap as `crypto.PrivKey`
- `protocol.go` — define protocol ID strings, message wire format (protobuf)
- `alias.go` — optional alias registry client: register nickname by signing with private key, lookup alias → PeerID via direct stream to bootstrap peer

Wire format for pubsub messages (protobuf):
```protobuf
syntax = "proto3";
package alkalyne;

message ChatMessage {
  string id = 1;              // UUID v4
  string sender_peer_id = 2;
  string recipient_peer_id = 3;  // "" for lobby messages
  string conversation_id = 4;    // "lobby" for lobby messages
  bytes encrypted_text = 5;   // E2E-encrypted with recipient's pubkey; plaintext for lobby
  int64 timestamp_ns = 6;
  // signed by sender's private key (signature covers fields 1,3,4,5,6)
  bytes signature = 7;
}

// Used between client ↔ mailbox relay (NOT gossipsub)
message MailboxMessage {
  string id = 1;
  string recipient_peer_id = 2;
  string sender_peer_id = 3;
  bytes encrypted_payload = 4; // serialized ChatMessage, encrypted to recipient
  int64 stored_at_ns = 5;
  bool ack = 6;                // true = "I picked this up"
}

// Alias registry protocol (direct stream to bootstrap peer)
message AliasRegister {
  string alias = 1;            // requested nickname (lowercase, alphanumeric)
  string peer_id = 2;
  bytes signature = 3;         // ed25519 sign(alias || peer_id) with requester's key
}

message AliasLookup {
  string alias = 1;
}

message AliasResponse {
  string alias = 1;
  string peer_id = 2;          // empty if not found
  string error = 3;
}
```

### `internal/tui/`
- `app.go` — root Bubble Tea model, manages sub-models, focus state, window size
- `contact_model.go` — `bubbles.List` model, filters by status/search query
- `chat_model.go` — manages Viewport for message history, TextInput for composing
- `relay_model.go` — relay list, status indicators, setup wizard steps
- `components.go` — StatusBar, HelpBar, MessageBubble, relay indicators via Lip Gloss
- `commands.go` — Bubble Tea `Cmd` functions: `sendMessageCmd`, `loadContactsCmd`, `refreshCmd`, `pollMailboxCmd`
- `styles.go` — Lip Gloss colour definitions and reusable style templates

### `internal/db/`
- `db.go` — open/close, connection pool config (WAL mode, busy timeout)
- `migrations/` — numbered `.sql` files, applied in order
- `messages.go` — InsertMessage, GetMessages(convID, limit, offset), DeleteConversation
- `contacts.go` — UpsertContact, GetContacts, DeleteContact
- `conversations.go` — GetOrCreateConversation, ListConversations

### `internal/models/`
- `message.go` — `Message` struct, `NewMessage()` constructor, `Verify()` method
- `contact.go` — `Contact` struct with status enum (Online/Offline/Pending)
- `config.go` — `Config` struct for TOML serialisation
- `relay.go` — `Relay` struct (PeerID, nickname, status, queued count, authorized)

### `internal/mailbox/`
- `relay.go` — relay daemon logic: listen for `MailboxMessage`, store encrypted blobs in DB, serve on pickup
- `client.go` — client logic: encrypt `ChatMessage` to recipient's pubkey, wrap in `MailboxMessage`, send to relay, handle ack and pickup polling
- `encrypt.go` — E2E encrypt/decrypt payloads using recipient Ed25519 public key (NaCl box or libp2p crypto helpers)
- `protocol.go` — protocol IDs for mailbox relay protocol (`/alkalyne/mailbox/1.0.0`)

## Database Schema

```sql
CREATE TABLE conversations (
    id          TEXT PRIMARY KEY,       -- UUID v4 or hash of participant PeerIDs
    peer_a      TEXT NOT NULL,          -- deterministic ordering: smaller PeerID first
    peer_b      TEXT NOT NULL,
    created_at  INTEGER NOT NULL,       -- Unix ns
    last_active INTEGER NOT NULL,       -- Unix ns
    UNIQUE(peer_a, peer_b)
);

CREATE TABLE messages (
    id              TEXT PRIMARY KEY,    -- UUID v4
    conversation_id TEXT NOT NULL REFERENCES conversations(id),
    sender_peer_id  TEXT NOT NULL,
    text            TEXT NOT NULL,
    timestamp_ns    INTEGER NOT NULL,
    signature       BLOB NOT NULL,
    local_status    TEXT NOT NULL DEFAULT 'sent',  -- sending, sent, delivered, read, failed, mailboxed
    delivered_via   TEXT,                          -- NULL or relay PeerID
    created_at      INTEGER NOT NULL
);

CREATE INDEX idx_messages_conv_timestamp ON messages(conversation_id, timestamp_ns);
CREATE INDEX idx_conversations_last_active ON conversations(last_active);

CREATE TABLE contacts (
    peer_id     TEXT PRIMARY KEY,
    nickname    TEXT NOT NULL DEFAULT '',
    added_at    INTEGER NOT NULL,
    last_seen   INTEGER,
    status      TEXT NOT NULL DEFAULT 'pending'  -- online, offline, pending
);

CREATE TABLE relays (
    peer_id     TEXT PRIMARY KEY,
    nickname    TEXT NOT NULL DEFAULT '',
    added_at    INTEGER NOT NULL,
    last_seen   INTEGER,
    status      TEXT NOT NULL DEFAULT 'offline',  -- online, offline
    is_self     INTEGER NOT NULL DEFAULT 0         -- 1 = this node is a relay itself
);

-- Mailbox stores encrypted blobs on relay nodes only
CREATE TABLE mailbox (
    id              TEXT PRIMARY KEY,    -- UUID v4
    recipient_peer_id TEXT NOT NULL,
    sender_peer_id    TEXT NOT NULL,
    encrypted_payload BLOB NOT NULL,     -- ChatMessage encrypted to recipient
    stored_at_ns    INTEGER NOT NULL,
    picked_up       INTEGER NOT NULL DEFAULT 0,
    picked_up_at_ns INTEGER
);

CREATE INDEX idx_mailbox_recipient ON mailbox(recipient_peer_id, picked_up);
```

## Data Flow: Sending a Message

### Direct delivery (recipient online)

1. User types text in InputField, presses Ctrl+E
2. `chat_model.go` constructs `models.Message` (UUID, sender PeerID, recipient PeerID, text, timestamp)
3. Signs message with host's private key → `signature`
4. Serialises to protobuf → publishes to GossipSub topic `alkalyne/chat/{conversation_id}`
5. Writes to local DB with `local_status = 'sent'`
6. Recipient's GossipSub handler in `pubsub.go` receives message
7. Verifies signature against sender's PeerID
8. Writes to DB with `local_status = 'delivered'`
9. Sends `ack` message back on control topic
10. Original sender receives ack → updates `local_status` to `'delivered'`
11. Both TUIs refresh via `tea.Batch(cmds...)` that reload from DB

### Mailbox delivery (recipient offline, relay available)

1. Steps 1-3 same as above
2. GossipSub publish fails or times out (no recipient on mesh)
3. `mailbox/client.go` encrypts the serialised `ChatMessage` to recipient's Ed25519 public key → `encrypted_payload`
4. Wraps in `MailboxMessage{id, recipient, sender, encrypted_payload, timestamp}`
5. Opens direct libp2p stream to relay peer using protocol `/alkalyne/mailbox/1.0.0`
6. Sends `MailboxMessage` to relay
7. Relay stores `encrypted_payload` in `mailbox` table (cannot decrypt — only recipient's key can)
8. Sender writes to local DB with `local_status = 'mailboxed'`, `delivered_via = relayPeerID`
9. Recipient comes online, connects to relay, sends `MailboxMessage{ack: true}` for each stored message
10. Relay marks `picked_up = 1`, returns encrypted blobs
11. Recipient decrypts with own private key, processes as normal message
12. Recipient's `ack` propagates back; sender updates status to `'delivered'`

## Discovery & Connectivity

### Method 1: Lobby (default, always on)
- Every client auto-joins topic `alkalyne/lobby` at startup
- Lobby messages are **not encrypted** (anyone in the topic can read them — they're public by design)
- Lobby messages are **not persisted** to DB (ephemeral)
- Presence tracked via GossipSub peer metadata on the lobby topic
- Lobby UI shows online count and peer list
- Clicking a lobby participant opens a DM conversation
- Like IRC: no server, no central registry, just a shared topic

### Method 2: Shareable identity link
- `alkalyne://12D3KooW...` URI scheme
- `Ctrl+L` copies link to clipboard
- Link can be pasted into any side channel (Signal, email, SMS, pastebin)
- Receiving side pastes into Alkalyne command (`Ctrl+V` or `:add alkalyne://...`)
- Protocol handler registration: future `xdg-open` / Windows URL scheme

### Method 3: Alias registry (optional, per bootstrap)
- Bootstraps peers may offer `/alkalyne/alias/1.0.0` protocol
- Register: `AliasRegister{alias, peer_id, signature}` — proof of ownership via Ed25519 sign
- Lookup: `AliasLookup{alias}` → `AliasResponse{peer_id}`
- Bootstrap validates uniqueness of alias and signature validity
- Not a central server — just a bootstrap peer offering it as a service. Users choose their bootstrap.
- Unregister/update: future

### Network Layer
- **LAN:** mDNS via `go-libp2p-discovery` — peers on same subnet auto-discover
- **WAN:** Bootstrap peers in config. Connect via multiaddr. Circuit relay v2 for NAT traversal.
- **DHT:** NOT used by default (privacy concern). Opt-in via `--dht` flag.
- **Mailbox relays:** Separate concept from circuit relay. Configured in `config.toml` under `[relays]`. Clients encrypt payloads before sending. Relays run headless (`alkalyne daemon`).

## Implementation Tasks

### Phase 2: Identity & libp2p host

**Branch:** `feat/identity-p2p`

**Files to create:**
```
internal/p2p/host.go
internal/p2p/identity.go
internal/p2p/identity_test.go
```

**Tasks:**
- `internal/p2p/identity.go` — load Ed25519 key from `~/.alkalyne/identity.key` or generate new one via `crypto/rand`. Wrap as `crypto.PrivKey`. Persist to disk on generation.
- `internal/p2p/host.go` — create libp2p `host.Host` with Noise transport, private key, multiaddrs from config. Configure QUIC + TCP transports.

**Deliverables:**
- `internal/p2p` package compiles with no CGo dependency
- `go test -v ./internal/p2p/...` — identity creation, loading, and round-trip test
- libp2p host starts, listens on configured ports, PeerID derived from key
- `golangci-lint run ./...` passes

**Depends on:** Phase 1 (project skeleton)

---

### Phase 3: LAN discovery + lobby pubsub

**Branch:** `feat/lobby-pubsub`

**Files to create:**
```
internal/p2p/pubsub.go
internal/p2p/discovery.go
internal/p2p/protocol.go
internal/p2p/pubsub_test.go  (build tag: integration)
```

**Tasks:**
- `internal/p2p/discovery.go` — mDNS discovery via `go-libp2p-discovery`. Emit discovered peer events.
- `internal/p2p/protocol.go` — define protocol IDs `/alkalyne/chat/1.0.0`, proto file for `ChatMessage`.
- `internal/p2p/pubsub.go` — create GossipSub router. Auto-join `alkalyne/lobby` on start. Publish/receive `ChatMessage`s on lobby topic. Expose `Subscribe(topic)` and `Publish(topic, msg)`.

**Deliverables:**
- Two instances on same LAN discover each other via mDNS
- Both auto-join `alkalyne/lobby` and see each other's presence
- Lobby messages exchanged in real time
- `go test --tags=integration ./internal/p2p/...` — two in-memory hosts exchange lobby messages

**Depends on:** Phase 2 (identity + host)

---

### Phase 6: WAN bootstrap + circuit relay

**Branch:** `feat/wan-bootstrap`

**Files to modify:** `internal/p2p/host.go`, `internal/p2p/discovery.go`

**Tasks:**
- Add bootstrap peer connector: connect to configured bootstrap peers with backoff (1s, 5s, 30s, 5min cap)
- Add `go-libp2p-circuit` relay transport for NAT traversal
- Bootstrap peer list loaded from config; can be updated at runtime

**Deliverables:**
- Instance connects to bootstrap peers on WAN
- Circuit relay v2 enabled for NAT traversal
- Test: two instances behind NAT connect via a known relay

**Depends on:** Phase 3 (pubsub working)

---

### Phase 7: Message signing + verification

**Branch:** `feat/message-signing`

**Files to create:**
```
internal/p2p/validation.go
internal/p2p/validation_test.go
```

**Tasks:**
- Add signature field (Ed25519) to `ChatMessage` protobuf
- Sign messages before publish: `sign(peerID || conversationID || text || timestamp)`
- Verify on receive: drop message if signature invalid, publish score penalty
- GossipSub validation hook via `WithMessageSignaturePolicy` and custom `validator`

**Deliverables:**
- Every sent message carries a valid Ed25519 signature
- Invalid/missing signature → message dropped, not stored, not forwarded
- Signed messages verified before display

**Depends on:** Phase 3 (pubsub)

---

### Phase 9: Alias registry

**Branch:** `feat/alias-registry`

**Files to create:**
```
internal/p2p/alias.go
internal/p2p/alias_test.go
internal/models/alias.go
pkg/alias/          # reusable alias registry logic
```

**Tasks:**
- Define `AliasRegister`, `AliasLookup`, `AliasResponse` protobuf messages
- Implement registry server (runs on bootstrap peer): accept `AliasRegister`, verify Ed25519 signature, store unique alias→PeerID mapping in SQLite
- Implement registry client: `Register(alias)`, `Lookup(alias)` via direct libp2p stream to bootstrap

**Deliverables:**
- `:register alice` sends signed registration to bootstrap, gets success/failure
- `:lookup alice` returns PeerID or "not found"
- Bootstrap cannot forge registrations (signature required)
- Duplicate alias rejected

**Depends on:** Phase 7 (signing verifies registration) + `internal/db` schema

---

### Phase 10: Mailbox relay daemon

**Branch:** `feat/mailbox-daemon`

**Files to create:**
```
internal/mailbox/relay.go
internal/mailbox/protocol.go
internal/mailbox/relay_test.go
cmd/alkalyne/daemon.go       # or add to main.go dispatch
```

**Tasks:**
- Protocol ID `/alkalyne/mailbox/1.0.0` for client↔relay direct streams
- Relay daemon logic: listen for incoming streams, accept `MailboxMessage`, store `encrypted_payload` in `mailbox` table
- On `ack` from recipient: mark `picked_up = 1`, return encrypted blob
- Cleanup old unpicked messages after TTL (24h configurable)

**Deliverables:**
- `./alkalyne daemon` starts headless relay, listens on configured port
- Accepts `MailboxMessage` from clients, stores in DB
- Returns stored messages on recipient pickup
- Authenticated by PeerID (relay only accepts from known peers) — optional allowlist

**Depends on:** Phase 6 (host with WAN reachability) + `internal/db` migrations

---

### Phase 11: Mailbox client

**Branch:** `feat/mailbox-client`

**Files to create:**
```
internal/mailbox/client.go
internal/mailbox/encrypt.go
internal/mailbox/client_test.go
```

**Tasks:**
- `internal/mailbox/encrypt.go` — NaCl box (Curve25519+XSalsa20+Poly1305): encrypt `ChatMessage` protobuf to recipient's public key, decrypt with own private key
- `internal/mailbox/client.go` — detect offline recipient (GossipSub publish timeout), fall back to mailbox relay:
  1. Encrypt `ChatMessage` to recipient's public key
  2. Open stream to relay, send `MailboxMessage`
  3. Mark local message status as `mailboxed`
- On reconnect: poll relay for queued messages, decrypt, process as normal

**Deliverables:**
- Offline messages are E2E-encrypted before reaching relay
- Relay cannot decrypt (only recipient's private key works)
- On reconnect, recipient picks up messages from relay
- `local_status = 'mailboxed'` shown in UI until delivered

**Depends on:** Phase 10 (relay daemon) + Phase 7 (signing produces valid ChatMessages)

---

### Phase 13: Delivery receipts (via mailbox)

**Branch:** `feat/delivery-receipts`

**Files to modify:** `internal/mailbox/client.go`, `internal/mailbox/relay.go`

**Tasks:**
- When recipient picks up from relay, relay sends `MailboxMessage{ack: true}` back to sender
- Sender updates `local_status` from `mailboxed` to `delivered`
- If recipient reads the message (in TUI), send a `read` receipt via GossipSub control topic

**Deliverables:**
- Sender sees `>>` (delivered) once recipient picks up from relay
- Read receipts: future (low priority)

**Depends on:** Phase 11 (mailbox client) + Phase 4 (TUI displays status)

---

### Phase 14: Search + contact management (backend)

**Branch:** `feat/search-contacts`

**Files to modify:** `internal/db/messages.go`, `internal/db/contacts.go`, `internal/p2p/pubsub.go`

**Tasks:**
- DB: full-text search across messages (FTS5 table or `LIKE` for MVP)
- DB: contact CRUD with `DELETE` cascade
- Backend search endpoint for TUI: `SearchMessages(query, limit, offset)`

**Deliverables:**
- `/search` returns matching messages from local history
- Contact add/remove/list works via backend

**Depends on:** Phase 5 (DB populated with messages)

---

### Phase 15-16: Polishing

**Branch:** `feat/polish`

**Tasks:**
- Reconnection: on disconnect, retry with exponential backoff, rejoin lobby topics
- Resource-constrained terminals: handle < 60 cols gracefully
- Rate-limited logging: debug logs that don't flood
- Error reporting: surface connection errors in TUI without crashing

**Depends on:** All prior phases

## Testing Strategy

| Layer    | Approach                                        | Tag          |
|----------|-------------------------------------------------|--------------|
| Models   | Unit tests, table-driven                        | (none)       |
| DB       | `t.TempDir()` for DB path, `exec.Schema`        | (none)       |
| P2P      | Two in-memory libp2p hosts in same test         | `integration`|
| TUI      | `tea.NewProgram` with mock messages             | (none)       |
| E2E      | Two OS processes, assert message delivery       | `e2e`        |

## Configuration (`~/.alkalyne/config.toml`)

```toml
data_dir = "~/.alkalyne"
listen_addrs = ["/ip4/0.0.0.0/tcp/9000", "/ip4/0.0.0.0/udp/9000/quic-v1"]
bootstrap_peers = []

[relays]
  # Mailbox relays for offline delivery
  [relays.my-pi]
    peer_id = "12D3KooW..."
    addrs = ["/ip4/192.168.1.50/tcp/9001"]
    enabled = true

nickname = "iris"
```
