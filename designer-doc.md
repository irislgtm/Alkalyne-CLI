# Alkalyne-CLI — UI Designer Document

## Philosophy

Chat is the interface. No split panes, no permanent sidebar, no status bars. The user is either in the conversation or switching between focused views via the keyboard. Every mode occupies the full terminal. Inspiration: terminal messengers with modal navigation — clean, dense, no chrome.

Target 80×24 minimum. Degrades gracefully below 60.

## Modes

The app has four modes. Only one is visible at a time. `Tab` cycles forward, `Shift+Tab` backward.

| Mode | Key | What you see |
|---|---|---|
| **Chat** | default | Messages + input line. Full width. |
| **Sidebar** | `Tab` | Contact/relay list overlays left side (30% width). Chat dims behind. |
| **Command** | `:` | Input line replaced by command prompt. Tab-completion. |
| **Help** | `?` | Keybinding reference overlay. Dismiss with any key. |

## Chat mode (default)

```
 alkalyne
                                 alice
                                 ──────────────────────────
                                 do you have the key?
                                                     09:15
                                 yeah got it
                                                     09:16
                                 ──────────────────────────
                                 > ready when you are
```

- Contact name at top, left-aligned. That's the header.
- Thin divider line below name.
- Messages grouped by sender — name only on first message of a block.
- Timestamps right-aligned on last message of each block.
- `> ` prompt on the input line, always at bottom.
- Status appended to own messages right side: `>` sending, `>>` delivered, `!` failed.
- No bubbles, no background fills, no borders around messages.
- Vertical scroll via `↑` `↓`, page via `PgUp` `PgDn`.
- Scroll indicator: single `..` line at top when scrolled up.

## Sidebar mode (overlay)

```
 alkalyne
          #lobby         3      │
          alice          o      │
          bob            ·      │
          charlie        ⊗  2   │
          ───              ───  │
          home-pi        ◎      │
                                 alice
                                 ──────────────────────────
                                 do you have the key?
                                                     09:15
                                 yeah got it
                                                     09:16
                                 ──────────────────────────
                                 > ready when you are
```

- Sidebar slides in from left, 26 chars wide, with a thin `│` separator.
- Chat is dimmed behind (or just narrows if the terminal supports it).
- `↑` `↓` navigates items. `Enter` opens that conversation.
- `Tab` cycles out back to chat.
- Groups: lobby pinned at top, then contacts sorted by recent, then relays.
- Section divider is `───` between groups.

## Command mode

```
 :add 12D3KooWHKbBcRi
```

- Full-width input at bottom. `Esc` closes. `Tab` completes.
- Available commands: `add`, `invite`, `relay`, `relay-setup`, `register`, `lookup`, `info`

## Help overlay

```
 alkalyne

 Tab        cycle focus       Ctrl+L    copy invite link
 Enter     open / send        Ctrl+E    send message
 ↑ ↓       navigate / scroll  Ctrl+D    delete conversation
 PgUp/PgDn scroll faster      Ctrl+B    toggle sidebar
 /         search              Ctrl+C    quit
 :         command mode        Esc       close overlay
 ?         this help

 Press any key to close.
```

- Centered, no border box. Two columns, left-aligned.
- Dismisses on any keypress. No `[x] close` button.

## Indicators

All unicode. No emoji. No backgrounds.

| Glyph | Meaning | Color |
|---|---|---|---|
| `o` | online | green |
| `·` | offline | dim |
| `⊗` | pending relay delivery | 140 purple |
| `◎` | relay online | 140 purple |
| `⊘` | relay offline | dim |
| `>` | message sent | dim |
| `>>` | delivered | green |
| `!` | send failed | red |
| `··` | more messages above | dim |

- `o` and `·` are ASCII/middle-dot — no emoji risk, consistent half-width
- `⊗` `◎` `⊘` are mathematical symbols — no emoji risk, consistent half-width

## Color usage

