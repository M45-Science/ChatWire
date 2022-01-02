package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/support"
	"fmt"

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

	sdata   *string
	idata   *int
	bdata   *bool
	fdata32 *float32
	fdata64 *float64
}

var settingType = []string{
	"string",
	"int",
	"bool",
	"float32",
	"float64",
}

var settingList = []settingListData{
	{
		Name: "Name",
		Desc: "Server name",
		Type: "string",

		sdata: &cfg.Local.Name,
	},
	{
		Name: "Port",
		Desc: "Server port",
		Type: "int",

		idata: &cfg.Local.Port,
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
		Name: "ResetScheduleText",
		Desc: "This is the text displayed that descrbes when the map will be reset",
		Type: "string",

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
		Type: "float32",

		fdata32: &cfg.Local.SlowConnect.DefaultSpeed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed set by SlowConnect while player is connecting.",
		Type: "float32",

		fdata32: &cfg.Local.SlowConnect.ConnectSpeed,
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
	{
		Name: "FriendlyFire",
		Desc: "Friendly fire on/off",
		Type: "bool",

		bdata: &cfg.Local.SoftModOptions.FriendlyFire,
	},
}

func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	arglen := len(args)
	if arglen == 0 {
		//arg1 := strings.ToLower(args[0])
		//arg2 := strings.ToLower(args[1])

		buf := ""
		for _, setting := range settingList {
			data := "(Empty)"
			if setting.Type == "string" {
				data = *setting.sdata
			} else if setting.Type == "int" {
				data = fmt.Sprintf("%d", *setting.idata)
			} else if setting.Type == "bool" {
				data = support.BoolToString(*setting.bdata)
			} else if setting.Type == "float32" {
				data = fmt.Sprintf("%f", *setting.fdata32)
			} else if setting.Type == "float64" {
				data = fmt.Sprintf("%f", *setting.fdata64)
			}
			buf = buf + fmt.Sprintf("%v\n`%v: %v`\n\n", setting.Desc, setting.Name, data)
		}
		buf = buf + "\n`set <setting>` will show options."
		fact.CMS(m.ChannelID, buf)
	}
}
