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

## Requirements

- Linux
- Go `1.25.0` or newer to build from source
- A Discord application with bot token, application ID, guild ID, and channel access
- Factorio account credentials for update and server management features

On first launch ChatWire generates default config files and supporting directories. It must be able to create files in its own directory and in the parent directory.

Common generated files and directories:

- `cw-local-config.json`
- `../cw-global-config.json`
- `cw.lock`
- `../playerdb.json`
- `../map-gen-json/`
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

### Firewall IP Bans

The moderator slash command `/ip-ban` manages UFW deny rules for public IPv4 addresses.

- `action=ban` adds a deny rule
- `action=unban` removes a deny rule
- `action=list` shows current IPv4 deny entries

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

Run the standard checks before committing:

```bash
go fmt ./...
go vet ./...
go test ./...
```

Some integration tests connect to Discord. Set `CW_TEST_TOKEN`, `CW_TEST_GUILD`, and `CW_TEST_APP` to enable them.

## Notes

- `-noDiscord` allows Factorio lifecycle management without connecting the bot.
- `-noAutoLaunch` prevents ChatWire from starting Factorio automatically on startup.
- `-runtimeSelfTest` is intended for exercising lifecycle actions in a live-like environment and exits when the requested cases finish.
