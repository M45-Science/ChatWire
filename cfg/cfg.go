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

	"ChatWire/botlog"
	"ChatWire/constants"
)

var Local config
var Global gconfig

//Local config
type config struct {
	Version string

	ServerCallsign string
	Name           string
	Port           int

	MapPreset    string
	MapGenPreset string

	AutoStart         bool
	AutoUpdate        bool
	UpdateFactExp     bool
	ResetScheduleText string
	WriteStatsDisc    bool
	ReportNewBans     bool
	ResetPingString   string
	HideAutosaves     bool
	DoWhitelist       bool

	FactorioData LFactDataStruct

	ChannelData    ChannelDataStruct
	SlowConnect    SlowConnectStruct
	SoftModOptions SoftModOptionsStruct
}

//Global config (shared with other servers)
type gconfig struct {
	Version string
	Domain  string

	RconPortOffset int
	RconPass       string

	GroupName      string
	FactorioData   GFactDataStruct
	DiscordData    DiscordDataStruct
	RoleData       RoleDataStruct
	PathData       PathDataStruct
	MapPreviewData MapPreviewDataStruct

	DiscordCommandPrefix string
	ResetPingString      string

	AuthServerBans bool
}

//Global Factorio data
type GFactDataStruct struct {
	Comment   string
	Username  string
	Token     string
	Autosaves int

	ServerDescription string
}

//Local Factorio settings
type LFactDataStruct struct {
	AutoSaveMinutes int
	AutoPause       bool
	AFKKickMinutes  int
}

//Global, these are paths we need
//bor = based on root
//boh = based on home
//ap = absolute path
type PathDataStruct struct {
	FactorioServersRoot string //root of factorio server
	FactorioHomePrefix  string //per-server
	ChatWireHomePrefix  string //per-server
	FactorioBinary      string

	RecordPlayersFilename string //boh
	SaveFilePath          string //boh
	BanFile               string //ap

	ScriptInserterPath string //bor
	DBFileName         string //bor
	LogCompScriptPath  string //bor
	FactUpdaterPath    string //bor
	FactUpdateCache    string //bor
	MapGenPath         string //bor

	MapPreviewPath   string //ap
	MapArchivePath   string //ap
	ImageMagickPath  string //ap
	ShellPath        string //ap
	RMPath           string //ap
	FactUpdaterShell string //ap
	ZipBinaryPath    string //ap
	MapPreviewURL    string
	ArchiveURL       string
}

//Discord-specific data global
type DiscordDataStruct struct {
	Comment string
	Token   string
	GuildID string

	StatTotalChannelID    string
	StatMemberChannelID   string
	StatRegularsChannelID string

	ReportChannelID   string
	AnnounceChannelID string
}

//Discord role info, global
type RoleDataStruct struct {
	ModeratorRoleName string
	RegularRoleName   string
	MemberRoleName    string
	NewRoleName       string
	PatreonRoleName   string
	NitroRoleName     string

	Comment         string
	ModeratorRoleID string
	RegularRoleID   string
	MemberRoleID    string
	NewRoleID       string
	PatreonRoleID   string
	NitroRoleID     string
}

//Map preview generation settings, global
type MapPreviewDataStruct struct {
	Comment    string
	Args       string
	Res        string
	Scale      string
	JPGQuality string
	JPGScale   string
}

//Discord data, per-server
type ChannelDataStruct struct {
	Comment string
	Pos     int
	ChatID  string
}

//Local, Factorio setting
type SlowConnectStruct struct {
	SlowConnect  bool
	DefaultSpeed float32
	ConnectSpeed float32
}

//Local soft-mod options
type SoftModOptionsStruct struct {
	RestrictMode      bool
	FriendlyFire      bool
	CleanMapOnBoot    bool
	DefaultUPSRate    int
	DisableBlueprints bool
	EnableCheats      bool
}

