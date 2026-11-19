<img src="img-source/logo-readme.png" alt="ChatWire Logo" width="400" height="137">

# ChatWire

Factorio server manager and Discord bridge.

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go CI](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![Go Report](https://goreportcard.com/badge/github.com/Distortions81/M45-ChatWire)](https://goreportcard.com/report/github.com/Distortions81/M45-ChatWire)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Distortions81/M45-ChatWire)](https://github.com/Distortions81/M45-ChatWire)
[![Vulncheck](https://github.com/Distortions81/M45-ChatWire/actions/workflows/vulncheck.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/vulncheck.yml)

[Command Overview](https://m45sci.xyz/help-discord-staff.html)

## What It Does

ChatWire runs alongside a Linux Factorio dedicated server and handles:

- Discord slash commands and chat bridging
- Factorio process launch, stop, restart, update, and reboot workflows
- Config generation and reloads
- Player database and ban list maintenance
- Mod and Factorio update checks
- Multi-instance deployments with shared parent-level files and directories

Recent lifecycle work centralized start, stop, restart, map change, map reset, update, and ChatWire reboot handling behind a single controller with progress-aware timeouts and stronger process health detection.

## Quickbar preset saving

SoftMod skips unchanged Store requests and keeps its one-second UI cooldown. ChatWire combines all Store and Rename requests into one write to `quickbar-presets.json` every five seconds when changes are queued. The latest toolbar and label per player/slot win; different players and slots are saved together. There is no extra change comparison in ChatWire. Each flush locks and reloads the latest shared file before applying only its queued slots, preserving other server instances' changes. Flushes are serialized, and an idle interval does not write. Loading presets and checking access read the committed data immediately. SoftMod shows “Saving…” for queued requests and only confirms saving after persistence; failures remain retryable. Orderly shutdown flushes pending changes early.

## Activity visibility

Enable **Hide Activity** in the per-server web settings, or set the local Discord setting `hide-activity` to `true`, to hide raw `[ACT]` notices and activity-summary copies in the map Discord channel. The JSON option is `Options.HideActivity` in `cw-local-config.json`. After saving web or JSON edits, use `reload-config` to apply them. It defaults to `false` (activity remains visible). Disabling it shows activity again. Activity logs, suspicion scoring, moderation report delivery and in-game witness warnings remain active. This setting is independent of the global **Suspicious Warning Mute**.

Moderators can run `/mute-warning-pings` in the moderation channel with a Factorio player name and an optional positive number of minutes (default 30). It suppresses moderator-role pings for that player across servers until expiry, while keeping reports, activity logs, and in-game warnings visible. Mutes survive ChatWire restarts. Register slash commands again after upgrading.

## Early-session activity review

For New and Member players, ChatWire counts structures manually mined and placed during the first 15 minutes after a recorded map join. If a player mines at least eight and mines at least three times as many as they place, a mining event can get a temporary 36-point bonus toward the existing warning threshold. The effective score must exceed 40 for New or 1,300 for Member. Regular and Veteran players are outside this automatic scoring rule. The bonus can qualify one warning per session and is never stored in the player database. An early-removal alert goes to the moderation report channel with the elapsed time and counts. All score crossings are reported. The first early-removal alert, or the first qualifying burst at that site with at least 120 net deconstruction marks or 48 structures mined at a 3:1 mining-to-placement ratio, mentions the configured moderator role. A qualifying in-game witness warning also pings moderators if the incident has not already done so. An earlier quiet report leaves that ping available. Follow-ups and map-channel copies carry no role mention. A different player, another site, or 30 quiet minutes can start a new incident.

Players get at most one in-game warning per player and site. It needs a normal-threshold alert, a GPS location and a burst with either 40 more deconstruction marks than cancellations or at least 16 manually mined structures with at least three times as many mined as placed. Those counts apply in the first 15 minutes after an observed join. Later, or when join time is unknown, the minimums are 160 net marks or 64 structures mined at the same ratio. The message gives the location and counts so witnesses can decide whether to use `/report` or, when eligible, `/banish`.

## Requirements

- Linux
- Go `1.27.4` or newer to build from source
- A Discord application with bot token, application ID, guild ID, and channel access
- Factorio account credentials for update and server management features

On first launch ChatWire generates default config files and supporting directories. It must be able to create files in its own directory and in the parent directory.

Common generated files and directories:

- `cw-local-config.json`
- `../cw-global-config.json`
- `cw.lock`
- `../playerdb.cwdb` (versioned binary player database with fast Zstandard compression)
- `../quickbar-presets.json` (personal quickbar presets shared by all instances, created on first store)
- `../playerdb.cwdb.lock` (persistent advisory-lock metadata; safe to leave in place)
- `../map-gen-json/`
- `./map-gen-json/` for per-server `/map-exchange` custom settings and cached map-generator fallbacks
- `./log/`
- `./audit-log/`
- `../www/public_html/archive/`

## Build

```bash
git clone https://github.com/Distortions81/M45-ChatWire.git
cd M45-ChatWire
go build
```

This produces the `ChatWire` binary in the current directory.

## Quick Start

1. Build the binary with `go build`.
2. Run `./ChatWire` once to generate `cw-local-config.json` and `../cw-global-config.json`.
3. Edit those config files with your Discord credentials, Factorio credentials, and server paths.
4. Register slash commands with `./ChatWire -regCommands`.
5. Start the service with `./ChatWire`.

If you need to regenerate config files, delete `cw-local-config.json` and `../cw-global-config.json`, then start ChatWire again.

The delayed M45 essentials panel uses one shared player timestamp (`th` in JSON)
to limit join reminders to once per 24 hours for New/Member ranks and once per
30 days for higher ranks. There is no lifetime count limit. Cooldown claims reload
and lock the shared database before writing, and stale local player saves preserve
the newer timestamp. The timestamp is reserved before replying to SoftMod, so a
lost reply or disconnect can skip that reminder rather than produce duplicates.

Binary player databases now write version 2, adding the reminder timestamp, and
still read version 1 and legacy JSON. Update all ChatWire instances sharing a
database together: older binaries cannot read version 2 and older JSON writers
would discard the new field. Rollback requires exporting to the old schema first.

The player database defaults to the compressed binary format. Set `Paths.DataFiles.DBFormat` to `"json"` to retain readable JSON output. Reads auto-detect both formats, so an existing JSON database is migrated on its next write when `DBFormat` is `"binary"`; the configured `DBFile` name does not have to change.

## Default Layout

The project expects a parent "base" directory layout so multiple Factorio servers can share common files.

Example:

```text
~/factServers/
  cw-a/
    ChatWire
  factorio/
    bin/x64/factorio
```

That layout is only a default convention. Adjust paths in the generated config files for your environment.

## Runtime Flags

| Flag | Description |
|------|-------------|
| `-cleanBans` | Clean and minimize the player database, including bans, then exit. |
| `-cleanDB` | Clean and minimize the player database, then exit. |
| `-deregCommands` | Deregister Discord commands and quit. |
| `-localTest` | Disable public/auth mode for testing. |
| `-noAutoLaunch` | Disable Factorio auto-launch. |
| `-noDiscord` | Disable Discord integration. |
| `-proxy` | HTTP caching proxy URL. Format: `http://example.domain/path` |
| `-regCommands` | Register Discord commands. |
| `-webControlConfig` | Optional path to the private web-control instance configuration; disabled when omitted. |
| `-runtimeSelfTest` | Run runtime self-test cases, then exit. Example: `start,restart,stop,change-map,update-check,chatwire-reboot` |
| `-runtimeSelfTestTimeout` | Per-step timeout for runtime self-tests. Default: `5m` |

## Discord Bot Setup

1. Visit <https://discord.com/developers/applications> and create a new application.
2. Under **Bot**, add a bot user.
3. Enable the privileged intents ChatWire relies on: Presence, Server Members, and Message Content.
4. Grant the bot the permissions it needs in your server, including viewing channels, managing channels and roles, sending messages, embedding links, attaching files, reading message history, deleting leaked messages, and using application commands.
5. Copy the bot token and the application ID.
6. In Discord, enable Developer Mode and copy your guild ID.
7. Copy a channel ID, or leave `ChatChannel` blank in `cw-local-config.json` and let ChatWire create one on first run.

After adding or changing slash commands, run `./ChatWire -regCommands` again.

## Operations

### Signals

- `SIGUSR1` queues a ChatWire reboot once no players are online.
- `SIGUSR2` reloads `cw-local-config.json` and `../cw-global-config.json`.

### Automatic Reloads

ChatWire watches the player database and ban list for file changes. Config reloads are triggered through `/chatwire action reload-config` or `SIGUSR2`.

For multi-instance environments, see `example-files/reload-config.sh`.

### SoftMod Recovery After Factorio Updates

When `Options.SoftModOptions.InjectSoftMod` is enabled, the first successful
Factorio launch after an installation or update waits 30 seconds after startup
readiness for the SoftMod handshake. If SoftMod is still missing, ChatWire queues
one additional Factorio restart to reinject it through the normal launch path.
This recovery does not repeat if SoftMod remains unavailable, and it defers to
an already queued stop, restart, or map change.

### Firewall IP Bans

The moderator slash command `/ip-ban` manages UFW deny rules for public IPv4 addresses.

- `action=ban` inserts a deny rule at position 1, ahead of existing user allow rules
- `action=unban` removes a deny rule
- `action=list` shows current IPv4 deny entries

UFW's default `before.rules` accepts established/related traffic before user
rules, so an existing session (including tracked UDP traffic) can survive a ban.
Inserting at position 1 fixes user-rule ordering; it does not clear conntrack.
Existing appended bans may need to be removed and re-added to move them ahead of
allow rules; UFW can skip inserting an identical rule that already exists.

This requires:

- UFW installed at `/usr/sbin/ufw`
- The ChatWire service user allowed to run `sudo ufw` without a password

Example sudoers entry:

```sudoers
chatwire ALL=(root) NOPASSWD: /usr/sbin/ufw
Defaults!/usr/sbin/ufw !requiretty
```

Replace `chatwire` with the actual service user on your host.

## Example Files

The [`example-files/`](example-files) directory includes:

- sample `systemd` units
- helper scripts for start, stop, reboot, cleanup, and config reload
- sample local and global config files

Copy them and adjust paths for your deployment.

## Development

The machine interface to M45 SoftMod is documented in
[`docs/softmod-protocol.md`](docs/softmod-protocol.md).

Moderator web controls are available through a separate `chatwire-web` service
and opt-in private instance sockets. Run `/web` in Discord to request a private,
single-use dashboard login link after configuring the service. See the
[setup guide](docs/web-control-setup.md), [API contract](docs/web-control-openapi.json),
and [architecture design](docs/web-control-api.md).

Run the standard checks before committing:

```bash
go fmt ./...
go vet ./...
go test ./...
```

Some integration tests connect to Discord. Set `CW_TEST_TOKEN`, `CW_TEST_GUILD`, and `CW_TEST_APP` to enable them.

The [native Factorio suite](docs/native-factorio-tests.md) validates SoftMod
handshakes, quickbar persistence, map previews, fixed-port startup, graceful
save-and-quit, and depleted-miner moderation with an isolated headless engine.
Set `CW_FACTORIO_BINARY` to opt in; it requires no Discord credentials.

## Notes

- `-noDiscord` allows Factorio lifecycle management without connecting the bot.
- `-noAutoLaunch` prevents ChatWire from starting Factorio automatically on startup.
- `-runtimeSelfTest` is intended for exercising lifecycle actions in a live-like environment and exits when the requested cases finish.

New maps initialized by SoftMod send their actual map exchange string on the first ChatWire handshake. ChatWire posts a **New map** embed with a short **Preview map** link in the server's chat channel, hiding the long exchange data. Oversized links remain attached intact as `map-preview.txt`. Normal restarts and SoftMod upgrades do not repost it. This requires SoftMod build 673 or newer and the matching ChatWire protocol handler.

`/map-preview` is also available on demand in Factorio and Discord. Factorio replies privately to the requesting player, or to RCON when called from the console. Discord invokes this fixed read-only command over RCON and returns an ephemeral link (or a text attachment for exceptionally long links). Register the new Discord slash command during the planned rollout; it requires the updated SoftMod to be loaded. Manual requests do not consume or repeat the automatic new-map announcement.

### Popular maps

The primary server automatically finds or creates a `popular-maps` Discord text channel. Set `Discord.PopularMapsChannel` in the shared global configuration to use a specific existing channel instead. The primary server checks the post every 15 minutes and edits it only when the displayed content changes. It lists the top five maps over the rolling last 24 hours, 7 days, and 30 days. Rankings use combined player hours: each minute, each running, unpaused server records its current player count. Data is stored under `map-popularity` in the shared servers root. Rolling tracking begins when this version is run. Older daily totals cannot be assigned accurately to rolling windows; separately reconstructed historical records may be estimates from player join and leave logs. Update all server instances to record their activity in the rolling windows. The bot needs permission to view the channel, read message history, and send and edit the post, and Manage Channels to create it.

The same post links to [m45sci.xyz](https://m45sci.xyz) and shows community rank counts from the shared Factorio player database. It also lists the three most recently reset maps, newest first. Each map appears once, using its latest completed reset in the ChatWire lifecycle logs.
