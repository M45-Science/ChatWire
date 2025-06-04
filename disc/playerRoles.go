package disc

import (
	"strings"

	"ChatWire/cfg"
)

// IsPatreon checks if player has the Patreon or Supporter role.
func IsPatreon(id string) bool {
	if id == "" || DS == nil {
		return false
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Patreon) ||
						strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Supporter) {
						return true
					}
				}
			}
		}
	}
	return false
}

// IsNitro checks if player has the Nitro role.
func IsNitro(id string) bool {
	if id == "" || DS == nil {
		return false
	}
	g := Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Nitro) {
						return true
					}
				}
			}
		}
	}
	return false
}
