# UI Mockup — Alkalyne in use

## First run

```
 alkalyne

                                 #lobby
                                 ──────────────────────────
```

---

## Lobby active

```
 alkalyne

                                 #lobby
                                 ──────────────────────────
                                 bob: morning everyone
                                                     09:12
                                 alice: hey bob ready for
                                 the demo today?
                                                     09:13
                                 charlie: @bob yes lets
                                 do it
                                                     09:14
                                 ──────────────────────────
                                 > _
```

---

## Sidebar open (Tab)

```
 alkalyne
          #lobby         3      │
          alice          o      │
          bob            ·      │
          charlie        ⊗  2   │
          ───              ───  │
          home-pi        ◎      │
                                 #lobby
                                 ──────────────────────────
                                 bob: morning everyone
                                                     09:12
                                 alice: hey bob ready for
                                 the demo today?
                                                     09:13
                                 charlie: @bob yes lets
                                 do it
                                                     09:14
                                 ──────────────────────────
                                 > _
```

---

## DM with alice

```
 alkalyne

                                 alice
                                 ──────────────────────────
                                 do you have the key?
                                                    09:15 >>
                                 yeah got it
                                                    09:16
                                 ──────────────────────────
                                 > sending the patch now
```

---

## DM — after sending, awaiting delivery

```
 alkalyne

                                 bob
                                 ──────────────────────────
                                 can you review the patch?
                                                    09:17
                                 ──────────────────────────
                                 > on it
```

---

## Message failed to send

```
 alkalyne

                                 bob
                                 ──────────────────────────
                                 can you review the patch?
                                                    09:17
                                 on it
                                                    09:18 !
                                 ──────────────────────────
                                 > _
```

---

## Scrolled up

```
 alkalyne

                                 alice
                                 ··
                                 ──────────────────────────
                                 do you have the key?
                                                    09:15 >>
                                 yeah got it
                                                    09:16
                                 ──────────────────────────
                                 > ready when you are
```

---

## Invite overlay (Ctrl+L)

```
 alkalyne

                   alkalyne://12D3KooWHKbBcRi

                              [y] copy
```

---

## Command mode (:)

```
 alkalyne

                                 alice
                                 ──────────────────────────
                                 do you have the key?
                                                    09:15 >>
                                 yeah got it
                                                    09:16
                                 ──────────────────────────
                                 :add 12D3KooWHKbBcRi
```

---

## Help overlay (?)

```
 alkalyne

 Tab        cycle focus         Ctrl+L    copy invite link
 Enter     open / send          Ctrl+E    send message
 ↑ ↓       navigate / scroll    Ctrl+D    delete conversation
 PgUp/PgDn scroll faster        Ctrl+B    toggle sidebar
 /         search                Ctrl+C    quit
 :         command mode          Esc       close overlay
 ?         this help

 press any key to close
```

---

## Relays offline, one contact with pending delivery

```
 alkalyne
          #lobby         1      │
          bob            ⊗      │
          charlie        ·      │
          ───              ───  │
          home-pi        ⊘      │
          backup-pi      ⊘      │
                                 bob
                                 ──────────────────────────
                                 sent you a patch
                                                    10:02
                                 will review it tonight
                                                    10:03
                                 ──────────────────────────
                                 > _
```

---

## Glyph reference

```
Glyph  Meaning        Color      Width
──────────────────────────────────────────
o      online         green      half
·      offline        dim        half
⊗      relay pending  140 purple half
◎      relay online   140 purple half
⊘      relay offline  dim        half
>      sent           dim        half
>>     delivered      green      half (2 chars)
!      failed         red        half
··     more above     dim        half (2 chars)
```
