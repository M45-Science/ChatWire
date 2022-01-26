package support

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/disc"
	"os"
	"strings"
)

/*  IsPatreon checks if user has patreon role */
func IsPatreon(id string) bool {
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.PatreonRoleID {
						return true
					}
				}
			}
		}
	}
	return false
}

/*  IsNitro checks if user has nitro role */
func IsNitro(id string) bool {
	if id == "" || disc.DS == nil {
		return false
	}
	g := disc.Guild

	if g != nil {
		for _, m := range g.Members {
			if m.User.ID == id {
				for _, r := range m.Roles {
					if r == cfg.Global.RoleData.NitroRoleID {
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
		strings.ToLower(txt) == "1" {
		return true, false
	} else if strings.ToLower(txt) == "false" ||
		strings.ToLower(txt) == "f" ||
		strings.ToLower(txt) == "no" ||
		strings.ToLower(txt) == "n" ||
		strings.ToLower(txt) == "off" ||
		strings.ToLower(txt) == "0" {
		return false, false
	}

	return false, true
}

/* Bool to sting */
func BoolToString(b bool) string {
	if b {
		return "on"
	} else {
		return "off"
	}
}

/* Delete old signal files */
func clearOldSignals() {
	if err := os.Remove(".qrestart"); err == nil {
		botlog.DoLog("old .qrestart removed.")
	}
	if err := os.Remove(".queue"); err == nil {
		botlog.DoLog("old .queue removed.")
	}
	if err := os.Remove(".stop"); err == nil {
		botlog.DoLog("old .stop removed.")
	}
	if err := os.Remove(".newmap"); err == nil {
		botlog.DoLog("old .newmap removed.")
	}
	if err := os.Remove(".message"); err == nil {
		botlog.DoLog("old .message removed.")
	}
	if err := os.Remove(".start"); err == nil {
		botlog.DoLog("old .start removed.")
	}
	if err := os.Remove(".halt"); err == nil {
		botlog.DoLog("old .halt removed.")
	}
}
