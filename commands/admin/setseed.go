package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
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
		fact.CMS(cfg.Local.Channel.ChatChannel, "Invalid seed, numbers only. 0  for reset")
		return
	}
	cfg.Local.Settings.Seed = uint64(num)
	if cfg.Local.Settings.Seed > 0 {
		fact.CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("Map seed set to: %v (one use)", cfg.Local.Settings.Seed))
	} else {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Map seed cleared.")
	}
	cfg.WriteLCfg()

}
