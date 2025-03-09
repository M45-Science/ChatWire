package util

import (
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
)

func GetFactorioFolder() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/"
}

func GetModsFolder() string {
	//Mod folder path
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"
}

func GetSavesFolder() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves
}

/*  IsPatreon checks if player has patreon role */
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

/*  IsNitro checks if player has nitro role */
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

/* Convert string to bool
 * True, error */
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

/* Bool to string */
func BoolToOnOff(b bool) string {
	if b {
		return "on"
	} else {
		return "off"
	}
}

/* Delete old signal files */
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
