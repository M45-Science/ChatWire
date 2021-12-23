package cfg

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"

	"ChatWire/botlog"
	"ChatWire/constants"
)

var Local config
var Global gconfig

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
	ResetPingString   string
	DefaultUPSRate    int
	DisableBlueprints bool
	EnableCheats      bool
	HideAutosaves     bool

	ChannelData    ChannelDataStruct
	SlowConnect    SlowConnectStruct
	SoftModOptions SoftModOptionsStruct
}

type gconfig struct {
	Version string

	RconPortOffset int
	RconPass       string

	DiscordData    DiscordDataStruct
	AdminData      AdminData
	RoleData       RoleDataStruct
	PathData       PathDataStruct
	MapPreviewData MapPreviewDataStruct

	DiscordCommandPrefix string
	ResetPingString      string

	AuthServerBans bool
}

type AdminData struct {
	IDs   []string
	Names []string
}

//Global
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

type DiscordDataStruct struct {
	Token   string
	GuildID string

	StatTotalChannelID    string
	StatMemberChannelID   string
	StatRegularsChannelID string

	ReportChannelID   string
	AnnounceChannelID string
}

type RoleDataStruct struct {
	Moderator string
	Admin     string
	Patreon   string
	Nitro     string
	Regular   string
	Member    string
	New       string
}

type MapPreviewDataStruct struct {
	Args       string
	Res        string
	Scale      string
	JPGQuality string
	JPGScale   string
}

//Local
type ChannelDataStruct struct {
	Pos    int
	ChatID string
}

type SlowConnectStruct struct {
	SlowConnect  bool
	DefaultSpeed float32
	ConnectSpeed float32
}

type SoftModOptionsStruct struct {
	DoWhitelist    bool
	RestrictMode   bool
	FriendlyFire   bool
	CleanMapOnBoot bool
}

func WriteGCfg() bool {
	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(Global); err != nil {
		botlog.DoLog("WriteGCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(constants.CWGlobalConfig)

	if err != nil {
		botlog.DoLog("WriteGCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(constants.CWGlobalConfig, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteGCfg: WriteFile failure")
	}

	return true
}

func ReadGCfg() bool {

	_, err := os.Stat(constants.CWGlobalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadGCfg: os.Stat failed")
		return false

	} else {

		file, err := ioutil.ReadFile(constants.CWGlobalConfig)

		if file != nil && err == nil {
			cfg := CreateGCfg()

			err := json.Unmarshal([]byte(file), &cfg)
			if err != nil {
				botlog.DoLog("ReadGCfg: Unmashal failure")
				botlog.DoLog(err.Error())
				os.Exit(1)
			}

			Global = cfg

			return true
		} else {
			botlog.DoLog("ReadGCfg: ReadFile failure")
			return false
		}
	}
}

func CreateGCfg() gconfig {
	cfg := gconfig{Version: "0.0.1"}
	return cfg
}

func WriteLCfg() bool {
	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(Local); err != nil {
		botlog.DoLog("WriteLCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(constants.CWLocalConfig)

	if err != nil {
		botlog.DoLog("WriteLCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(constants.CWLocalConfig, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteLCfg: WriteFile failure")
	}

	return true
}

func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadLCfg: os.Stat failed")
		return false

	} else {

		file, err := ioutil.ReadFile(constants.CWLocalConfig)

		if file != nil && err == nil {
			cfg := CreateLCfg()

			err := json.Unmarshal([]byte(file), &cfg)
			if err != nil {
				botlog.DoLog("ReadLCfg: Unmashal failure")
				botlog.DoLog(err.Error())
				os.Exit(1)
			}

			Local = cfg

			return true
		} else {
			botlog.DoLog("ReadLCfg: ReadFile failure")
			return false
		}
	}
}

func CreateLCfg() config {
	cfg := config{Version: "0.0.1"}
	return cfg
}
