package admin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/disc"
)

/* Set map seed */
func SetSeed(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	num, err := strconv.Atoi(args[0])
	if err != nil {
		buf := "Numbers only."
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}
	cfg.Local.Settings.Seed = uint64(num)
	if cfg.Local.Settings.Seed > 0 {
		buf := fmt.Sprintf("Map seed set to: %v (one use)", cfg.Local.Settings.Seed)

		disc.EphemeralResponse(s, i, "Complete:", buf)
	} else {
		buf := "Map seed cleared."
		disc.EphemeralResponse(s, i, "Complete:", buf)
	}
	cfg.WriteLCfg()

}
