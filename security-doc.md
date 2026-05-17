# Alkalyne-CLI — Security Analyst Document

## Threat Model

### Assumptions
- **Local machine is trusted.** The user's filesystem, memory, and terminal are secure. No sandboxing against local malware.
- **Network is untrusted.** All traffic travels over the open internet. Eavesdroppers, active MITM, and traffic analysis are in scope.
- **No central authority.** No PKI, no CA, no server to revoke or rotate keys. Identity is self-sovereign.
- **Metadata exposed.** GossipSub topics, message timing, and peer IP addresses are visible to the network. Content is not.

### Assets
| Asset              | Sensitivity       | Storage                          |
|--------------------|-------------------|----------------------------------|
| Ed25519 private key| **Critical**      | `~/.alkalyne/identity.key`      |
| Message history    | **High**          | `~/.alkalyne/data.db`            |
| Contact list       | **Medium**        | `~/.alkalyne/data.db`            |
| Peer IP addresses  | **Low-medium**    | libp2p peerstore (in-memory)     |
| Mailbox payloads (on relay) | **None (encrypted)** | `mailbox` table on relay DB |

### Attack Surface (ranked)

1. **Inbound libp2p connections** — Noise handshake, protocol negotiation
2. **GossipSub messages** — malformed protobufs, flood attacks, spam
3. **Mailbox relays** — malicious relay could drop messages, log metadata, or attempt decryption
4. **Bootstrap peers** — malicious bootstrap directing to eclipsed network
5. **mDNS discovery** — LAN injection of fake peers (bailiwick attack)
6. **Config/identity file** — filesystem permissions
7. **Local SQLite DB** — plaintext message history at rest

## Encryption Architecture

### Transport Layer (built into go-libp2p)

```
┌─ Peer A ─┐                    ┌─ Peer B ─┐
│ Ed25519   │─── Noise XX ─────→│ Ed25519   │
│ privKey   │←── handshake ─────│ privKey   │
│           │   ───────────────→│           │
│           │   ChaCha20Poly1305│           │
│           │←──────────────────│           │
└───────────┘                   └───────────┘
```

- All libp2p connections use **Noise XX** handshake with **ChaCha20Poly1305** AEAD.
- Provides: mutual authentication, forward secrecy, integrity, confidentiality.
- PeerID is the hash of the Ed25519 public key — cryptographic binding between identity and connection.

### Message Layer (application-level signing)

Each `ChatMessage` protobuf carries a signature over fields 1-6:

```
to_sign = id || sender_peer_id || recipient_peer_id ||
          conversation_id || text || timestamp_ns
signature = Ed25519.Sign(privKey, to_sign)
```

Recipient verifies: `Ed25519.Verify(peerID.PublicKey, to_sign, signature)`.

### Mailbox Layer (E2E before relay)

```
┌─ Sender ─┐                   ┌─ Relay ─┐              ┌─ Recipient ─┐
│ plaintext │                  │          │              │             │
│     ↓     │                  │          │              │             │
│ Encrypt   │──MailboxMessage─→│ Encrypted│──on pickup──→│ Decrypt     │
│ to recip. │                  │ blobs    │              │     ↓       │
│ pubkey    │                  │ (can't   │              │ plaintext   │
└───────────┘                  │  read)   │              └─────────────┘
                               └──────────┘
```

- Before sending to a mailbox relay, the `ChatMessage` protobuf is encrypted using **NaCl box** (Curve25519+XSalsa20+Poly1305) with recipient's public key.
- Relay stores only the `encrypted_payload` — it cannot decrypt.
- Relay sees: sender PeerID, recipient PeerID, timestamp, blob size. **Not** message content.
- Recipient's client downloads encrypted blobs on reconnect, decrypts locally.

### At Rest