Dark theme. 16 ANSI colors. No truecolor dependency.

| Token | ANSI | Where |
|---|---|---|
| `primary` | 39 blue | unread badge count, @mentions, cursor |
| `success` | 76 green | online dot, delivered check |
| `warning` | 214 yellow | pending states |
| `error` | 196 red | failed, disconnected |
| `mailbox` | 140 purple | relay glyphs, mailbox status |
| `textDim` | 245 grey | timestamps, dividers, secondary info |

Apply color to individual glyphs and badge numbers only. Not to text backgrounds, not to full lines.

## Keybindings

```
Tab       cycle mode (chat → sidebar → chat)
Shift+Tab cycle backward
↑↓        navigate sidebar / scroll messages
Enter     open conversation / send message
Ctrl+E    send (alternative)
Ctrl+L    copy invite link
Ctrl+B    toggle sidebar
Ctrl+D    delete current conversation
/         search
:         command mode
?         help
Ctrl+C    quit
Esc       dismiss overlay / clear input
PgUp/PgDn scroll page
```

## States

```
State         What you see
─────────────────────────────────────────
First run     Chat mode, empty lobby.
              No text, nothing.
Lobby empty   `#lobby` at top of message area.
              No "welcome" message.
Disconnected  `!` appears right of the
              contact name in header.
Error         `!` with red color on the
              affected item in sidebar.
Relay offline `⊘` next to relay in sidebar.
```

## Rules

- **No permanent chrome.** No status bar, no help bar, no version string, no connection count.
- **Connection status** is visible only when something is wrong — `!` in header.
- **Peer count** is visible only in sidebar (next to `#lobby`).
- **No instructional text anywhere.** Learn by doing or by `?`.
- **No empty state banners.** Empty is empty.
- **Single mode at a time.** No split views. No background panels.
- **Messages are plain text.** No bubbles, no background fills, no borders.
- **Color on glyphs and numbers only.** Never on message text or backgrounds.

## Implementation Tasks

### Phase 4: Minimal TUI (lobby + chat + sidebar)

**Branch:** `feat/basic-tui`

**Files to create:**
```
internal/tui/app.go               # root model, init, update, view, 4-mode dispatch
internal/tui/chat_model.go         # chat mode: Viewport + TextInput, grouped messages
internal/tui/sidebar_model.go      # sidebar overlay: lobby/contacts/relay list
internal/tui/command_model.go      # command mode (:) with Tab-completion
internal/tui/help_model.go         # help overlay (?) — keybinding reference
internal/tui/styles.go             # Lip Gloss styles for the palette, glyphs, indicators
internal/tui/app_test.go           # unit tests for model transitions
```

**Tasks:**
- 4-mode dispatch in `AppModel.Update`: `ChatMode`, `SidebarMode`, `CommandMode`, `HelpMode`
- `ChatModel`: `Viewport` for message list (auto-scroll to bottom), `TextInput` at last line, `───` divider below header name, timestamps right-aligned, status glyphs (`>`, `>>`, `!`) on own messages
- `SidebarModel`: `bubbles.List` in a 26-char left overlay with `│` separator. 3 groups: lobby (pinned), contacts, relays. Section dividers with `───`.
- `CommandModel`: full-width text input at bottom, `Tab` completion from command list, `Esc` to close
- `HelpModel`: centered keybinding reference, dismiss on any keypress
- Styles: dark theme, glyph-only coloring, no background fills

**Deliverables:**
- `go build ./cmd/alkalyne` starts TUI in chat mode with lobby displayed
- `Tab` cycles between chat ↔ sidebar modes
- `:` opens command input, `Esc` closes
- `?` shows help overlay, any key dismisses
- `↑` `↓` scroll messages, `PgUp` `PgDn` page scroll
- `Ctrl+L` opens invite overlay with PeerID link (copy placeholder)

**Depends on:** Phase 1 (project skeleton — `main.go` must call TUI)

