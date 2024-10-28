package cfg

import (
	"bytes"
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

var ServerPrefix = ""
var Local local
var Global global

/* GLOBAL CONFIG */
type global struct {
	GroupName     string
	PrimaryServer string
	Discord       discord
	Factorio      factData

	Paths   dataPaths
	Options globalOptions
}

type discord struct {
	Token       string
	Guild       string
	Application string

	ReportChannel   string
	AnnounceChannel string
	Roles           roles
	SusPingRole     string
}

type roles struct {
	Admin     string
	Moderator string
	Regular   string
	Veteran   string
	Member    string
	New       string

	Patreon   string
	Supporter string
	Nitro     string

	RoleCache roleCache
}

type roleCache struct {
	Admin     string
	Moderator string
	Regular   string
	Veteran   string
	Member    string
	New       string

	Patreon   string
	Supporter string
	Nitro     string
}

type factData struct {
	Username string
	Token    string
}

type dataPaths struct {
	ChatWirePrefix string
	Folders        folderPaths
	Binaries       binaryPaths
	URLs           urlPaths
	DataFiles      dataFiles
}

type folderPaths struct {
	FactorioDir   string
	ServersRoot   string
	Saves         string
	Mods          string
	MapGenerators string
	MapArchives   string
	ModPack       string
	FTP           string
}

type binaryPaths struct {
	FactBinary   string
	UpdaterShell string
	Shell        string
}

type urlPaths struct {
	Domain      string
	PathPrefix  string
	LogPath     string
	LogsPathWeb string
	ArchivePath string
	ModPackPath string
}

type dataFiles struct {
	DBFile string
	Bans   string
}

type globalOptions struct {
	Description        string
	ResetPingRole      string
	UseAuthserver      bool
	AutosaveMax        int
	RconOffset         int
	ShutupSusWarn      bool
	DisableSpamProtect bool
}

func WriteGCfg() bool {
	tempPath := constants.CWGlobalConfig + "." + Local.Callsign + ".tmp"
	finalPath := constants.CWGlobalConfig

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(Global); err != nil {
		cwlog.DoLogCW("WriteGCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("WriteGCfg: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteGCfg: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename Gcfg file.")
		return false
	}

	return true
}

func setGlobalDefaults() {
	/* Automatic global defaults */
	if Global.Paths.DataFiles.DBFile == "" {
		Global.Paths.DataFiles.DBFile = "playerdb.json"
		_, err := os.Create(Global.Paths.DataFiles.DBFile)
		if err != nil {
			cwlog.DoLogCW("setGlobalDefaults: Could not create " + Global.Paths.DataFiles.DBFile)
		}
	}
	if Global.Paths.Folders.MapGenerators == "" {
		Global.Paths.Folders.MapGenerators = "map-gen-json"

		err := os.MkdirAll(Global.Paths.Folders.ServersRoot+"/"+Global.Paths.Folders.MapGenerators, os.ModePerm)
		if err != nil {
			cwlog.DoLogCW("Could not create map-gen-json directory.")
			//return false
		}
	}
	if Global.Paths.URLs.Domain == "" {
		Global.Paths.URLs.Domain = "localhost"
	}
	if Global.Options.RconOffset == 0 {
		Global.Options.RconOffset = 10000
	}
	if Global.GroupName == "" {
		Global.GroupName = glob.RandomBase64String(3)
		cwlog.DoLogCW("No group name specified. Random one generated.")
	}

	if Global.Paths.Folders.ServersRoot == "" {
		ex, err := os.Executable()
		if err == nil {
			exPath := filepath.Dir(ex)
			p := filepath.Clean(filepath.Join(exPath, ".."))
			Global.Paths.Folders.ServersRoot = p + "/"
		}
	}

	if Global.Paths.ChatWirePrefix == "" {
		Global.Paths.ChatWirePrefix = "cw-"
	}
	if Global.Paths.Folders.Saves == "" {
		Global.Paths.Folders.Saves = "saves"
	}
	if Global.Paths.Folders.Mods == "" {
		Global.Paths.Folders.Mods = "mods"
	}
	if Global.Paths.Folders.MapArchives == "" {
		Global.Paths.Folders.MapArchives = Global.Paths.Folders.ServersRoot + "www/public_html/archive/"
	}
	if Global.Paths.Folders.ModPack == "" {
		Global.Paths.Folders.ModPack = Global.Paths.Folders.ServersRoot + "www/public_html/modpack/"
	}
	if Global.Paths.Folders.FTP == "" {
		Global.Paths.Folders.FTP = "/home/upload/"
	}
	if Global.Paths.Folders.FactorioDir == "" {
		Global.Paths.Folders.FactorioDir = "factorio"
	}
	if Global.Paths.URLs.PathPrefix == "" {
		currentUser, err := user.Current()
		if err == nil {
			Global.Paths.URLs.PathPrefix = "/u/" + currentUser.Name
		} else {
			Global.Paths.URLs.PathPrefix = "/~username"
		}
	}
	if Global.Paths.URLs.LogPath == "" {
		Global.Paths.URLs.LogPath = "/logs/"
	}
	if Global.Paths.URLs.LogsPathWeb == "" {
		Global.Paths.URLs.LogsPathWeb = "/current-logs/"
	}
	if Global.Paths.URLs.ArchivePath == "" {
		Global.Paths.URLs.ArchivePath = "/archive/"
	}
	if Global.Paths.URLs.ModPackPath == "" {
		Global.Paths.URLs.ModPackPath = "/modpack/"
	}
	if Global.Paths.Binaries.Shell == "" {
		Global.Paths.Binaries.Shell = "/bin/bash"
	}
	if Global.Paths.Binaries.Shell == "" {
		Global.Paths.Binaries.Shell = "/"
	}
	if Global.Paths.Binaries.FactBinary == "" {
		Global.Paths.Binaries.FactBinary = "bin/x64/factorio"
	}
	if Global.Discord.Guild == "" {
		cwlog.DoLogCW("No Discord Guild ID specified. This MUST be set!")
		Global.Discord.Guild = "MY DISCORD GUILD ID"
	}
	if Global.Discord.Application == "" {
		Global.Discord.Application = "MY DISCORD APP ID"
	}
	if Global.Discord.Token == "" {
		cwlog.DoLogCW("No Discord Token specified. This MUST be set!")
		Global.Discord.Token = "MY DISCORD BOT TOKEN"
	}
	if Global.Factorio.Username == "" {
		cwlog.DoLogCW("No Factorio Username specified. This MUST be set!")
		Global.Factorio.Username = "MY FACTORIO USERNAME"
	}
	if Global.Factorio.Token == "" {
		cwlog.DoLogCW("No Factorio Token specified. This MUST be set!")
		Global.Factorio.Token = "MY FACTORIO TOKEN"
	}
	if Global.Options.AutosaveMax == 0 {
		Global.Options.AutosaveMax = 250
	}
	if Global.PrimaryServer == "" {
		Global.PrimaryServer = "a"
	}
}

func ReadGCfg() bool {

	_, err := os.Stat(constants.CWGlobalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateGCfg()
		Global = newcfg

		setGlobalDefaults()
		return true
	} else { /* Otherwise just read in the config */
		file, err := os.ReadFile(constants.CWGlobalConfig)

		if file != nil && err == nil {
			newcfg := CreateGCfg()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				cwlog.DoLogCW("ReadGCfg: Unmarshal failure")
				cwlog.DoLogCW(err.Error())
				return false
			}

			Global = newcfg
			setGlobalDefaults()

			return true
		} else {
			cwlog.DoLogCW("ReadGCfg: ReadFile failure")
			return false
		}
	}
}

func CreateGCfg() global {
	newcfg := global{}
	return newcfg
}
