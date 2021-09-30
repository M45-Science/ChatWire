package admin

import (
	"strconv"
	"strings"

	"../../cfg"
	"../../fact"
	"../../support"
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

func Set(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	arglen := len(args)
	if arglen > 1 {
		arg1 := strings.ToLower(args[0])
		arg2 := strings.ToLower(args[1])

		if arg1 == "name" && arg2 != "" {
			cfg.Local.Name = arg2
			fact.CMS(m.ChannelID, "Name set to: "+cfg.Local.ServerCallsign+"-"+arg2)
			cfg.WriteLCfg()
		} else if arg1 == "port" && arg2 != "" {
			num, err := strconv.Atoi(arg2)
			if err == nil && num > 1 && num < 65535 {
				fact.CMS(m.ChannelID, "Changing port to: "+arg2)
				cfg.Local.Port = num
				cfg.WriteLCfg()
			} else {
				fact.CMS(m.ChannelID, "Invalid port number")
			}
		} else if arg1 == "mappreset" && arg2 != "" {
			fact.CMS(m.ChannelID, "Changing map preset to: "+arg2)
			cfg.Local.MapPreset = arg2
			cfg.WriteLCfg()
		} else if arg1 == "mapgenpreset" && arg2 != "" {
			fact.CMS(m.ChannelID, "Changing map-gen preset to: "+arg2)
			cfg.Local.MapGenPreset = arg2
			cfg.WriteLCfg()
		} else if arg1 == "autostart" && arg2 != "" {
			res, lerr := handlebool("Auto start", arg2, m)
			if !lerr {
				cfg.Local.AutoStart = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "autoupdate" && arg2 != "" {
			res, lerr := handlebool("Auto update", arg2, m)
			if !lerr {
				cfg.Local.AutoUpdate = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "updateexp" && arg2 != "" {
			res, lerr := handlebool("Update factorio to experimental version", arg2, m)
			if !lerr {
				cfg.Local.UpdateFactExp = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "slowconnect" && arg2 != "" {
			res, lerr := handlebool("Slow connect", arg2, m)
			if !lerr {
				cfg.Local.SlowConnect.SlowConnect = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "defaultspeed" && arg2 != "" {
			num, err := strconv.ParseFloat(arg2, 64)
			if err == nil && num > 0.1 && num < 10.0 {
				fact.CMS(m.ChannelID, "Changing default speed to: "+arg2)
				cfg.Local.SlowConnect.DefaultSpeed = float32(num)
				cfg.WriteLCfg()
			} else {
				fact.CMS(m.ChannelID, "Valid speeds: 0.1 to 10.0")
			}
		} else if arg1 == "connectspeed" && arg2 != "" {
			num, err := strconv.ParseFloat(arg2, 64)
			if err == nil && num > 0.1 && num < 10.0 {
				fact.CMS(m.ChannelID, "Changing connect speed to: "+arg2)
				cfg.Local.SlowConnect.ConnectSpeed = float32(num)
				cfg.WriteLCfg()
			} else {
				fact.CMS(m.ChannelID, "Valid speeds: 0.1 to 10.0")
			}
		} else if arg1 == "dowhitelist" && arg2 != "" {
			res, lerr := handlebool("Do whitelist", arg2, m)
			if !lerr {
				cfg.Local.SoftModOptions.DoWhitelist = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "restrictmode" && arg2 != "" {
			res, lerr := handlebool("Restrict mode", arg2, m)
			if !lerr {
				cfg.Local.SoftModOptions.RestrictMode = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "friendlyfire" && arg2 != "" {
			res, lerr := handlebool("Friendly fire", arg2, m)
			if !lerr {
				cfg.Local.SoftModOptions.FriendlyFire = res
				cfg.WriteLCfg()
			}
		} else if arg1 == "cleanmaponboot" && arg2 != "" {
			res, lerr := handlebool("Clean map on boot", arg2, m)
			if !lerr {
				cfg.Local.SoftModOptions.CleanMapOnBoot = res
				cfg.WriteLCfg()
			}
		}
	} else {
		fact.CMS(m.ChannelID, "Usage: ```set <setting> <value>\nSettings:\nName <text>, Port <number>, MapPreset <preset>,  MapGenPreset <text>, AutoStart <on/off>, AutoUpdate <on/off>, UpdateExp <on/off>, SlowConect <on/off>, DefaultSpeed <0.1 to 10.0>, ConnectSpeed <0.1 to 1.0>, DoWhitelist <on/off>, RestrictMode <on/off>, FriendlyFire <true/false>, CleanMapOnBoot <true/false> (requires CleanMap mod)\n```")
	}
}
