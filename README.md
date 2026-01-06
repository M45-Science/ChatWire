<img src="img-source/logo-readme.png" alt="ChatWire Logo" width="400" height="137">

# ChatWire

Factorio Server Manager & Discord Bridge

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
[![Go CI](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![Go Report](https://goreportcard.com/badge/github.com/Distortions81/M45-ChatWire)](https://goreportcard.com/report/github.com/Distortions81/M45-ChatWire)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Distortions81/M45-ChatWire)](https://github.com/Distortions81/M45-ChatWire)
[![Vulncheck](https://github.com/Distortions81/M45-ChatWire/actions/workflows/vulncheck.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/vulncheck.yml)

[Command Overview](https://m45sci.xyz/help-discord-staff.html)

### Requirements:
Linux<br>
Golang 1.24.11+<br>
<br>
ChatWire is approximately 15.9k lines of Go code across 120 files.
Launching will create a default auto-config to get you started.<br>
Needs permissions to create files and directories in its own directory, and **up one directory**.<br>
<br>
Some dirs and files that can be auto-created:<br>
cw-local-config.json, ../cw-global-config.json<br>
cw.lock, ../playerdb.json<br>
../map-gen-json/, ./log/, ./audit-log/, ../www/public_html/archive/<br>
`Discord token, appid,  guild-id and channel-id are required, as well as Factorio username and token.`<br>
<br>
### Building ChatWire:<br>
```bash
git clone https://github.com/Distortions81/M45-ChatWire.git
cd M45-ChatWire
go build
```
This produces the `ChatWire` binary in the current directory.<br>
Launching the binary for the first time will create `cw-local-config.json` and `../cw-global-config.json`.<br>
Edit these files to provide your Discord credentials, Factorio token and server paths.<br>
After configuring run `./ChatWire -regCommands` to register slash commands.<br>
<br>
### Default path layout:<br>
A 'base' folder the chatwire folder resides in.<br>
`~/factServers/`<br>
<br>
For ChatWire:<br>
`./cw-a/ChatWire-binary-here`<br>
<br>
Factorio:<br>
`./cw-a/factorio/`<br>
<br>
Binary:<br>
`./cw-a/factorio/bin/x64/Factorio`<br>
**This is setup to have many servers running, and so some files and directories are setup to be common.**<br>
<br>

### Example files
The `example-files` directory contains sample systemd units, configuration files
and helper scripts for managing multiple servers. Copy them and adjust the paths
as needed for your environment.<br>

### Launch parameters:

| Flag | Description |
|------|-------------|
| `-cleanBans` | Clean and minimise the player database, remove bans then exit. |
| `-cleanDB` | Clean and minimise the player database then exit. |
| `-deregCommands` | Deregister Discord commands and quit. |
| `-localTest` | Disable public/auth mode for testing. |
| `-noAutoLaunch` | Disable auto-launch. |
| `-noDiscord` | Disable Discord integration. |
| `-proxy` | HTTP caching proxy URL. Format: `proxy/http://example.domain/path`. |
| `-regCommands` | Register Discord commands. |
<br>

### Setting up your Discord bot
1. Visit <https://discord.com/developers/applications> and create a **New Application**.
2. Under **Bot** click **Add Bot**. Enable the *Presence*, *Server Members* and *Message Content* intents and grant these permissions: view channels, manage channels, manage roles, send messages, embed links, attach files, mention all roles, manage messages (delete message if register code leaked), read message history and use application commands.
3. Copy the **Token** from the bot page and note the **Application ID** from *OAuth2 > General*.
4. In Discord enable *Developer Mode* and right click your server to **Copy ID** for the guild ID.
5. Copy a channel ID or leave `ChatChannel` blank in `cw-local-config.json` and ChatWire will create one on first run.

### Development and Testing

Run `go fmt` to format the code and `go vet` for linting before committing. Tests can be executed with:
```bash
go fmt ./...
go vet ./...
go test ./...
```
These are the same checks executed by the CI pipeline.
Some integration tests connect to Discord. Set CW_TEST_TOKEN, CW_TEST_GUILD and CW_TEST_APP to enable them.

### Regenerating configuration

If you need to reset the configuration files, delete `cw-local-config.json` and `../cw-global-config.json` and start ChatWire again. Fresh copies will be generated automatically. You can also reload the configs at runtime using the `ReloadConfig` moderator command.

### Running the bot locally

Build the binary and register the Discord slash commands:
```bash
go build
./ChatWire -regCommands
./ChatWire
```
Ensure the generated configuration files contain your Discord token, application ID, guild ID and channel ID, along with Factorio credentials.

### Signals

Send `SIGUSR1` to queue a reboot once there are no players online.
Send `SIGUSR2` to reload `cw-local-config.json` and `../cw-global-config.json`.

### Automatic reloads
ChatWire monitors the player database and ban list for modifications.
Configuration files are reloaded via `/chatwire action reload-config` or `SIGUSR2`.
To reload configs for all running instances at once, see `example-files/reload-config.sh`.
