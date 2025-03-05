package moderator

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
	"ChatWire/util"
)

/* Change server settings */
func ConfigServer(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()
	buf := ""

	/* Check all values, Discord limits could be bypassed */
	for _, o := range a.Options {
		for _, co := range SettingList {
			if strings.EqualFold(co.Name, o.Name) {
				if o.Type == discordgo.ApplicationCommandOptionBoolean {
					if o.BoolValue() {
						*co.BData = true
					} else {
						*co.BData = false
					}
					buf = buf + fmt.Sprintf("%v: set to: %v", co.Name, *co.BData)
					if co.FactUpdateCommand != "" && fact.FactorioBooted {
						fact.WriteFact(co.FactUpdateCommand + fmt.Sprintf(" %v", util.BoolToString(*co.BData)))
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

				} else if o.Type == discordgo.ApplicationCommandOptionNumber {
					if co.Type == TYPE_F32 {
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
					} else if co.Type == TYPE_F64 {
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
	}
	if buf == "" {
		buf = "Setting not found!"
	}
	support.ConfigSoftMod()
	if cfg.WriteLCfg() {
		if !fact.GenerateFactorioConfig() {
			disc.InteractionEphemeralResponse(i, "Error:", "(Unable to write Factorio server settings, check file permissions.")
			return
		}

		disc.InteractionEphemeralResponse(i, "Status:", buf)
		fact.UpdateChannelName() //For channel renaming, only updates if changed.
	} else {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to save cw-local, check file permissions.")
	}

}
