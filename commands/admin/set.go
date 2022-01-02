package admin

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/support"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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

func checkMapGen(text string) bool {

	genNames := GetMapGenNames()
	for _, name := range genNames {
		if strings.EqualFold(name, text) {
			return true
		}
	}
	return false
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

	maxint int
	maxf32 float32
	maxf64 float64

	minint int
	minf32 float32
	minf64 float64

	validstrings []string
	maxstrlen    int
	minstrlen    int

	checkString func(string) bool
	listString  func() []string
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

		maxstrlen: 64,
		minstrlen: 4,
		sdata:     &cfg.Local.Name,
	},
	{
		Name: "Port",
		Desc: "Server port",
		Type: "int",

		maxint: 65535 - cfg.Global.RconPortOffset,
		minint: 1024,

		idata: &cfg.Local.Port,
	},
	{
		Name: "MapPreset",
		Desc: "Map preset",
		Type: "string",

		maxstrlen:    64,
		minstrlen:    4,
		validstrings: constants.MapTypes,

		sdata: &cfg.Local.MapPreset,
	},
	{
		Name: "MapGenPreset",
		Desc: "Map generation preset",
		Type: "string",

		minstrlen: 4,
		maxstrlen: 64,

		checkString: checkMapGen,
		listString:  GetMapGenNames,

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

		minstrlen: 4,
		maxstrlen: 256,

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

		maxf32: 10.0,
		minf32: 0.1,

		fdata32: &cfg.Local.SlowConnect.DefaultSpeed,
	},
	{
		Name: "ConnectSpeed",
		Desc: "Speed set by SlowConnect while player is connecting.",
		Type: "float32",

		maxf32: 10.0,
		minf32: 0.1,

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
	arg1 := ""
	arg2thru := ""

	if arglen > 0 {
		arg1 = strings.ToLower(args[0])
	}
	if arglen > 1 {
		arg2thru = strings.Join(args[1:], " ")
	}

	if arg1 == "help" || arg1 == "" {

		buf := ""
		for _, setting := range settingList {
			data := ""
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
			if data == "" {
				data = "(empty)"
			}
			buf = buf + fmt.Sprintf("%v\n`%v: %v`\n", setting.Desc, setting.Name, data)
		}
		buf = buf + "\n`set <setting>` will show options."
		fact.CMS(m.ChannelID, buf)
	} else if arg1 != "" {
		found := false
		for _, setting := range settingList {
			if strings.EqualFold(setting.Name, arg1) {
				found = true
				if setting.Type == "string" {
					if len(setting.validstrings) > 0 {
						foundValid := false
						for _, validstring := range setting.validstrings {
							if strings.EqualFold(validstring, arg2thru) {
								foundValid = true
								break
							}
						}
						if !foundValid {
							buf := "```"
							buf = buf + "Valid options below:\n"
							for _, validstring := range setting.validstrings {
								buf = buf + fmt.Sprintf("%v\n", validstring)
							}
							buf = buf + "```"
							fact.CMS(m.ChannelID, buf)
							return
						}
					} else if setting.checkString != nil {
						if !setting.checkString(arg2thru) {
							fact.CMS(m.ChannelID, "Invalid string.")
							if setting.listString != nil {
								buf := "```"
								buf = buf + "Valid options below:\n"
								for _, validstring := range setting.listString() {
									buf = buf + fmt.Sprintf("%v\n", validstring)
								}
								buf = buf + "```"
								fact.CMS(m.ChannelID, buf)
							}
							return
						}
					}
					if setting.minstrlen > 0 && len(arg2thru) < setting.minstrlen {
						fact.CMS(m.ChannelID, "String too short. Minimum length: "+strconv.Itoa(setting.minstrlen))
						return
					} else if setting.maxstrlen > 0 && len(arg2thru) > setting.maxstrlen {
						fact.CMS(m.ChannelID, "String too long. Maximum length: "+strconv.Itoa(setting.maxstrlen))
					}
					*setting.sdata = arg2thru
					fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, arg2thru))
				} else if setting.Type == "int" {
					val, err := strconv.Atoi(arg2thru)
					if err != nil {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Numbers only!", setting.Name))
						return
					} else if val > setting.maxint || val < setting.minint {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.minint, setting.maxint))
						return
					} else {
						*setting.idata = val
						cfg.WriteLCfg()
						fact.GenerateFactorioConfig()
						fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, val))
					}
				} else if setting.Type == "bool" {
					val, err := support.StringToBool(arg2thru)
					if err == true {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be true/on/t/1 or false/off/f/0!", setting.Name))
						return
					} else {
						*setting.bdata = val
						cfg.WriteLCfg()
						fact.GenerateFactorioConfig()
						fact.CMS(m.ChannelID, fmt.Sprintf("%v is now %v", setting.Name, support.BoolToString(val)))
					}
				} else if setting.Type == "float32" {
					val64, err := strconv.ParseFloat(arg2thru, 32)
					val := float32(val64)

					if err != nil {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be a number!", setting.Name))
						return
					} else if val > setting.maxf32 || val < setting.minf32 {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.minf32, setting.maxf32))
						return
					} else {
						*setting.fdata32 = val
						cfg.WriteLCfg()
						fact.GenerateFactorioConfig()
						fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, val))
					}
				} else if setting.Type == "float64" {
					val, err := strconv.ParseFloat(arg2thru, 64)
					if err != nil {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be a number!", setting.Name))
						return
					} else if val > setting.maxf64 || val < setting.minf64 {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.minf64, setting.maxf64))
						return
					} else {
						*setting.fdata64 = val
						cfg.WriteLCfg()
						fact.GenerateFactorioConfig()
						fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, val))
					}
				} else {
					fact.CMS(m.ChannelID, fmt.Sprintf("Unknown type %v for %v", setting.Type, setting.Name))
				}
			}
		}
		if !found {
			fact.CMS(m.ChannelID, "No setting by that name.")
		}
	} else {
		fact.CMS(m.ChannelID, "`set <setting> <value>`")
	}
}
