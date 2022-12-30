## M45-ChatWire²
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https://opensource.org/licenses/MPL-2.0)
<br>
[![Go](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![ReportCard](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml)
[![CodeQL](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml)
[![BinaryBuild](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml/badge.svg)](https://github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml)

### Requirements:
Linux<br>
Golang 1.19+<br>
<br>
ChatWire is currently approximately 8500 lines of go code.
Launching will create a default auto-config to get you started.<br>
Needs permissions to create files and directories in its own directory, and **up one directory**.<br>
<br>
Some dirs and files that can be auto-created:<br>
cw-local-config.json, ../cw-global-config.json<br>
cw.lock, ../playerdb.json<br>
../map-gen-json/, ./logs/, ../update-cache/, ../public_html/archive/<br>
`Discord token, appid,  guild-id and channel-id are required, as well as Factorio username and token.`<br>
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
Launch params:<br>
Usage of ./ChatWire:<br>
  -deregCommands<br>
        Deregister discord commands<br>
  -localTest<br>
        Turn off public/auth mode for testing<br>
  -noAutoLaunch<br>
        Turn off Factorio auto-launch<br>
  -regCommands<br>
        Register discord commands<br>
        
