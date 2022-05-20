package cfg

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"path/filepath"

	"ChatWire/constants"
	"ChatWire/cwlog"
)

var ServerPrefix = ""
var Local local
var Global global

/* GLOBAL CONFIG */
type global struct {
	GroupName string
	Discord   discord
	Factorio  factData

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
}

type roles struct {
	Moderator string
	Regular   string
	Member    string
	New       string

	Patreon string
	Nitro   string

	RoleCache roleCache
}

type roleCache struct {
	Moderator string
	Regular   string
	Member    string
	New       string

	Patreon string
	Nitro   string
}

type factData struct {
	Username string
	Token    string
	RCONPass string
}

type dataPaths struct {
	FactorioPrefix string
	ChatWirePrefix string
	Folders        folderPaths
	Binaries       binaryPaths
	URLs           urlPaths
	DataFiles      dataFiles
}

type folderPaths struct {
	ServersRoot   string
	Saves         string
	MapGenerators string
	MapPreviews   string
	MapArchives   string
	UpdateCache   string
}

type binaryPaths struct {
	FactBinary      string
	FactUpdater     string
	UpdaterShell    string
	Shell           string
	ImgCmd          string
	RmCmd           string
	ZipCmd          string
	SoftModInserter string
}

type urlPaths struct {
	Domain        string
	LogURL        string
	ArchiveURL    string
	MapPreviewURL string
}

type dataFiles struct {
	DBFile        string
	RecordPlayers string
	Bans          string
}

type globalOptions struct {
	Description     string
	PingString      string
	UseAuthserver   bool
	AutosaveMax     int
	RconOffset      int
	PreviewSettings prevSettings
}

type prevSettings struct {
	Arguments string
	PNGRes    string
	PNGScale  string

	JPGQuality string
	JPGScale   string
}

/* LOCAL CONFIG */
type local struct {
	Callsign string
	Name     string
	Port     int

	Settings settings

	Channel channel
	Options localOptions
}

type settings struct {
	MapGenerator string
	MapPreset    string
	Seed         uint64
	AFKMin       int
	AutosaveMin  int
	AutoPause    bool
}

type channel struct {
	Position    int
	ChatChannel string
}

type localOptions struct {
	ScheduleText string
	PingString   string

	AutoStart     bool
	AutoUpdate    bool
	ExpUpdates    bool
	ReportBans    bool
	HideAutosaves bool
	HideResearch  bool
	Whitelist     bool
	ModUpdate     bool

	SoftModOptions softmodOptions
}