**Renders what:** `UI-MOCKUP.md` — first run, lobby active, sidebar open, help overlay, invite overlay

---

### Phase 5: DM conversations + persistence display

**Branch:** `feat/dm-conversations`

**Files to create/modify:**
```
internal/tui/chat_model.go         # modify: add conversation switching
internal/tui/commands.go           # Bubble Tea Cmd functions for backend calls
```

**Tasks:**
- Conversation switching: selecting a contact in sidebar → switches chat to that DM
- Messages loaded from `internal/db` for the selected conversation
- Sent messages written to DB and published to pubsub
- Lobby is ephemeral (no persistence); DMs persist and scroll
- Group messages by sender block, timestamps on last message of each block

**Deliverables:**
- Click contact in sidebar → see DM history
- Send message → stored locally, published to recipient's topic
- Lobby messages appear but are gone after restart
- DMs survive restart

**Depends on:** Phase 4 (TUI exists) + `internal/db` (messages/conversations queries)

**Renders:** DM view in `UI-MOCKUP.md`

---

### Phase 8: Shareable invite link

**Branch:** `feat/invite-link`

**Files to modify:**
```
internal/tui/app.go               # add Ctrl+L handler, invite overlay state
internal/tui/styles.go             # invite overlay styles
```

**Tasks:**
- `Ctrl+L` → overlay showing `alkalyne://{peer_id}` centered in terminal
- `y` to copy to clipboard (via `golang.org/x/term` or `osascript`/`xclip`/`wl-copy`)
- `Esc` to close overlay
- On receiving side: `:add alkalyne://peerid` or `:add 12D3KooW...` adds contact

**Deliverables:**
- `Ctrl+L` shows invite link overlay
- `y` copies to system clipboard
- `:add <peerid>` adds to contacts DB

**Depends on:** Phase 4 (TUI infrastructure) + Phase 2 (PeerID exists)

**Renders:** Invite overlay in `UI-MOCKUP.md`

---

### Phase 12: TUI relay management

**Branch:** `feat/tui-relays`

**Files to create/modify:**
```
internal/tui/relay_setup_model.go  # guided wizard for configuring a relay
internal/tui/sidebar_model.go      # modify: add relay section, status glyphs
internal/tui/command_model.go      # modify: add relay commands
```

**Tasks:**
- Sidebar: show relay section below contacts. `◎` (online/purple), `⊘` (offline/dim). Queued message count as badge.
- `:relay-setup` → full-screen wizard:
  1. Prompt for relay nickname
  2. Prompt for listen address (default `0.0.0.0:9001`)
  3. Prompt for authorized peers (or `all`)
  4. Output generated config snippet + `--relay-config` CLI flags
- `:relay` → show current relay status
- `:relay-add <name> <peerid> <addr>` → add relay to config
- `:relay-remove <name>` → remove relay from config

**Deliverables:**
- Relays visible in sidebar with online/offline glyphs
- Wizard generates valid config for `alkalyne daemon`
- `:relay-add/:relay-remove` updates config live

**Depends on:** Phase 4 (TUI exists) + Phase 10 (relay daemon exists)

**Renders:** Relay setup wizard in `UI-MOCKUP.md`

---

### Phase 14: Search UI

**Branch:** `feat/search-ui`

**Files to modify:**
```
internal/tui/chat_model.go         # add search mode
internal/tui/command_model.go      # modify: /search integration
```

**Tasks:**
- `/` enters search mode in chat: input becomes search query
- Backend: `SearchMessages(query, limit, offset)` returns matching messages
- Results displayed inline in chat view as a scrollable list
- `Enter` on a result → jumps to that conversation and scrolls to that message
- `Esc` clears search and returns to normal chat

**Deliverables:**
- `/` from any mode opens search
- Type query → results appear (matching messages with context)
- `Enter` navigates to result conversation

**Depends on:** Phase 5 (messages stored in DB)
