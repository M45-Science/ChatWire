package moderator

import (
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

const (
	TYPE_STRING  = 0
	TYPE_INT     = 1
	TYPE_BOOL    = 2
	TYPE_F32     = 3
	TYPE_F64     = 4
	TYPE_CHANNEL = 5
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
	TYPE_CHANNEL,
}

/* Global settings */
var GSettingList = []SettingListData{
	{
		Name:      "group-name",
		ShortDesc: "Group Name",
		Desc:      "The name of the server group, used as prefix to name.",
		Type:      TYPE_STRING,

		MaxStrLen: 5,
		MinStrLen: 2,
		SData:     &cfg.Global.GroupName,
	},
	{
		Name:      "primary-server",
		ShortDesc: "Primary Server",
		Desc:      "Specify a server callsign to handle all global commands.",
		Type:      TYPE_STRING,

		MaxStrLen: 2,
		MinStrLen: 1,
		SData:     &cfg.Global.PrimaryServer,
	},

	/* Discord */
	{
		Name:      "report-channel",
		ShortDesc: "Report Channel",
		Desc:      "Channel-ID for user reports.",
		Type:      TYPE_CHANNEL,

		SData: &cfg.Global.Discord.ReportChannel,
	},
	{
		Name:      "announce-channel",
		ShortDesc: "Announce Channel",
		Desc:      "Channel-ID for announcements.",
		Type:      TYPE_CHANNEL,

		SData: &cfg.Global.Discord.AnnounceChannel,
	},

	/* Options */
	{
		Name:      "description",
		ShortDesc: "Description",
		Desc:      "This description is shown in the server browser.",
		Type:      TYPE_STRING,

		MaxStrLen: 128,
		MinStrLen: 1,
		SData:     &cfg.Global.Options.Description,
	},

	{
		Name:      "autosave-max",
		ShortDesc: "Autosave Max",
		Desc:      "Maximum number of autosaves to keep.",
		Type:      TYPE_INT,

		MaxInt: 1024,
		MinInt: 64,
		IData:  &cfg.Global.Options.AutosaveMax,
	},
	{
		Name:      "sus-shutup",
		ShortDesc: "Sus Shutup",
		Desc:      "Disable new-player suspicious activity warning.",
		Type:      TYPE_BOOL,

		BData: &cfg.Global.Options.ShutupSusWarn,
	},
	{
		Name:      "disable-spam-protect",
		ShortDesc: "Disable Spam Protect",
		Desc:      "Disable chat-spam protection (auto-ban)",
		Type:      TYPE_BOOL,

		BData: &cfg.Global.Options.DisableSpamProtect,
	},
}

/* Local Settings */
var SettingList = []SettingListData{
	{
		Name:      "name",
		ShortDesc: "Server Name",
		Desc:      "Server name, do not include callsign/letter.",
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
		Desc:      "Map generator to use, SELECT 'NONE' FOR MODS THAT REMOVE VANILLA RESOURCES.",
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
		Desc:      "Auto-start and auto-restart Factorio.",
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
		Desc:      "When updating factorio, use experimental releases.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.ExpUpdates,
	},
	{
		Name:      "disable-blueprints",
		ShortDesc: "No Blueprints",
		Desc:      "Disable blueprints (softmod)",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData:             &cfg.Local.Options.SoftModOptions.DisableBlueprints,
		FactUpdateCommand: "/blueprints",
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
		Name:      "hide-research",
		ShortDesc: "Hide Science Research",
		Desc:      "Don't display science research on Discord.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.HideResearch,
	},
	{
		Name:      "members-only",
		ShortDesc: "Members Only",
		Desc:      "Only members or higher can connect.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.MembersOnly,
	},
	{
		Name:      "regulars-only",
		ShortDesc: "regulars only",
		Desc:      "Only regulars or higher can connect.",
		Type:      TYPE_BOOL,

		DefBool: false,

		BData: &cfg.Local.Options.RegularsOnly,
	},
	{
		Name:      "restrict-new-players",
		ShortDesc: "Restrict New",
		Desc:      "Enable new-player restrictions. (softmod).",
		Type:      TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.Restrict,

		DefBool: false,

		FactUpdateCommand: "/restrict",
	},
	{
		Name:      "friendly-fire",
		ShortDesc: "Friendly Fire",
		Desc:      "Allow damage to teammates or buildings (softmod)",
		Type:      TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.FriendlyFire,

		DefBool: false,

		FactUpdateCommand: "/friendlyfire",
	},
	{
		Name:      "one-life",
		ShortDesc: "One-Life",
		Desc:      "Enable onelife mode, cannot be undone.",
		Type:      TYPE_BOOL,

		BData: &cfg.Local.Options.SoftModOptions.OneLife,

		DefBool: false,

		FactUpdateCommand: "/onelife",
	},
	{
		Name:      "afk-kick-mins",
		ShortDesc: "AFK Mins",
		Desc:      "If idle/afk, kick after this amount of time (game time, at 60UPS).",
		Type:      TYPE_INT,

		MaxInt: 120,
		MinInt: 5,
		DefInt: 15,

		IData: &cfg.Local.Settings.AFKMin,
	},
	{
		Name:      "autosave-mins",
		ShortDesc: "Autosave Mins",
		Desc:      "Save the map this often (game time, 60ups)",
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
		Desc:      "Auto-update installed game mods.",
		Type:      TYPE_BOOL,
		DefBool:   true,

		BData: &cfg.Local.Options.ModUpdate,
	},
	{
		Name:      "map-seed",
		ShortDesc: "Map Seed",
		Desc:      "Seed for map gen, clears after map reset.",
		Type:      TYPE_INT,
		DefInt:    0,

		IData: &cfg.Local.Settings.Seed,
	},
	{
		Name:      "soft-mod-inject",
		ShortDesc: "SoftMod Inject",
		Desc:      "Inject softmod scripts.",
		Type:      TYPE_BOOL,
		DefBool:   false,

		BData: &cfg.Local.Options.SoftModOptions.InjectSoftMod,
	},
	{
		Name:      "map-reset-skip",
		ShortDesc: "Map reset skip",
		Desc:      "Skip the next map reset.",
		Type:      TYPE_BOOL,
		DefBool:   false,

		BData: &cfg.Local.Options.SkipReset,
	},
	{
		Name:      "Speed",
		ShortDesc: "Game Speed",
		Desc:      "Set the game speed (1.0 normal)",
		Type:      TYPE_F32,

		MaxF32: 10,
		MinF32: 0.01,
		DefInt: 1,

		FData32:           &cfg.Local.Options.Speed,
		FactUpdateCommand: "/gspeed",
	},
	{
		Name:      "scenario",
		ShortDesc: "Scenario",
		Desc:      "Specify a scenario to use for new maps, NONE for none.",
		Type:      TYPE_STRING,

		MaxStrLen: 64,
		MinStrLen: 1,

		SData: &cfg.Local.Settings.Scenario,
	},
}

/* Get list of map generation presets, because an invalid one will make map generation fail */
func GetMapGenNames() []string {
	path := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Folders.MapGenerators
	files, err := os.ReadDir(path)
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
	if text == "" || text == "none" {
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
