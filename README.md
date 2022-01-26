## M45-ChatWire
[![License: MPL 2.0](https:/* img.shields.io/badge/License-MPL_2.0-brightgreen.svg)](https:/* opensource.org/licenses/MPL-2.0)
<br>
[![Go](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml/badge.svg)](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/go.yml)
[![ReportCard](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml/badge.svg)](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/report.yml)
[![CodeQL](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml/badge.svg)](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/codeql-analysis.yml)
[![BinaryBuild](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml/badge.svg)](https:/* github.com/Distortions81/M45-ChatWire/actions/workflows/build-linux64.yml)
### Requirements:
Linux<br>
Golang 1.17+<br>
Factorio Headless 1.1+<br>
ImageMagick *(optional)*<br>
Zip *(optional)*<br>
<br>
Launching will create a default auto-config to get you started.<br>
Needs permisisons to create files and directories in its own directory, and **up one directory**.<br>
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

