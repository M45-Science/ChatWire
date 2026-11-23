# Native Factorio integration tests

These opt-in tests run the real headless engine and a temporary copy of SoftMod.
They require Factorio 2.1.24 or newer, including its bundled quality expansion.
The suite has been verified with 2.1.24 (87808) and 2.1.25 (88022).

From the ChatWire repository:

```bash
CW_FACTORIO_BINARY=/path/to/factorio/bin/x64/factorio \
CW_SOFTMOD_SOURCE=/path/to/SoftMod \
go test -v ./support -run '^TestNativeFactorio' -count=1
```

`CW_SOFTMOD_SOURCE` defaults to the sibling `../SoftMod` repository. Supply the
absolute path when checking out the repositories elsewhere. Without
`CW_FACTORIO_BINARY`, ordinary `go test ./...` skips the native tests. Tests do
not download or update Factorio; obtain a headless installation from
[Factorio's downloads](https://factorio.com/download).

Each test stages Lua files, runtime images and locales in a private temporary
scenario. Config, mods, player data, saves and quickbar JSON live in that test's
temporary directory. No Discord session, account credentials or running server
is needed. Fixed game and RCON ports are selected from available ports before
launch; the lifecycle case uses the same `--port` and `--rcon-port` flags and
port-offset calculation as ChatWire. Test processes have deadlines and are
reaped on failure. Existing server files and settings are not modified.

## Coverage

- **Join essentials:** no immediate Help/Info popup; idle joins do not start the
  timer; first activity starts the eight-second delay; open interfaces defer the
  introduction; stale replies are ignored; one right-side panel opens without
  taking focus; Banish, Discord, Patreon and To-do actions work; Map Vote has its own page and
  Back navigation; supporter credits are sorted and deduplicated; the Help/Info
  button toggles the new panel; disconnects clear state.

- **Quickbar persistence:** real GUI Store event, Factorio's exchange encoding,
  ChatWire's permission check and queued write, pending acknowledgement, final
  acknowledgement after flushing, a fresh file-backed Load request, and real
  GUI loading in a second engine process. The toolbar includes sparse slots,
  an uncommon-quality filter, and slot 100.
- **Handshake and map preview:** actual `/chatwire` requests over stdin, response
  IDs and SoftMod version, one automatic new-map preview, repeated handshakes
  without duplicates, and `/map-preview` through the real RCON client. The URL
  must match ChatWire's conversion of the engine's actual exchange string.
- **Lifecycle:** real startup output passes ChatWire's readiness matcher;
  `/quit` performs a graceful save and exits successfully in both blocking and
  nonblocking saving modes; `--test-save` loads
  the quit-save and verifies a persistent storage marker and the consumed
  new-map announcement flag. A second ordinary `--start-server` launch loads
  that save and handshakes without announcing a new map again.
- **Moderation:** real entity deconstruction and cancellation events for 64
  empty drills, a drill with ore but no power, and 64 ordinary structures. The
  2.1.25 uses the actual depleted-miner planner filter over a mixed selection;
  2.1.24 uses equivalent individual marks because the filter did not exist. The
  resulting protocol messages pass through ChatWire's activity handler. Empty
  drills stay auditable without suspicion or alerts; their cancellations must
  not reduce the ordinary structure incident. The live drill remains scored.

- **Ban synchronization:** two live headless maps with one temporary player
  database. A stale promotion and delayed sync replies preserve committed
  moderation. Native RCON and ChatWire moderator unbans reach both native ban
  lists. Imported community bans retain their reasons and Regular exemptions.

The Lua assertions use `--test-scenario`, `--test-save`, `--test-arg` and
`script.finish_test(true)`. A finite `--test-ticks` limit prevents runaway tests,
but Factorio returns success when that limit is reached too. The Go runner
therefore also requires the explicit `NATIVE-FACTORIO-PASS` assertion marker.

These tests exercise protocol, persistence and headless engine behavior. They
do not connect a multiplayer client or deliver Discord messages. The normal Go
suite separately tests lifecycle-controller transitions and Discord payloads.