type softmodOptions struct {
	Restrict          bool
	FriendlyFire      bool
	CleanMap          bool
	DisableBlueprints bool
	Cheats            bool
	SlowConnect       slowConnect
}
type slowConnect struct {
	Enabled      bool
	Speed        float32
	ConnectSpeed float32
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

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

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

/* Used for map names */
func randomBase64String(l int) string {
	buff := make([]byte, int(math.Ceil(float64(l)/float64(1.33333333333))))
	rand.Read(buff)
	str := base64.RawURLEncoding.EncodeToString(buff)
	/* strip 1 extra character we get from odd length results */
	return str[:l]
}

func setGlobalDefaults() {
	/* Automatic global defaults */
	if Global.Paths.DataFiles.DBFile == "" {
		Global.Paths.DataFiles.DBFile = "playerdb.dat"
		_, err := os.Create(Global.Paths.DataFiles.DBFile)
		if err != nil {
			cwlog.DoLogCW("Could not create playerdb.dat")
		}
	}
	if Global.Paths.DataFiles.RecordPlayers == "" {
		Global.Paths.DataFiles.RecordPlayers = "recordPlayers.dat"
		_, err := os.Create(Global.Paths.DataFiles.RecordPlayers)
		if err != nil {
			cwlog.DoLogCW("Could not create recordPlayers.dat")
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
	if Global.Factorio.RCONPass == "" {
		Global.Factorio.RCONPass = randomBase64String(64)
		cwlog.DoLogCW("No RCON password specified. Random one generated.")
	}
	if Global.GroupName == "" {
		Global.GroupName = randomBase64String(3)
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

	if Global.Paths.FactorioPrefix == "" {
		Global.Paths.FactorioPrefix = "fact-"
	}
	if Global.Paths.ChatWirePrefix == "" {
		Global.Paths.ChatWirePrefix = "cw-"
	}
	if Global.Paths.Folders.Saves == "" {
		Global.Paths.Folders.Saves = "saves"
	}
	if Global.Paths.Folders.UpdateCache == "" {
		Global.Paths.Folders.UpdateCache = Global.Paths.Folders.ServersRoot + "/update-cache/"
	}
	if Global.Paths.Folders.MapPreviews == "" {
		Global.Paths.Folders.MapPreviews = Global.Paths.Folders.ServersRoot + "/public_html/map-preview/"
	}
	if Global.Paths.Folders.MapArchives == "" {
		Global.Paths.Folders.MapArchives = Global.Paths.Folders.ServersRoot + "/public_html/archive/"
	}
	if Global.Paths.URLs.ArchiveURL == "" {
		Global.Paths.URLs.ArchiveURL = "https://" + Global.Paths.URLs.Domain + "/~username/map-preview/"
	}
	if Global.Paths.URLs.ArchiveURL == "" {
		Global.Paths.URLs.ArchiveURL = "https://" + Global.Paths.URLs.Domain + "/~username/archive/"
	}
	if Global.Paths.Binaries.ImgCmd == "" {
		Global.Paths.Binaries.ImgCmd = "/usr/bin/convert"
	}
	if Global.Paths.Binaries.RmCmd == "" {
		Global.Paths.Binaries.RmCmd = "/bin/rm"
	}
	if Global.Paths.Binaries.Shell == "" {
		Global.Paths.Binaries.Shell = "/bin/bash"
	}
	if Global.Paths.Binaries.ZipCmd == "" {
		Global.Paths.Binaries.ZipCmd = "/usr/bin/unzip"
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
	if Global.Options.PreviewSettings.JPGQuality == "" {
		Global.Options.PreviewSettings.JPGQuality = "85"
	}
	if Global.Options.PreviewSettings.JPGScale == "" {
		Global.Options.PreviewSettings.JPGScale = "256x256"
	}
	if Global.Options.PreviewSettings.PNGRes == "" {
		Global.Options.PreviewSettings.PNGRes = "256"
	}
	if Global.Options.PreviewSettings.PNGScale == "" {
		Global.Options.PreviewSettings.PNGRes = "1"
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
		file, err := ioutil.ReadFile(constants.CWGlobalConfig)

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

func WriteLCfg() bool {
	tempPath := constants.CWLocalConfig + "." + Local.Callsign + ".tmp"
	finalPath := constants.CWLocalConfig

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(Local); err != nil {
		cwlog.DoLogCW("WriteLCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("WriteLCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteLCfg: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename Lcfg file.")
		return false
	}

	return true
}

func setLocalDefaults() {
	/* Automatic local defaults */
	if Local.Name == "" {
		Local.Name = "unnamed"
	}
	if Local.Callsign == "" {
		Local.Callsign = "a"
	}
	if Local.Port <= 0 {
		Local.Port = 7000
	}
	if Local.Settings.AFKMin <= 0 {
		Local.Settings.AFKMin = 15
	}
	if Local.Settings.AutosaveMin <= 0 {
		Local.Settings.AutosaveMin = 15
	}
	if Local.Channel.ChatChannel == "" {
		cwlog.DoLogCW("ReadLCfg: ChatID not set, this MUST be set to a valid Discord channel ID!")
		Local.Channel.ChatChannel = "MY DISCORD CHANNEL ID"
	}
}

func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadLCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateLCfg()
		Local = newcfg
		setLocalDefaults()
		if !Local.Settings.AutoPause {
			Local.Settings.AutoPause = true
		}
		WriteLCfg() /* Write the defaults */
		return true
	} else { /* Just read the config */

		file, err := ioutil.ReadFile(constants.CWLocalConfig)

		if file != nil && err == nil {
			newcfg := CreateLCfg()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				cwlog.DoLogCW("ReadLCfg: Unmarshal failure")
				cwlog.DoLogCW(err.Error())
				return false
			}

			Local = newcfg
			setLocalDefaults()

			/* Automatic local defaults */
			found := false
			for _, t := range constants.MapTypes {
				if Local.Settings.MapPreset == t {
					found = true
				}
			}
			if !found {
				Local.Settings.MapPreset = constants.MapTypes[1]
				cwlog.DoLogCW("ReadLCfg: MapPreset not valid, setting to " + Local.Settings.MapPreset)
			}

			if newcfg.Options.Whitelist {
				ServerPrefix = constants.MembersPrefix
			} else {
				ServerPrefix = ""
			}
			return true
		} else {
			cwlog.DoLogCW("ReadLCfg: ReadFile failure")
			return false
		}

	}
}

func CreateLCfg() local {
	newcfg := local{}
	return newcfg
}
