package admin

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"io/ioutil"
	"strings"
)

const TYPE_STRING = 0
const TYPE_INT = 1
const TYPE_BOOL = 2
const TYPE_F32 = 3
const TYPE_F64 = 4

//Used for $set command
type SettingListData struct {
	Name string
	Desc string
	Type int

	SData   *string
	IData   *int
	BData   *bool
	FData32 *float32
	FData64 *float64

	MaxInt int
	MaxF32 float32
	MaxF64 float64

	MinInt int
	MinF32 float32
	MinF64 float64

	ValidStrings []string
	MaxStrLen    int
	MinStrLen    int

	CheckString       func(string) bool
	ListString        func() []string
	FactUpdateCommand string
}

//List of datatypes for settings
var SettingType = []int{
	TYPE_STRING,
	TYPE_INT,
	TYPE_BOOL,
	TYPE_F32,
	TYPE_F64,
}

//List of settings
var SettingList = []SettingListData{
	{
		Name: "Name",
		Desc: "Server name",
		Type: TYPE_STRING,

		MaxStrLen: 64,
		MinStrLen: 4,
		SData:     &cfg.Local.Name,

		FactUpdateCommand: "/cname",
	},
	{
		Name: "Port",
		Desc: "Server port",
		Type: TYPE_INT,

		MaxInt: 65535 - cfg.Global.RconPortOffset,
		MinInt: 1024,

		IData: &cfg.Local.Port,
	},
	{
		Name: "MapPreset",
		Desc: "Map preset",
		Type: TYPE_STRING,

		MaxStrLen:    64,
		MinStrLen:    4,
		ValidStrings: constants.MapTypes,

		SData: &cfg.Local.MapPreset,
	},
	{
		Name: "MapGenPreset",
		Desc: "Map generation preset",
		Type: TYPE_STRING,

		MinStrLen: 0,
		MaxStrLen: 64,

		CheckString: CheckMapGen,
		ListString:  GetMapGenNames,

		SData: &cfg.Local.MapGenPreset,
	},
	{
		Name: "AutoStart",
		Desc: "Start factorio on boot",
		Type: TYPE_BOOL,

		BData: &cfg.Local.AutoStart,
	},
	{
		Name: "AutoUpdate",
		Desc: "Auto-update factorio",
		Type: TYPE_BOOL,

		BData: &cfg.Local.AutoUpdate,
	},
	{
		Name: "UpdateFactExp",
		Desc: "Update factorio to experimental releases",
		Type: TYPE_BOOL,

		BData: &cfg.Local.UpdateFactExp,
	},
	{
		Name: "ResetScheduleText",
		Desc: "This is the text displayed that descrbes when the map will be reset",
		Type: TYPE_STRING,

		MinStrLen: 4,
		MaxStrLen: 256,

		SData:             &cfg.Local.ResetScheduleText,
		FactUpdateCommand: "/resetint",
	},
	{
		Name: "DisableBlueprints",
		Desc: "Disable blueprints",
		Type: TYPE_BOOL,

		BData:             &cfg.Local.DisableBlueprints,
		FactUpdateCommand: "/blueprints",
	},
	{
		Name: "EnableCheats",
		Desc: "Enable cheats",
		Type: TYPE_BOOL,

		BData:             &cfg.Local.EnableCheats,
		FactUpdateCommand: "/cheats",
	},
	{
		Name: "HideAutosaves",
		Desc: "Hide autosaves from Discord.",
		Type: TYPE_BOOL,

		BData: &cfg.Local.HideAutosaves,
	},
	{
		Name: "SlowConnect",
		Desc: "Lowers game speed while players are connecting, for large maps.",
		Type: TYPE_BOOL,

		BData: &cfg.Local.SlowConnect.SlowConnect,
	},
	{
		Name: "DefaultSpeed",
		Desc: "Speed set by SlowConnect after player is done connecting.",
		Type: TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,

		FData32: &cfg.Local.SlowConnect.DefaultSpeed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed set by SlowConnect while player is connecting.",
		Type: TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,

		FData32: &cfg.Local.SlowConnect.ConnectSpeed,
	},
	{
		Name: "DoWhitelist",
		Desc: "Members-only mode",
		Type: TYPE_BOOL,

		BData: &cfg.Local.SoftModOptions.DoWhitelist,
	},
	{
		Name: "RestrictMode",
		Desc: "Turns on new-player restrictions.",
		Type: TYPE_BOOL,

		BData: &cfg.Local.SoftModOptions.RestrictMode,

		FactUpdateCommand: "/restrict",
	},
	{
		Name: "FriendlyFire",
		Desc: "Friendly fire on/off",
		Type: TYPE_BOOL,

		BData: &cfg.Local.SoftModOptions.FriendlyFire,

		FactUpdateCommand: "/friendlyfire",
	},
	{
		Name: "AFKKickMinutes",
		Desc: "How many minutes before a player is kicked for being AFK",
		Type: TYPE_INT,

		MaxInt: 120,
		MinInt: 5,

		IData: &cfg.Local.FactorioData.AFKKickMinutes,
	},
	{
		Name: "AutoSaveMinutes",
		Desc: "How many minutes between autosaves",
		Type: TYPE_INT,

		MaxInt: 30,
		MinInt: 5,

		IData: &cfg.Local.FactorioData.AutoSaveMinutes,
	},
	{
		Name: "AutoPause",
		Desc: "Pause when no players online.",
		Type: TYPE_BOOL,

		BData: &cfg.Local.FactorioData.AutoPause,
	},
}

//Get list of map generation presets, because an invalid one will make map generation fail
func GetMapGenNames() []string {
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.MapGenPath
	files, err := ioutil.ReadDir(path)
	if err != nil {
		botlog.DoLog(err.Error())
		return nil
	}

	var output []string

	for _, f := range files {
		if strings.HasSuffix(f.Name(), "-gen.json") {
			output = append(output, strings.TrimSuffix(f.Name(), "-gen.json"))
		}
	}
	return output
}

//See if this map gen exists
func CheckMapGen(text string) bool {

	if text == "" {
		return true
	}
	genNames := GetMapGenNames()
	for _, name := range genNames {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
}
