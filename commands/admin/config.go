package admin

import (
	"ChatWire/disc"
	"ChatWire/glob"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

/* Change server settings */
func Config(s *discordgo.Session, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()
	buf := ""

	for _, o := range a.Options {
		for _, co := range SettingList {
			if co.Name == o.Name {
				if o.Type == discordgo.ApplicationCommandOptionBoolean {
					if o.BoolValue() {
						*co.BData = true
					} else {
						*co.BData = false
					}
					buf = buf + fmt.Sprintf("%v: set to: %v\n", co.Name, *co.BData)
				} else if o.Type == discordgo.ApplicationCommandOptionString {
					co.SData = glob.Ptr(o.StringValue())
					buf = buf + fmt.Sprintf("%v: set to: %v\n", co.Name, *co.SData)
				} else if o.Type == discordgo.ApplicationCommandOptionInteger {
					val := int(o.IntValue())
					if val > co.MaxInt || val < co.MinInt {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						co.IData = &val
						buf = buf + fmt.Sprintf("%v: set to: %v\n", co.Name, *co.IData)
					}
				}
			} else if o.Type == discordgo.ApplicationCommandOptionNumber {
				if co.Type == TYPE_F32 {
					val := float32(o.FloatValue())
					if val > co.MaxF32 || val < co.MinF32 {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						co.FData32 = &val
						buf = buf + fmt.Sprintf("%v: set to: %v\n", co.Name, *co.FData32)
					}
				} else if co.Type == TYPE_F64 {
					val := float64(o.FloatValue())
					if val > co.MaxF64 || val < co.MinF64 {
						buf = buf + fmt.Sprintf("%v: invalid value %v\n", co.Name, val)
					} else {
						co.FData64 = &val
						buf = buf + fmt.Sprintf("%v: set to: %v\n", co.Name, *co.FData64)
					}
				}
			}
		}
	}
	if buf != "" {
		disc.EphemeralResponse(s, i, "Status:", buf)
	}
}