func WriteGCfg() bool {
	tempPath := constants.CWGlobalConfig + "." + Local.ServerCallsign + ".tmp"
	finalPath := constants.CWGlobalConfig

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	Global.Version = "0.0.1"

	if err := enc.Encode(Global); err != nil {
		botlog.DoLog("WriteGCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("WriteGCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteGCfg: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		botlog.DoLog("Couldn't rename Gcfg file.")
		return false
	}

	return true
}

//Used for map names
func randomBase64String(l int) string {
	buff := make([]byte, int(math.Ceil(float64(l)/float64(1.33333333333))))
	rand.Read(buff)
	str := base64.RawURLEncoding.EncodeToString(buff)
	return str[:l] // strip 1 extra character we get from odd length results
}

//Read global/shared server config data
func ReadGCfg() bool {

	_, err := os.Stat(constants.CWGlobalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateGCfg()
		Global = newcfg

		//Automatic global defaults
		if Global.PathData.DBFileName == "" {
			Global.PathData.DBFileName = "playerdb.dat"
			_, err := os.Create(Global.PathData.DBFileName)
			if err != nil {
				botlog.DoLog("Could not create playerdb.dat")
				return false
			}
		}
		if Global.PathData.MapGenPath == "" {
			Global.PathData.MapGenPath = "map-gen-json"

			err := os.MkdirAll(Global.PathData.FactorioServersRoot+"/"+Global.PathData.MapGenPath, os.ModePerm)
			if err != nil {
				botlog.DoLog("Could not create map-gen-json directory.")
				return false
			}
		}
		if Global.Domain == "" {
			Global.Domain = "private"
		}
		if Global.RconPortOffset == 0 {
			Global.RconPortOffset = 10000
		}
		if Global.RconPass == "" {
			Global.RconPass = randomBase64String(64)
			botlog.DoLog("No RCON password specified. Random one generated.")
		}
		if Global.GroupName == "" {
			Global.GroupName = randomBase64String(3)
			botlog.DoLog("No group name specified. Random one generated.")
		}
		if Global.DiscordCommandPrefix == "" {
			Global.DiscordCommandPrefix = "$"
		}

		if Global.PathData.FactorioServersRoot == "" {
			ex, err := os.Executable()
			if err == nil {
				exPath := filepath.Dir(ex)
				p := filepath.Clean(filepath.Join(exPath, ".."))
				Global.PathData.FactorioServersRoot = p
			}
		}

		if Global.PathData.FactorioHomePrefix == "" {
			Global.PathData.FactorioHomePrefix = "fact-"
		}
		if Global.PathData.ChatWireHomePrefix == "" {
			Global.PathData.ChatWireHomePrefix = "cw-"
		}
		if Global.PathData.RecordPlayersFilename == "" {
			Global.PathData.RecordPlayersFilename = "most-player.dat"
		}
		if Global.PathData.SaveFilePath == "" {
			Global.PathData.SaveFilePath = "saves"
		}
		if Global.PathData.FactUpdateCache == "" {
			Global.PathData.FactUpdateCache = Global.PathData.FactorioServersRoot + "/update-cache/"
		}
		if Global.PathData.MapPreviewPath == "" {
			Global.PathData.MapPreviewPath = Global.PathData.FactorioServersRoot + "/public_html/map-preview/"
		}
		if Global.PathData.MapArchivePath == "" {
			Global.PathData.MapArchivePath = Global.PathData.FactorioServersRoot + "/public_html/archive/"
		}
		if Global.PathData.MapPreviewURL == "" {
			Global.PathData.MapPreviewURL = "http://" + Global.Domain + "/~username/map-preview/"
		}
		if Global.PathData.ArchiveURL == "" {
			Global.PathData.ArchiveURL = "http://" + Global.Domain + "/~username/archive/"
		}
		if Global.PathData.ImageMagickPath == "" {
			Global.PathData.ImageMagickPath = "/usr/bin/convert"
		}
		if Global.PathData.RMPath == "" {
			Global.PathData.RMPath = "/bin/rm"
		}
		if Global.PathData.ShellPath == "" {
			Global.PathData.ShellPath = "/bin/bash"
		}
		if Global.PathData.ZipBinaryPath == "" {
			Global.PathData.ZipBinaryPath = "/usr/bin/unzip"
		}
		if Global.PathData.FactorioBinary == "" {
			Global.PathData.FactorioBinary = "bin/x64/factorio"
		}
		if Global.DiscordData.GuildID == "" {
			botlog.DoLog("No Discord Guild ID specified. This MUST be set!")
			Global.DiscordData.GuildID = "MY DISCORD GUILD/SERVER ID"
		}
		if Global.DiscordData.Token == "" {
			botlog.DoLog("No Discord Token specified. This MUST be set!")
			Global.DiscordData.Token = "MY DISCORD BOT TOKEN"
		}
		if Global.FactorioData.Username == "" {
			botlog.DoLog("No Factorio Username specified. This MUST be set!")
			Global.FactorioData.Username = "MY FACTORIO USERNAME"
		}
		if Global.FactorioData.Token == "" {
			botlog.DoLog("No Factorio Token specified. This MUST be set!")
			Global.FactorioData.Token = "MY FACTORIO TOKEN"
		}
		return true
	} else { //Otherwise just read in the config
		file, err := ioutil.ReadFile(constants.CWGlobalConfig)

		if file != nil && err == nil {
			newcfg := CreateGCfg()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				botlog.DoLog("ReadGCfg: Unmarshal failure")
				botlog.DoLog(err.Error())
				return false
			}

			Global = newcfg

			return true
		} else {
			botlog.DoLog("ReadGCfg: ReadFile failure")
			return false
		}
	}
}