| Data                      | Protection                    |
|---------------------------|-------------------------------|
| Ed25519 private key       | File permissions 0600         |
| SQLite database           | File permissions 0600         |
| Message content in DB     | **Plaintext** (local machine trusted — see assumptions) |
| Mailbox payload on relay  | **E2E-encrypted** (relay cannot read) |

**Rationale for plaintext DB:** Encrypting the DB adds key management complexity (where to store the DB encryption key?) with no benefit under the trusted-local-machine assumption. If user wants encrypted-at-rest, they should use full-disk encryption (LUKS, BitLocker, FileVault). Future option: optional SQLite SEE or `sqlcipher`.

## Identity

- **No usernames or passwords.** Identity = Ed25519 keypair. PeerID = hash of public key (e.g. `12D3KooW...`).
- **First run:** generate key, save to `~/.alkalyne/identity.key` (0600).
- **Nicknames** are local-only aliases stored in the contacts table. They are never sent over the wire.
- **No recovery mechanism.** Lose identity.key = lose your identity. Display a warning on first run: `BACK UP ~/.alkalyne/identity.key`.

## Spam & Abuse Mitigation

| Attack                | Mitigation                                    |
|-----------------------|-----------------------------------------------|
| Message flood         | GossipSub peer scoring (P1-P7 parameters)     |
| Malformed protobuf    | Reject on deserialisation failure, score penalty|
| Sybil attack          | Limited by no DHT — only known/relayed peers  |
| Spam from known peer  | Local blocklist, manual per-peer mute         |
| Replay attack         | Deduplicate by message ID, reject duplicates  |

### Mailbox-specific attacks

| Attack                       | Mitigation                                    |
|------------------------------|-----------------------------------------------|
| Relay drops messages         | Sender retries with multiple relays (configurable) |
| Relay logs metadata          | Metadata exposure accepted (IPs, timing). Content E2E-protected. |
| Relay attempts decryption    | Impractical: NaCl box with Curve25519. Relay would need recipient's private key. |
| Sybil relay injection        | Relays are explicitly configured (peer ID allowlist), not discovered |
| Replay via relay             | Message ID dedup + timestamp freshness check  |

GossipSub v1.2+ scoring defaults:
```go
params.TopicWeights["alkalyne/chat"] = 0.5
params.P1 = 0.5  // time in mesh
params.P2 = 0    // first message deliveries (not applicable for chat)
params.P3 = 0.2  // mesh message rate
params.P4 = 0.2  // mesh message rate (invalid)
params.P5 = -0.1 // application-specific penalty threshold
```

## Privacy Considerations

1. **IP leakage:** Peer IPs are visible to connected peers. Use Tor/I2P for anonymity (future: `--proxy socks5://127.0.0.1:9050`).
2. **DHT disabled by default.** Enabling DHT (`--dht`) exposes PeerID-IP mappings to the global network.
3. **Lobby is public.** Everyone on the network sees who's in the lobby, their PeerIDs, and all lobby messages. Lobby messages are plaintext by design. Do not send sensitive information in lobby. This is equivalent to joining an IRC channel.
4. **DM topics are public** (but encrypted). Anyone who knows the conversation ID can subscribe to the topic and see message *metadata* — but cannot decrypt or forge signatures.
5. **Traffic analysis.** GossipSub message timing and size leak conversational patterns. Padding future: fixed-size message frames.
6. **Alias registry.** If using a bootstrap peer's alias service, the bootstrap learns the mapping between your human-readable alias and your PeerID. This is a tradeoff: convenience for privacy. Users should choose a bootstrap peer they trust, or skip alias registration entirely (the lobby + invite link methods require no registry).

## Config & Key File Security

```bash
# Expected permissions on first run:
~/.alkalyne/
├── identity.key    # 0600, owned by user
├── config.toml     # 0600 (contains listen addresses, bootstrap peers)
└── data.db         # 0600 (SQLite, WAL + SHM files masked by same perms)
```

