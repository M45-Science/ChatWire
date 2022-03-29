package admin

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

/* Set map seed */
func SetSeed(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		fact.CMS(m.ChannelID, "Invalid seed, numbers only. 0  for reset")
		return
	}
	cfg.Local.Seed = uint64(num)
	if cfg.Local.Seed > 0 {
		fact.CMS(m.ChannelID, fmt.Sprintf("Map seed set to: %v (one use)", cfg.Local.Seed))
	} else {
		fact.CMS(m.ChannelID, "Map seed cleared.")
	}
	cfg.WriteLCfg()

}
