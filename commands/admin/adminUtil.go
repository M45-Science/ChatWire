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
	Name      string
	ShortDesc string
	Desc      string
	Type      int

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
		Name:      "name",
		ShortDesc: "Server Name",
		Desc:      "Server name, not including callsign/letter.",
		Type:      TYPE_STRING,

		MaxStrLen: 64,
		MinStrLen: 1,
		SData:     &cfg.Local.Name,

		FactUpdateCommand: "/cname",
	},
	{
		Name:      "port",
		ShortDesc: "Port Number",
		Desc:      "UDP port the server will run on.",
		Type:      TYPE_INT,

		MaxInt: 65535 - cfg.Global.Options.RconOffset,
		MinInt: 1024,

		IData: &cfg.Local.Port,
	},
	{
		Name:      "map-preset",
		ShortDesc: "Map preset",
		Desc:      "Factorio map preset to use.",
		Type:      TYPE_STRING,

		MaxStrLen:    64,
		MinStrLen:    1,
		ValidStrings: constants.MapTypes,
		CheckString:  CheckMapTypes,

		SData: &cfg.Local.Settings.MapPreset,
	},
	{
		Name:      "map-generator",
		ShortDesc: "Map Generator",
		Desc:      "Map generator to use, select 'none' for mods that remove vanilla resources.",
		Type:      TYPE_STRING,

		MinStrLen: 1,
		MaxStrLen: 64,

		CheckString: CheckMapGen,
		ListString:  GetMapGenNames,

		SData: &cfg.Local.Settings.MapGenerator,
	},
	{
		Name:      "auto-start-factorio",
		ShortDesc: "Auto-Start",
		Desc:      "Auto-start Factorio when ChatWire boots.",
		Type:      TYPE_BOOL,

		DefBool: true,

		BData: &cfg.Local.Options.AutoStart,
	},
	{
		Name:      "auto-update-factorio",
		ShortDesc: "Auto-Update",
		Desc:      "Auto-update Factorio to newest stable version.",
		Type:      TYPE_BOOL,

		DefBool: true,

		BData: &cfg.Local.Options.AutoUpdate,
	},
	{
		Name:      "auto-update-experimental",
		ShortDesc: "Experimental Updates",
		Desc:      "Force Factorio updater to use experimental versions if auto-update is on.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.ExpUpdates,
	},
	{
		Name:      "reset-schedule-description",
		ShortDesc: "Reset Schedule",
		Desc:      "Description of map reset schedule for server description and info window.",
		Type:      TYPE_STRING,

		MinStrLen: 1,
		MaxStrLen: 99,

		SData:             &cfg.Local.Options.ScheduleText,
		FactUpdateCommand: "/resetint",
	},
	{
		Name:      "disable-blueprints",
		ShortDesc: "No Blueprints",
		Desc:      "Disable blueprints",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData:             &cfg.Local.Options.SoftModOptions.DisableBlueprints,
		FactUpdateCommand: "/blueprints",
	},
	{
		Name:      "enable-cheats",
		ShortDesc: "Sandbox Mode",
		Desc:      "Cheats enabled (sandbox mode)",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData:             &cfg.Local.Options.SoftModOptions.Cheats,
		FactUpdateCommand: "/cheats",
	},
	{
		Name:      "hide-autosaves",
		ShortDesc: "Hide Autosaves",
		Desc:      "Don't display autosaves on Discord.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.HideAutosaves,
	},
	{
		Name:      "slow-connect",
		ShortDesc: "Slow Connect",
		Desc:      "Slow game to connect-speed when players are connecting.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.SoftModOptions.SlowConnect.Enabled,
	},
	{
		Name:      "default-speed",
		ShortDesc: "Default Speed",
		Desc:      "Normal speed while playing, 1.0 is normal, 0.5 would be half (30fps)",
		Type:      TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,
		DefF32: 1.0,

		FData32: &cfg.Local.Options.SoftModOptions.SlowConnect.Speed,
	},
	{
		Name:      "connect-speed",
		ShortDesc: "Connect Speed",
		Desc:      "Speed to slow to when players are connecting, 1.0 is normal speed so 0.5 is half (30fps).",
		Type:      TYPE_F32,

		MaxF32: 10.0,
		MinF32: 0.1,
		DefF32: 0.5,

		FData32: &cfg.Local.Options.SoftModOptions.SlowConnect.ConnectSpeed,
	},
	{
		Name:      "members-only",
		ShortDesc: "Members Only",
		Desc:      "Only members, regulars and moderators can connect.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.Whitelist,
	},
	{
		Name:      "restrict-new-players",
		ShortDesc: "Restrict New",
		Desc:      "New player permission restrictions.",
		Type:      TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.Restrict,

		DefBool: false,

		FactUpdateCommand: "/restrict",
	},
	{
		Name:      "friendly-fire",
		ShortDesc: "Friendly Fire",
		Desc:      "Allow friendly fire (damage to teammates or buildings).",
		Type:      TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.FriendlyFire,

		DefBool: false,

		FactUpdateCommand: "/friendlyfire",
	},
	{
		Name:      "afk-kick-mins",
		ShortDesc: "AFK Mins",
		Desc:      "If AFK, kick after this amount of time (in game time, based on 60fps).",
		Type:      TYPE_INT,

		MaxInt: 120,
		MinInt: 5,
		DefInt: 15,

		IData: &cfg.Local.Settings.AFKMin,
	},
	{
		Name:      "autosave-mins",
		ShortDesc: "Autosave Mins",
		Desc:      "Save the map every X minutes (in game time, based on 60fps)",
		Type:      TYPE_INT,

		MaxInt: 30,
		MinInt: 5,
		DefInt: 10,

		IData: &cfg.Local.Settings.AutosaveMin,
	},
	{
		Name:      "auto-pause",
		ShortDesc: "Auto Pause",
		Desc:      "Game pauses when no players are connected.",
		Type:      TYPE_BOOL,
		DefBool:   true,

		BData: &cfg.Local.Settings.AutoPause,
	},
	{
		Name:      "auto-mod-update",
		ShortDesc: "Auto Mod Update",
		Desc:      "Auto-update game mods.",
		Type:      TYPE_BOOL,
		DefBool:   true,

		BData: &cfg.Local.Options.AutoUpdate,
	},
	{
		Name:      "auto-clean-map",
		ShortDesc: "Auto Clean",
		Desc:      "Run CleanMap mod on boot (only works with mod installed)",
		Type:      TYPE_BOOL,
		DefBool:   false,

		BData: &cfg.Local.Options.SoftModOptions.CleanMap,
	},
	{
		Name:      "map-seed",
		ShortDesc: "Map Seed",
		Desc:      "Seed for map gen, clears after map reset.",
		Type:      TYPE_INT,
		DefInt:    0,

		IData: &cfg.Local.Settings.Seed,
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

	output = append(output, "none")
	for _, f := range files {
		if strings.HasSuffix(f.Name(), "-gen.json") {
			output = append(output, strings.TrimSuffix(f.Name(), "-gen.json"))
		}
	}
	return output
}

/* See if this map gen exists */
func CheckMapGen(text string) bool {

	/* Allow no generator */
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

func CheckMapTypes(text string) bool {

	names := constants.MapTypes
	for _, name := range names {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
}