- On startup, verify permissions. If too permissive (`group`/`other` readable), log warning and `os.Chmod` to 0600.
- On Windows: use `ACL` equivalent via `golang.org/x/sys/windows`.

## Threat Response (non-exhaustive)

### If identity.key is compromised
1. Generate a new keypair (`alkalyne identity rotate`)
2. Manually notify all contacts of the new PeerID
3. Old messages are still readable by attacker (they were encrypted with Noise session keys, but attacker had private key → could have MITM'd)

### If SQLite DB is stolen
Attacker reads all message history. Mitigation: full-disk encryption. Future: optional `sqlcipher` integration.

### If bootstrap peer goes rogue
- Bootstrap can see connecting peers' IPs and timestamps
- Bootstrap cannot forge messages (no Ed25519 key)
- Bootstrap cannot decrypt messages (Noise session keys per-connection)
- Mitigation: run your own bootstrap, or use multiple bootstrap peers

### If mailbox relay goes rogue
- Relay sees: sender/recipient PeerIDs, message count/timing, encrypted blob sizes
- Relay **cannot** read or forge messages (E2E encryption + signatures)
- Relay **can** drop messages → mitigation: configure multiple relays, sender retries across them
- Mitigation: relay must be explicitly authorized by PeerID in config. No auto-discovery of relays.

## Future Security Features (not in v1)

| Feature                    | Priority | Notes                                    |
|----------------------------|----------|------------------------------------------|
| `sqlcipher` for DB-at-rest | Medium   | Key derived from identity.key            |
| Perfect Forward Secrecy for history | Low | Ratchet (Signal protocol) per conversation |
| Tor/I2P proxy support      | Medium   | SOCKS5 proxy dial option                 |
| Message recall             | Low      | Delete from all peers (best-effort)      |
| Ephemeral conversations    | Low      | Messages deleted after expiry TTL        |
| Deniable handshake         | Low      | Noise `XK` pattern instead of `XX`       |
| Multi-relay redundancy     | Medium   | Send to N relays, first ack wins         |
| Relay reputation scores    | Low      | Track relay reliability, auto-exclude poor relays |

## Verification Checklist

- [ ] Ed25519 keys generated with `crypto/rand` (not `math/rand`)
- [ ] Noise handshake is **required** for all connections (libp2p default; verify no `NoSecurity` option)
- [ ] Message signature verification failure → message **dropped**, not stored
- [ ] Mailbox payload encrypted **before** sending to relay (verify relay never sees plaintext)
- [ ] Mailbox decryption uses recipient's private key **only** — no key sharing with relay
- [ ] SQLite DB permissions set to 0600 on open
- [ ] Identity key permissions set to 0600 on write
- [ ] Relays have explicit PeerID allowlist — no anonymous relay usage
- [ ] Lobby messages are NOT encrypted (plaintext by design) — verify no E2E for lobby topic
- [ ] Lobby messages are NOT persisted to DB
- [ ] Alias registry verifies Ed25519 signature before accepting registration
- [ ] Alias registry enforces uniqueness (one alias per peer ID)
- [ ] No logging of private keys or message plaintext (debug logs may log peer IDs)
- [ ] `t.Parallel()` safe in DB tests (each test uses `t.TempDir()`)
- [ ] SECURITY.md file at repo root with contact instructions for vulnerabilities

## Implementation Tasks

This workstream runs **alongside every phase**, verifying and hardening as code lands. Each task maps to one or more engineering phases.

### Ongoing: Code review + verification (all phases)

- Review every PR against the verification checklist (above)
- Verify: no `NoSecurity` option passed to libp2p
- Verify: all `crypto/rand` usage (not `math/rand`)
- Verify: no secrets/keys logged at any log level
- Verify: imports follow stdlib → external → internal order

### Phase 2: Identity hardening

**Branch:** `sec/identity-hardening`

**Files to create/modify:**
```
internal/p2p/identity.go           # review/modify
internal/p2p/identity_test.go      # add security tests
```

**Tasks:**
- Verify Ed25519 key is generated with `crypto/rand`
- Verify key file permissions set to 0600 on write
- On Windows: verify ACL-equivalent via `golang.org/x/sys/windows`
- Test: corrupt or missing identity.key produces clear error, not panic
- Test: invalid key file (wrong format) produces clear error
- Verify: terminal displays backup warning on first run

**Deliverables:**
- `go test -v ./internal/p2p/...` — key security tests pass

---

### Phase 3: Lobby security review

No separate branch. Review-only:

- Verify lobby messages are NOT encrypted (plaintext by design)
- Verify lobby messages are NOT persisted to DB
- Verify GossipSub peer scoring defaults protect against floods

---

### Phase 7: Signing implementation

**Branch:** `sec/message-signing`

**Files to create:**
```
internal/p2p/validation.go         # review
internal/p2p/validation_test.go    # security-focused tests
```

**Tasks:**
- Verify signing covers: `id || sender_peer_id || conversation_id || text || timestamp_ns` (NOT recipient — lobby has no recipient)
- Verify verification failure → message **dropped**, not stored, not forwarded
- Verify replay protection: message ID dedup (UUID v4, reject duplicate IDs within TTL window)
- Verify GossipSub validator function is registered and enforced
- Test: malformed signature → drop
- Test: missing signature → drop
- Test: replayed message with same UUID → drop
- Test: correct signature → accept and forward

**Deliverables:**
- `go test -v ./internal/p2p/...` — signing verification security tests pass

---

### Phase 9: Alias registry security review

**Branch:** `sec/alias-registry`

**Tasks:**
- Verify `AliasRegister` signature proves ownership: `sign(alias || peer_id)` with requester's Ed25519 key
- Verify registry rejects duplicate aliases (one alias per peer ID, one peer ID per alias)
- Verify registry validates alias format (lowercase alphanumeric, 3-32 chars)
- Verify registry returns error for invalid signature, not a panic
- Test: register with wrong key → rejected
- Test: register duplicate → rejected

---

### Phase 10-11: Mailbox relay E2E encryption

**Branch:** `sec/mailbox-encryption`

**Files to create:**
```
internal/mailbox/encrypt.go        # review
internal/mailbox/encrypt_test.go   # security tests
internal/mailbox/client_test.go    # add security checks
```

**Tasks:**
- Verify NaCl box (Curve25519+XSalsa20+Poly1305) implementation is correct
- Verify `encrypted_payload` in `mailbox` table cannot be decrypted by relay
- Verify recipient's private key is the only key that can decrypt
- Verify encrypted payload includes authentication tag (NaCl box does this natively)
- Test: encrypt with Alice's pubkey → Bob's privkey cannot decrypt
- Test: encrypt → decrypt round-trip succeeds
- Test: tampered payload → decryption fails (auth tag mismatch)

**Deliverables:**
- Mailbox encryption NaCl box implementation verified
- Cross-key decryption test passes (attacker cannot decrypt)
- Relay integration test: relay stores encrypted blobs, confirms it cannot read plaintext

---

### Phase 13: Delivery receipt spoofing resistance

**Branch:** `sec/delivery-receipts`

**Tasks:**
- Verify delivery `ack` messages are signed by the relay or recipient
- Verify sender only updates status when a valid signed `ack` is received
- Test: forged ack → rejected
- Test: valid ack → status updated

---

### Global: File permissions hardening

**Branch:** `sec/file-permissions`

**Files to modify:** `cmd/alkalyne/main.go`

**Tasks:**
- On startup, check `~/.alkalyne/identity.key` permissions
- If group/other readable → `os.Chmod` to 0600, log warning
- Same for `data.db` and `config.toml`
- On Windows: stub (`golang.org/x/sys/windows` ACL set)
- Test: files start at 0600 or are corrected on first access
