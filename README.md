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
Launch params:
```text
Usage of ChatWire:
  -cleanBans
        Clean/minimize player database, along with bans and exit.
  -cleanDB
        Clean/minimize player database and exit.
  -deregCommands
        Deregister discord commands and quit.
  -localTest
        Turn off public/auth mode for testing
  -noAutoLaunch
        Turn off auto-launch
  -regCommands
        Register discord commands
```
        
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

### Web Control Panel

Moderators can generate a temporary token with the `/web-panel` command. The control panel exposes
information from the `/info` command such as versions, uptime, next map reset and player statistics.
It provides buttons for common moderator actions like starting or stopping Factorio, synchronising
mods or updating the game. A map section lists the most recent autosaves and lets you load one with
a single click or supply your own file name. Another form allows running arbitrary RCON commands.
The page is rendered in a dark theme styled similarly to the public staff documentation. Open the
link provided by `/web-panel` in a web browser and supply the token as a query parameter. When
ChatWire is started with `-localTest` the URL will use `127.0.0.1` so it can be accessed locally.
The panel now mirrors information from `/info verbose` including the last save name and UPS
statistics. Additional forms allow adjusting play hours, scheduling map resets and changing a
player's level directly from the browser.
