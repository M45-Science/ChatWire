package admin

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/commands/moderator"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/support"
)

/* Change server settings */
func GConfigServer(s *discordgo.Session, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()
	buf := ""

	/* Check all values, Discord limits could be bypassed */
	for _, o := range a.Options {
		for _, co := range moderator.GSettingList {
			if strings.EqualFold(co.Name, o.Name) {
				if o.Type == discordgo.ApplicationCommandOptionBoolean {
					if o.BoolValue() {
						*co.BData = true
					} else {
						*co.BData = false
					}
					buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.BData)
					if co.FactUpdateCommand != "" && fact.FactorioBooted {
						fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", support.BoolToString(*co.BData)))
						buf = buf + " (live update)\n"
					} else {
						buf = buf + "\n"
					}
				} else if o.Type == discordgo.ApplicationCommandOptionString {
					val := o.StringValue()
					if co.CheckString != nil {
						if !co.CheckString(val) {
							buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
							continue
						}
					}
					if co.MaxStrLen != 0 {
						if len(val) > co.MaxStrLen {
							buf = buf + fmt.Sprintf("%v: text too long %v\n", co.Name, val)
							continue
						}
					}
					if co.MinStrLen != 0 {
						if len(val) < co.MinStrLen {
							buf = buf + fmt.Sprintf("%v: text too short %v\n", co.Name, val)
							continue
						}
					}

					*co.SData = o.StringValue()
					buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.SData)
					if co.FactUpdateCommand != "" && fact.FactorioBooted {
						fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", val))
						buf = buf + " (live update)\n"
					} else {
						buf = buf + "\n"
					}
				} else if o.Type == discordgo.ApplicationCommandOptionInteger {
					val := int(o.IntValue())
					if val > co.MaxInt || val < co.MinInt {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						*co.IData = val
						buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.IData)

						if co.FactUpdateCommand != "" && fact.FactorioBooted {
							fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", val))
							buf = buf + " (live update)\n"
						} else {
							buf = buf + "\n"
						}
					}
				}
			} else if o.Type == discordgo.ApplicationCommandOptionNumber {
				if co.Type == moderator.TYPE_F32 {
					val := float32(o.FloatValue())
					if val > co.MaxF32 || val < co.MinF32 {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						*co.FData32 = val
						buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.FData32)

						if co.FactUpdateCommand != "" && fact.FactorioBooted {
							fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", val))
							buf = buf + " (live update)\n"
						} else {
							buf = buf + "\n"
						}
					}
				} else if co.Type == moderator.TYPE_F64 {
					val := float64(o.FloatValue())
					if val > co.MaxF64 || val < co.MinF64 {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						*co.FData64 = val
						buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.FData64)
						if co.FactUpdateCommand != "" && fact.FactorioBooted {
							fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", val))
							buf = buf + " (live update)\n"
						} else {
							buf = buf + "\n"
						}
					}
				}
			}
		}
	}
	if buf != "" {
		support.ConfigSoftMod()
		if cfg.WriteGCfg() {
			disc.EphemeralResponse(s, i, "Status:", buf)
		} else {
			disc.EphemeralResponse(s, i, "Error:", "Unable to save cw-global, check file permissions.")
		}
	}
}
