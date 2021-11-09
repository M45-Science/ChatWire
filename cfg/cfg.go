package cfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"../constants"
	"../glob"
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
	LogID  string
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
		log("WriteGCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(constants.CWGlobalConfig)

	if err != nil {
		log("WriteGCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(constants.CWGlobalConfig, outbuf.Bytes(), 0644)

	if err != nil {
		log("WriteGCfg: WriteFile failure")
	}

	return true
}

func ReadGCfg() bool {

	_, err := os.Stat(constants.CWGlobalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		log("ReadGCfg: os.Stat failed")
		return false

	} else {

		file, err := ioutil.ReadFile(constants.CWGlobalConfig)

		if file != nil && err == nil {
			cfg := CreateGCfg()

			err := json.Unmarshal([]byte(file), &cfg)
			if err != nil {
				log("ReadGCfg: Unmashal failure")
				log(err.Error())
				os.Exit(1)
			}

			Global = cfg

			return true
		} else {
			log("ReadGCfg: ReadFile failure")
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
		log("WriteLCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(constants.CWLocalConfig)

	if err != nil {
		log("WriteLCfg: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(constants.CWLocalConfig, outbuf.Bytes(), 0644)

	if err != nil {
		log("WriteLCfg: WriteFile failure")
	}

	return true
}

func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		log("ReadLCfg: os.Stat failed")
		return false

	} else {

		file, err := ioutil.ReadFile(constants.CWLocalConfig)

		if file != nil && err == nil {
			cfg := CreateLCfg()

			err := json.Unmarshal([]byte(file), &cfg)
			if err != nil {
				log("ReadLCfg: Unmashal failure")
				log(err.Error())
				os.Exit(1)
			}

			Local = cfg

			return true
		} else {
			log("ReadLCfg: ReadFile failure")
			return false
		}
	}
}

func CreateLCfg() config {
	cfg := config{Version: "0.0.1"}
	return cfg
}

func log(text string) {

	t := time.Now()
	date := fmt.Sprintf("%02d-%02d-%04d_%02d-%02d-%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute(), t.Second())

	buf := fmt.Sprintf("%s %s", date, text)
	_, err := glob.BotLogDesc.WriteString(buf + "\n")
	if err != nil {
		fmt.Println(err)
	}
	println(buf)
}
