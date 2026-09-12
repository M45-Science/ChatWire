package support

import (
	"strings"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func ConfigSoftMod() {
	supporters := append([]string{}, disc.RoleList.Patreons...)
	supporters = append(supporters, disc.RoleList.Supporters...)
	fact.WriteSoftModCommand("config", map[string]any{
		"server_name":    strings.ToUpper(cfg.Local.Callsign + "-" + cfg.Local.Name),
		"restrict":       cfg.Local.Options.SoftModOptions.Restrict,
		"friendly_fire":  cfg.Local.Options.SoftModOptions.FriendlyFire,
		"one_life":       cfg.Local.Options.SoftModOptions.OneLife,
		"reset_duration": resetDurationText(),
		"reset_date":     resetDateText(),
		"blueprints":     !cfg.Local.Options.SoftModOptions.DisableBlueprints,
		"cheats":         cfg.Local.Options.SoftModOptions.Cheats,
		"supporters":     supporters,
		"nitro":          append([]string{}, disc.RoleList.NitroBooster...),
		"speed":          cfg.Local.Options.Speed,
	})
}
