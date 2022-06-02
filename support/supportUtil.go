package support

import (
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"
)

func ProtectIdiots(text string) bool {
	//If there are any active register codes, check if an idiot pasted it in chat
	idiotID := ""
	checkme := strings.ToLower(text)
	checkme = sclean.AlphaOnly(checkme)

	if len(glob.PassList) > 0 {
		for i, o := range glob.PassList {
			password := strings.ToLower(o.Code)
			password = sclean.AlphaOnly(password)

			if strings.Contains(checkme, password) {
				idiotID = i
				break
			}

			/* Just in case they miss part of it when copying */
			clen := len(password)
			if clen > 3 {
				trimEnd := password[0 : clen-2]
				trimStart := password[2:clen]

				if strings.Contains(checkme, trimEnd) {
					idiotID = i
					break
				} else if strings.Contains(checkme, trimStart) {
					idiotID = i
					break
				}
			}
		}
		if idiotID != "" {
			delete(glob.PassList, idiotID)
			return true
		}
		return false
	}

	return false
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
