package admin

import (
	"io/ioutil"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

const (
	TYPE_STRING = 0
	TYPE_INT    = 1
	TYPE_BOOL   = 2
	TYPE_F32    = 3
	TYPE_F64    = 4
)

/* Used for set command */
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

	DefInt  int
	DefF32  float32
	DefF64  float64
	DefBool bool

	ValidStrings []string
	MaxStrLen    int
	MinStrLen    int

	CheckString       func(string) bool
	ListString        func() []string
	FactUpdateCommand string
}

/* List of datatypes for settings */
var SettingType = []int{
	TYPE_STRING,
	TYPE_INT,
	TYPE_BOOL,
	TYPE_F32,
	TYPE_F64,
}

/* List of settings */
var SettingList = []SettingListData{
	{
		Name: "Name",
		Desc: "Name",
		Type: TYPE_STRING,

		MaxStrLen: 64,
		MinStrLen: 4,
		SData:     &cfg.Local.Name,

		FactUpdateCommand: "/cname",
	},
	{
		Name: "Port",
		Desc: "Port",
		Type: TYPE_INT,

		MaxInt: 65535 - cfg.Global.Options.RconOffset,
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

		SData: &cfg.Local.Settings.MapPreset,
	},
	{
		Name: "MapGenPreset",
		Desc: "Map generator",
		Type: TYPE_STRING,

		MinStrLen: 0,
		MaxStrLen: 64,

		CheckString: CheckMapGen,
		ListString:  GetMapGenNames,

		SData: &cfg.Local.Settings.MapGenerator,
	},
	{
		Name: "AutoStart",
		Desc: "Start on boot",
		Type: TYPE_BOOL,

		DefBool: true,

		BData: &cfg.Local.Options.AutoStart,
	},
	{
		Name: "AutoUpdate",
		Desc: "Auto-update Factorio",
		Type: TYPE_BOOL,

		DefBool: true,

		BData: &cfg.Local.Options.AutoUpdate,
	},
	{
		Name: "UpdateFactExp",
		Desc: "Update Factorio to exp",
		Type: TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.ExpUpdates,
	},
	{
		Name: "ResetScheduleText",
		Desc: "Map reset schedule",
		Type: TYPE_STRING,

		MinStrLen: 4,
		MaxStrLen: 256,

		SData:             &cfg.Local.Options.ScheduleText,
		FactUpdateCommand: "/resetint",
	},
	{
		Name: "DisableBlueprints",
		Desc: "Blueprints disabled",
		Type: TYPE_BOOL,

		DefBool: false,

		BData:             &cfg.Local.Options.SoftModOptions.DisableBlueprints,
		FactUpdateCommand: "/blueprints",
	},
	{
		Name: "EnableCheats",
		Desc: "Cheats enabled",
		Type: TYPE_BOOL,

		DefBool: false,

		BData:             &cfg.Local.Options.SoftModOptions.Cheats,
		FactUpdateCommand: "/cheats",
	},
	{
		Name: "HideAutosaves",
		Desc: "Hide autosaves(Discord)",
		Type: TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.HideAutosaves,
	},
	{
		Name: "SlowConnect",
		Desc: "Slow on connect",
		Type: TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.SoftModOptions.SlowConnect.Enabled,
	},
	{
		Name: "DefaultSpeed",
		Desc: "Speed while playing",
		Type: TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,
		DefF32: 1.0,

		FData32: &cfg.Local.Options.SoftModOptions.SlowConnect.Speed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed while connecting",
		Type: TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,
		DefF32: 0.5,

		FData32: &cfg.Local.Options.SoftModOptions.SlowConnect.ConnectSpeed,
	},
	{
		Name: "DoWhitelist",
		Desc: "Members-only",
		Type: TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.Whitelist,
	},
	{
		Name: "RestrictMode",
		Desc: "New player restrictions",
		Type: TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.Restrict,

		DefBool: false,

		FactUpdateCommand: "/restrict",
	},
	{
		Name: "FriendlyFire",
		Desc: "Friendly fire",
		Type: TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.FriendlyFire,

		DefBool: false,

		FactUpdateCommand: "/friendlyfire",
	},
	{
		Name: "AFKKickMinutes",
		Desc: "AFK kick minutes",
		Type: TYPE_INT,

		MaxInt: 120,
		MinInt: 5,
		DefInt: 15,

		IData: &cfg.Local.Settings.AFKMin,
	},
	{
		Name: "AutoSaveMinutes",
		Desc: "Autosave minutes",
		Type: TYPE_INT,

		MaxInt: 30,
		MinInt: 5,
		DefInt: 10,

		IData: &cfg.Local.Settings.AutosaveMin,
	},
	{
		Name:    "AutoPause",
		Desc:    "Pause when empty",
		Type:    TYPE_BOOL,
		DefBool: true,

		BData: &cfg.Local.Settings.AutoPause,
	},
	{
		Name:    "AutoModUpdate",
		Desc:    "Auto-update game mods",
		Type:    TYPE_BOOL,
		DefBool: true,

		BData: &cfg.Local.Options.AutoUpdate,
	},
	{
		Name:    "CleanMapOnBoot",
		Desc:    "Run CleanMap mod on boot",
		Type:    TYPE_BOOL,
		DefBool: false,

		BData: &cfg.Local.Options.SoftModOptions.CleanMap,
	},
}

/* Get list of map generation presets, because an invalid one will make map generation fail */
func GetMapGenNames() []string {
	path := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Folders.MapGenerators
	files, err := ioutil.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW(err.Error())
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

/* See if this map gen exists */
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
