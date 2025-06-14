<img src="img-source/logo-readme.png" alt="ChatWire Logo" width="400" height="137">

# ChatWire

Factorio Server Manager & Discord Bridge

[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
<br>
[![Go](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![ReportCard](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml)
[![CodeQL](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml)
[![BinaryBuild](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml)

[Command Overview](https://m45sci.xyz/help-discord-staff.html)

### Requirements:
Linux<br>
Golang 1.24+<br>
<br>
ChatWire is approximately 15k lines of go code over 77 files.
Launching will create a default auto-config to get you started.<br>
Needs permissions to create files and directories in its own directory, and **up one directory**.<br>
<br>
Some dirs and files that can be auto-created:<br>
cw-local-config.json, ../cw-global-config.json<br>
cw.lock, ../playerdb.json<br>
../map-gen-json/, ./logs/, ../update-cache/, ../public_html/archive/<br>
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
        
### Launch parameters:

| Flag | Description |
|------|-------------|
| `-cleanBans` | Clean and minimise the player database, remove bans then exit. |
| `-cleanDB` | Clean and minimise the player database then exit. |
| `-deregCommands` | Deregister Discord commands and quit. |
| `-localTest` | Disable public/auth mode for testing. |
| `-noAutoLaunch` | Disable auto-launch. |
| `-noDiscord` | Disable Discord integration. |
| `-panel` | Enable web control panel. |
| `-proxy` | HTTP caching proxy URL. Format: `proxy/http://example.domain/path`. |
| `-regCommands` | Register Discord commands. |
<br>

### Discord bot perms:
The bot needs presence intent, server members intent, message content intent
Perms: view channels, manage channels, Manage roles, send messages, embed links, attach files, mention all roles, manage messages (delete message, if register code leaked), read message history, use application commands.

### Development and Testing

Run `go fmt` to format the code and `go vet` for linting before committing. Tests can be executed with:
```bash
go fmt ./...
go vet ./...
go test ./...
```
These are the same checks executed by the CI pipeline.

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

### Factorio Agent

ChatWire can run Factorio through a small helper daemon. The agent listens on a
Unix socket located next to the agent binary (e.g. `factorio-agent.sock`) and understands a byte protocol:

The `-debug` flag prints verbose logs from the agent. Without it the agent runs silently.

```
0x01 <binary> <args>\n  start Factorio (binary path followed by arguments)
0x02          stop Factorio
0x03          query running status (returns 0x01 or 0x00)
0x04 <line>\n write a command to stdin
0x05          read buffered stdout terminated by NUL
```

Whenever new stdout lines are available the agent sends `0x02 0x06` once per
second. An example systemd unit is available at `misc/factorio-agent.service`.

Enable and start this service so ChatWire can communicate with the agent.

### Example Scripts

Helper scripts live in `example-scripts/`. They automate common tasks like
deploying builds, queuing restarts and generating systemd units. See
`example-scripts/README.md` for details.

### Web Control Panel (Work In Progress)

- Start ChatWire with `-panel` to enable the panel server.
- Generate a temporary token with the `/web-panel` command.
- See `/info` details like versions, uptime, next map reset and player stats.
- Buttons let you start/stop Factorio, sync mods and update the game.
- The map section lists recent saves for one-click loading or custom filenames.
- Send arbitrary RCON commands from the browser.
- Styled in a dark theme similar to the staff docs.
- With `-localTest` the URL uses `127.0.0.1` for local access.
- Mirrors `/info verbose` showing the last save name and UPS stats.
- Extra forms adjust play hours, schedule map resets and set player levels.

