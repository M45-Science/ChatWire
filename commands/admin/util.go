package admin

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"io/ioutil"
	"strings"
)

//Used for $set command
type SettingListData struct {
	Name string
	Desc string
	Type string

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

	CheckString func(string) bool
	ListString  func() []string
}

var SettingType = []string{
	"string",
	"int",
	"bool",
	"float32",
	"float64",
}

var SettingList = []SettingListData{
	{
		Name: "Name",
		Desc: "Server name",
		Type: "string",

		MaxStrLen: 64,
		MinStrLen: 4,
		SData:     &cfg.Local.Name,
	},
	{
		Name: "Port",
		Desc: "Server port",
		Type: "int",

		MaxInt: 65535 - cfg.Global.RconPortOffset,
		MinInt: 1024,

		IData: &cfg.Local.Port,
	},
	{
		Name: "MapPreset",
		Desc: "Map preset",
		Type: "string",

		MaxStrLen:    64,
		MinStrLen:    4,
		ValidStrings: constants.MapTypes,

		SData: &cfg.Local.MapPreset,
	},
	{
		Name: "MapGenPreset",
		Desc: "Map generation preset",
		Type: "string",

		MinStrLen: 4,
		MaxStrLen: 64,

		CheckString: CheckMapGen,
		ListString:  GetMapGenNames,

		SData: &cfg.Local.MapGenPreset,
	},
	{
		Name: "AutoStart",
		Desc: "Start factorio on boot",
		Type: "bool",

		BData: &cfg.Local.AutoStart,
	},
	{
		Name: "AutoUpdate",
		Desc: "Auto-update factorio",
		Type: "bool",

		BData: &cfg.Local.AutoUpdate,
	},
	{
		Name: "UpdateFactExp",
		Desc: "Update factorio to experimental releases",
		Type: "bool",

		BData: &cfg.Local.UpdateFactExp,
	},
	{
		Name: "ResetScheduleText",
		Desc: "This is the text displayed that descrbes when the map will be reset",
		Type: "string",

		MinStrLen: 4,
		MaxStrLen: 256,

		SData: &cfg.Local.ResetScheduleText,
	},
	{
		Name: "DisableBlueprints",
		Desc: "Disable blueprints",
		Type: "bool",

		BData: &cfg.Local.DisableBlueprints,
	},
	{
		Name: "EnableCheats",
		Desc: "Enable cheats",
		Type: "bool",

		BData: &cfg.Local.EnableCheats,
	},
	{
		Name: "HideAutosaves",
		Desc: "Hide autosaves from Discord.",
		Type: "bool",

		BData: &cfg.Local.HideAutosaves,
	},
	{
		Name: "SlowConnect",
		Desc: "Lowers game speed while players are connecting, for large maps.",
		Type: "bool",

		BData: &cfg.Local.SlowConnect.SlowConnect,
	},
	{
		Name: "DefaultSpeed",
		Desc: "Speed set by SlowConnect after player is done connecting.",
		Type: "float32",

		MaxF32: 10.0,
		MinF32: 0.1,

		FData32: &cfg.Local.SlowConnect.DefaultSpeed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed set by SlowConnect while player is connecting.",
		Type: "float32",

		MaxF32: 10.0,
		MinF32: 0.1,

		FData32: &cfg.Local.SlowConnect.ConnectSpeed,
	},
	{
		Name: "DoWhitelist",
		Desc: "Members-only mode",
		Type: "bool",

		BData: &cfg.Local.SoftModOptions.DoWhitelist,
	},
	{
		Name: "RestrictMode",
		Desc: "Turns on new-player restrictions.",
		Type: "bool",

		BData: &cfg.Local.SoftModOptions.RestrictMode,
	},
	{
		Name: "FriendlyFire",
		Desc: "Friendly fire on/off",
		Type: "bool",

		BData: &cfg.Local.SoftModOptions.FriendlyFire,
	},
}

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

func CheckMapGen(text string) bool {

	genNames := GetMapGenNames()
	for _, name := range genNames {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
}
