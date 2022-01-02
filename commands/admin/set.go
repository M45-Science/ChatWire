package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/support"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

//Change server settings
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
		for _, setting := range SettingList {
			data := ""
			limits := ""
			if setting.Type == "string" {
				data = *setting.SData
				if setting.ValidStrings != nil {
					limits = "(valid check)"
				} else if setting.CheckString != nil {
					limits = "(valid check)"
				}
			} else if setting.Type == "int" {
				data = fmt.Sprintf("%v", *setting.IData)
				limits = fmt.Sprintf("(%v-%v)", setting.MinInt, setting.MaxInt)
			} else if setting.Type == "bool" {
				data = support.BoolToString(*setting.BData)
			} else if setting.Type == "float32" {
				data = fmt.Sprintf("%v", *setting.FData32)
				limits = fmt.Sprintf("(%v-%v)", setting.MinF32, setting.MaxF32)
			} else if setting.Type == "float64" {
				data = fmt.Sprintf("%v", *setting.FData64)
				limits = fmt.Sprintf("(%v-%v)", setting.MinF64, setting.MaxF64)
			}
			if data == "" {
				data = "(empty)"
			}
			buf = buf + fmt.Sprintf("%v\n`%v: %v %v`\n", setting.Desc, setting.Name, data, limits)
		}
		buf = buf + "\n`set <setting>` will show options. (EMPTY) can be used to blank a string.\n"
		fact.CMS(m.ChannelID, buf)
	} else if arg1 != "" {
		found := false
		for _, setting := range SettingList {
			if strings.EqualFold(setting.Name, arg1) {
				found = true
				if setting.Type == "string" {
					if arg2thru == "(EMPTY)" {
						arg2thru = ""
					}
					if len(setting.ValidStrings) > 0 {
						foundValid := false
						for _, validstring := range setting.ValidStrings {
							if strings.EqualFold(validstring, arg2thru) {
								foundValid = true
								break
							}
						}
						if !foundValid {
							buf := "```"
							buf = buf + "Valid options below:\n"
							for _, validstring := range setting.ValidStrings {
								buf = buf + fmt.Sprintf("%v\n", validstring)
							}
							buf = buf + "```"
							fact.CMS(m.ChannelID, buf)
							return
						}
					} else if setting.CheckString != nil {
						if !setting.CheckString(arg2thru) {
							if setting.ListString != nil {
								buf := "```"
								buf = buf + "Valid options below:\n"
								for _, validstring := range setting.ListString() {
									buf = buf + fmt.Sprintf("%v\n", validstring)
								}
								buf = buf + "```"
								fact.CMS(m.ChannelID, buf)
							}
							return
						}
					}
					if setting.MinStrLen > 0 && len(arg2thru) < setting.MinStrLen {
						fact.CMS(m.ChannelID, "String too short. Minimum length: "+strconv.Itoa(setting.MinStrLen))
						return
					} else if setting.MaxStrLen > 0 && len(arg2thru) > setting.MaxStrLen {
						fact.CMS(m.ChannelID, "String too long. Maximum length: "+strconv.Itoa(setting.MaxStrLen))
					}
					*setting.SData = arg2thru
					fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, arg2thru))
				} else if setting.Type == "int" {
					val, err := strconv.Atoi(arg2thru)
					if err != nil {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Numbers only!", setting.Name))
						return
					} else if val > setting.MaxInt || val < setting.MinInt {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.MinInt, setting.MaxInt))
						return
					} else {
						*setting.IData = val
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
						*setting.BData = val
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
					} else if val > setting.MaxF32 || val < setting.MinF32 {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.MinF32, setting.MaxF32))
						return
					} else {
						*setting.FData32 = val
						cfg.WriteLCfg()
						fact.GenerateFactorioConfig()
						fact.CMS(m.ChannelID, fmt.Sprintf("Set %v to %v", setting.Name, val))
					}
				} else if setting.Type == "float64" {
					val, err := strconv.ParseFloat(arg2thru, 64)
					if err != nil {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be a number!", setting.Name))
						return
					} else if val > setting.MaxF64 || val < setting.MinF64 {
						fact.CMS(m.ChannelID, fmt.Sprintf("Invalid value for %v. Must be between %v and %v!", setting.Name, setting.MinF64, setting.MaxF64))
						return
					} else {
						*setting.FData64 = val
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
