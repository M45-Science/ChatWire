package admin

import (
	"ChatWire/cfg"
	"ChatWire/disc"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

/* Set map seed */
func SetSeed(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	num, err := strconv.Atoi(args[0])
	if err != nil {
		buf := "Numbers only."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return
	}
	cfg.Local.Settings.Seed = uint64(num)
	if cfg.Local.Settings.Seed > 0 {
		buf := fmt.Sprintf("Map seed set to: %v (one use)", cfg.Local.Settings.Seed)

		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	} else {
		buf := "Map seed cleared."
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
	}
	cfg.WriteLCfg()

}
