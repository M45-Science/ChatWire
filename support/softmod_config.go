package support

import (
	"strings"

	"ChatWire/cfg"
	"ChatWire/disc"
	"ChatWire/fact"
)

func ConfigSoftMod() {
	fact.WriteFact("/cname " + strings.ToUpper(cfg.Local.Callsign+"-"+cfg.Local.Name))

	/* Config new-player restrictions */
	if cfg.Local.Options.SoftModOptions.Restrict {
		fact.WriteFact("/restrict on")
	} else {
		fact.WriteFact("/restrict off")
	}

	/* Config friendly fire */
	if cfg.Local.Options.SoftModOptions.FriendlyFire {
		fact.WriteFact("/friendlyfire on")
	} else {
		fact.WriteFact("/friendlyfire off")
	}

	/* Config friendly fire */
	if cfg.Local.Options.SoftModOptions.OneLife {
		fact.WriteFact("/onelife on")
	} else {
		fact.WriteFact("/onelife off")
	}

	UpdateDuration()
	UpdateInterval()

	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		fact.WriteFact("/blueprints off")
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		fact.WriteFact("/enablecheats on")
	}

	/* Patreon list */
	if len(disc.RoleList.Patreons) > 0 {
		fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ", ") + ", " +
			strings.Join(disc.RoleList.Supporters, ", "))
	}
	if len(disc.RoleList.NitroBooster) > 0 {
		fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ", "))
	}

	fact.WriteFact("/gspeed %0.2f", cfg.Local.Options.Speed)

}