//Make new empty gconfig data
func CreateGCfg() gconfig {
	newcfg := gconfig{Version: "0.0.2"}
	newcfg.FactorioData.Comment = "THESE ARE REQUIRED!"
	newcfg.DiscordData.Comment = "TOKEN AND GUILD ID ARE REQUIRED!"
	newcfg.RoleData.Comment = "THESE IDS ARE AUTOMATIC! DO NOT EDIT! ONLY SUPPLY ROLE NAMES!"
	newcfg.MapPreviewData.Comment = "https://wiki.factorio.com/Command_line_parameters"
	return newcfg
}

//Write local server settings file
func WriteLCfg() bool {
	tempPath := constants.CWLocalConfig + "." + Local.ServerCallsign + ".tmp"
	finalPath := constants.CWLocalConfig

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	Local.Version = "0.0.1"

	if err := enc.Encode(Local); err != nil {
		botlog.DoLog("WriteLCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("WriteLCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteLCfg: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		botlog.DoLog("Couldn't rename Lcfg file.")
		return false
	}

	return true
}

//Read local server settings file
func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadLCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateLCfg()
		Local = newcfg

		//Automatical local defaults
		if Local.Name == "" {
			Local.Name = "unnamed"
		}
		if Local.ServerCallsign == "" {
			Local.ServerCallsign = "a"
		}
		if Local.Port <= 0 {
			Local.Port = 34197
		}
		if Local.ChannelData.ChatID == "" {
			botlog.DoLog("ReadLCfg: ChatID not set, this MUST be set to a valid Discord channel ID!")
			Local.ChannelData.ChatID = "MY DISCORD CHANNEL ID"
		}
		WriteLCfg() //Write the defaults
		return true
	} else { //Just read the config

		file, err := ioutil.ReadFile(constants.CWLocalConfig)

		if file != nil && err == nil {
			newcfg := CreateLCfg()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				botlog.DoLog("ReadLCfg: Unmarshal failure")
				botlog.DoLog(err.Error())
				return false
			}

			Local = newcfg

			//Automatic local defaults
			found := false
			for _, t := range constants.MapTypes {
				if Local.MapPreset == t {
					found = true
				}
			}
			if !found {
				Local.MapPreset = constants.MapTypes[1]
				botlog.DoLog("ReadLCfg: MapPreset not valid, setting to " + Local.MapPreset)
			}

			return true
		} else {
			botlog.DoLog("ReadLCfg: ReadFile failure")
			return false
		}

	}
}

//Make empty local config
func CreateLCfg() config {
	newcfg := config{Version: "0.0.2"}
	newcfg.ChannelData.Comment = "CHANNEL ID REQUIRED! POSITION IS OPTIONAL!"
	return newcfg
}
