package support

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
)

func checkHours() {
	for glob.ServerRunning {

		if cfg.Local.Options.PlayHourEnable {
			shouldPlay := WithinHours()

			if !shouldPlay && fact.FactIsRunning {
				buf := fmt.Sprintf("It is now past %v GMT, server will stop in 10 minutes.",
					cfg.Local.Options.PlayEndHour)
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)

				time.Sleep(time.Minute * 10)

				fact.FactAutoStart = false
				fact.QuitFactorio("Time is up...")
			} else if shouldPlay && !fact.FactIsRunning {
				buf := fmt.Sprintf("It is now past %v GMT, server will now start.",
					cfg.Local.Options.PlayStartHour)
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
				fact.FactAutoStart = true
			}
		}

		time.Sleep(time.Minute)
	}
}

func WithinHours() bool {

	if cfg.Local.Options.PlayHourEnable {
		curTime := time.Now().UTC().Hour()

		if cfg.Local.Options.PlayStartHour > cfg.Local.Options.PlayEndHour {
			if curTime <= cfg.Local.Options.PlayStartHour &&
				curTime >= cfg.Local.Options.PlayEndHour {
				return true
			}
		} else {
			if curTime >= cfg.Local.Options.PlayStartHour &&
				curTime <= cfg.Local.Options.PlayEndHour {
				return true
			}
		}
		return false
	} else {
		return true
	}
}

/* Check if an idiot pasted their register code to a chat channel */
func ProtectIdiots(text string) bool {
	idiotID := ""
	checkme := strings.ToLower(text)
	checkme = sclean.AlphaOnly(checkme)

	/* Only run if there are active registration codes */
	if len(glob.PassList) > 0 {
		for i, o := range glob.PassList {
			password := strings.ToLower(o.Code)
			password = sclean.AlphaOnly(password)

			/* Found an active register code */
			if strings.Contains(checkme, password) {
				idiotID = i
				break
			}

			/* Just in case they miss part of it when copying/pasting */
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
		/* We got one, invalidate their code */
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

/* Bool to string */
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
