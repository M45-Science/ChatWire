package admin

import (
	"strings"

	"ChatWire/fact"
	"ChatWire/support"

	"github.com/bwmarrin/discordgo"
)

func handlebool(name string, arg string, m *discordgo.MessageCreate) (bool, bool) {
	larg, lerr := support.StringToBool(arg)
	if !lerr {
		fact.CMS(m.ChannelID, name+" is now "+support.BoolToString(larg))
		return larg, false
	} else {
		fact.CMS(m.ChannelID, "Invalid value, use: true/false, t/f, yes/no, y/n, on/off, 1/0")
		return false, true
	}
}

type settingListData struct {
	Name string
	Desc string
	Type string

	sdata *string
	idata *int
	bdata *bool
	fdata *float64
}

var settingType = []string{
	"string",
	"int",
	"bool",
	"float",
}

var settingList = []settingListData{
	{
	Name: "Name",
	Desc: "Server name",
	Type: "string",

	sdata: &cfg.Local.ServerName,
	},
	{
		Name: "Port",
		Desc: "Server port",
		Type: "int",

		idata: &cfg.Local.ServerPort,
	},
	{
		Name: "MapPreset",
		Desc: "Map preset",
		Type: "string",

		sdata: &cfg.Local.MapPreset,
	},
	{
		Name: "MapGenPreset",
		Desc: "Map generation preset",
		Type: "string",

		sdata: &cfg.Local.MapGenPreset,
	},
	{
		Name: "AutoStart",
		Desc: "Start factorio on boot",
		Type: "bool",

		bdata: &cfg.Local.AutoStart,
	},
	{
		Name: "AutoUpdate",
		Desc: "Auto-update factorio",
		Type: "bool",

		bdata: &cfg.Local.AutoUpdate,
	},
	{
		Name: "UpdateFactExp",
		Desc: "Update factorio to experimental releases",
		Type: "bool",

		bdata: &cfg.Local.UpdateFactExp,
	},
	{
		Name:  "ResetScheduleText",
		Desc:  "This is the text displayed that descrbes when the map will be reset",
		Type:  "string",

		sdata: &cfg.Local.ResetScheduleText,
	},
	{
		Name: "DisableBlueprints",
		Desc: "Disable blueprints",
		Type: "bool",

		bdata: &cfg.Local.DisableBlueprints,
	},
	{
		Name: "EnableCheats",
		Desc: "Enable cheats",
		Type: "bool",

		bdata: &cfg.Local.EnableCheats,
	},
	{
		Name: "HideAutosaves",
		Desc: "Hide autosaves from Discord.",
		Type: "bool",

		bdata: &cfg.Local.HideAutosaves,
	},
	{
		Name: "SlowConnect",
		Desc: "Lowers game speed while players are connecting, for large maps.",
		Type: "bool",

		bdata: &cfg.Local.SlowConnect.SlowConnect,
	},
	{
		Name: "DefaultSpeed",
		Desc: "Speed set by SlowConnect after player is done connecting.",
		Type: "float",

		fdata: &cfg.Local.SlowConnect.DefaultSpeed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed set by SlowConnect while player is connecting.",
		Type: "float",

		fdata: &cfg.Local.SlowConnect.ConnectSpeed,
	},
	{
		Name: "DoWhitelist",
		Desc: "Members-only mode",
		Type: "bool",

		bdata: &cfg.Local.SoftModOptions.DoWhitelist,
	},
	{
		Name: "RestrictMode",
		Desc: "Turns on new-player restrictions.",
		Type: "bool",

		bdata: &cfg.Local.SoftModOptions.RestrictMode,
	},


}

func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	arglen := len(args)
	if arglen > 1 {
		arg1 := strings.ToLower(args[0])
		arg2 := strings.ToLower(args[1])

		for 
	}
}
