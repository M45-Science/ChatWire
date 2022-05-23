## M45-ChatWire
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
<br>
[![Go](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![ReportCard](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml)
[![CodeQL](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml)
[![BinaryBuild](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml)
### V2 Changes:
Disord slash commands, with autocomplete.<br>
Better handling of LUA errors.<br>
Suppress server list slow-transfer messages.<br>
Updated to very latest version of DiscordGo.<br>
Much faster boot/shutdown, with clean close.<br>
Map reset messages, and other messages will not get lost on shutdown.<br>
Utilize BotReady event in the new Discord API (faster).<br>
Major config file reorganization.<br>
Discord commands renamed, much clearer.<br>
Always auto-configure any missing settings.<br>
Discord channel sorting improved.<br>
SoftMod presence/version detection.<br>
Handle absent soft-mod, chat/online commands... etc.<br>
Improved player-online command (caching, event based)<br>
Fixed multiple issues with ban messages.<br>
Cleaned up code for waiting for factorio to close (faster).<br>
HideResearch setting.<br>
Experimental detection/warning of possible griefing.<br>
Automatically ban players from global ban list if they are already playing.<br>
Automatically put steam URL in channel topic.<br>
No longer require DMs to be on to register. (Ephemeral message)<br>
Registration automatically supplies a steam link to connect with.<br>
Removed a number of unused or obsolete functions and files.<br>
Many messages rewritten to be clearer.<br>
Vast majority of messages moved to ephemeral/private messages.<br>
Many timers relaxed to reduce load.<br>
Better handling of operations that need to detect if Factorio is running or fully booted.<br>
Map archives now show as attachments in chat.<br>
Map previews now directly emebed in response.<br>
Automatically create map preview directory.<br>
Changed map seed generation, as well as new map names.<br>
Fixed many typos.<br>
Many other small adjustments.<br>
Attempt to protect players from publicly posting registration codes.<br>
(invalidates code if typed in chat/discord)<br>
Automatically warn/inform players about using /register on members-only servers.<br>
Factorio /register command is now more forgiving about code formatting.<br>

### Requirements:
Linux<br>
Golang 1.17+<br>
Factorio Headless 1.1+<br>
ImageMagick *(optional)*<br>
Zip *(optional)*<br>
<br>
Launching will create a default auto-config to get you started.<br>
Needs permissions to create files and directories in its own directory, and **up one directory**.<br>
<br>
`Discord token, guild-id and channel-id are required, as well as Factorio username and token.`<br>
<br>
### Default path layout:<br>
A 'base' folder the chatwire folder resides in.<br>
`~/factServers/`<br>
<br>
For ChatWire:<br>
`~/factServers/cw-a/ChatWire-binary-here`<br>
<br>
Factorio:<br>
`~/factServers/fact-a/`<br>
<br>
Binary:<br>
`~/factServers/fact-a/bin/x64/Factorio`<br>
<br>
In the cw-global-config.json:<br>
```
"PathData": {
		"FactorioServersRoot": "/home/user/factServers/",
		"FactorioHomePrefix": "fact-",
		"ChatWireHomePrefix": "cw-",
		"FactorioBinary": "bin/x64/factorio",
	},
 ```
**This is setup to have many servers running, and so some files and directories are setup to be common.**<br>

