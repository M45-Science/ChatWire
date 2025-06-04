package util

import (
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
)

// TrimPrefixIgnoreCase removes prefix from s in a case-insensitive manner.
func TrimPrefixIgnoreCase(s, prefix string) string {
	if strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
		return s[len(prefix):]
	}
	return s
}

// ContainsIgnoreCase reports whether substr is within s ignoring case.
func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(
		strings.ToLower(s), strings.ToLower(substr),
	)
}

// GetFactorioFolder returns the path to the Factorio installation for the
// current server.
func GetFactorioFolder() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/"
}

// GetModsFolder returns the path to the mod directory.
func GetModsFolder() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"
}

// GetSavesFolder returns the path to the saves directory.
func GetSavesFolder() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves
}

// IsPatreon checks if player has the Patreon role.
func IsPatreon(id string) bool {
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Patreon) {
						return true
					} else if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Supporter) {
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
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

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

// StringToBool converts a string to a boolean. The second return value
// indicates whether the conversion failed.
func StringToBool(txt string) (bool, bool) {
	if strings.ToLower(txt) == "true" ||
		strings.ToLower(txt) == "t" ||
		strings.ToLower(txt) == "yes" ||
		strings.ToLower(txt) == "y" ||
		strings.ToLower(txt) == "on" ||
		strings.ToLower(txt) == "enable" ||
		strings.ToLower(txt) == "enabled" ||
		strings.ToLower(txt) == "1" {
		return true, false
	} else if strings.ToLower(txt) == "false" ||
		strings.ToLower(txt) == "f" ||
		strings.ToLower(txt) == "no" ||
		strings.ToLower(txt) == "n" ||
		strings.ToLower(txt) == "off" ||
		strings.ToLower(txt) == "disable" ||
		strings.ToLower(txt) == "disabled" ||
		strings.ToLower(txt) == "0" {
		return false, false
	}

	return false, true
}

// BoolToOnOff converts a boolean to the strings "on" or "off".
func BoolToOnOff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

// ClearOldSignals removes leftover signal files from previous runs.
func ClearOldSignals() {
	if err := os.Remove(".qrestart"); err == nil {
		cwlog.DoLogCW("old .qrestart removed.")
	}
	if err := os.Remove(".queue"); err == nil {
		cwlog.DoLogCW("old .queue removed.")
	}
	if err := os.Remove(".stop"); err == nil {
		cwlog.DoLogCW("old .stop removed.")
	}
	if err := os.Remove(".newmap"); err == nil {
		cwlog.DoLogCW("old .newmap removed.")
	}
	if err := os.Remove(".message"); err == nil {
		cwlog.DoLogCW("old .message removed.")
	}
	if err := os.Remove(".start"); err == nil {
		cwlog.DoLogCW("old .start removed.")
	}
	if err := os.Remove(".halt"); err == nil {
		cwlog.DoLogCW("old .halt removed.")
	}
}
