package support

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"fmt"
	"strings"
	"time"
)

var lastDur string

func UpdateDuration() {
	if fact.HasResetTime() {
		buf := "/resetdur " + fact.TimeTillReset()
		if fact.HasResetInterval() {
			buf = buf + " (" + fact.FormatResetInterval() + ")"
		}
		/* Don't write it, if nothing has changed */
		if !strings.EqualFold(buf, lastDur) {
			fact.WriteFact(buf)
		}

		lastDur = buf
	} else {
		buf := "/resetdur"

		/* Don't write it, if nothing has changed */
		if !strings.EqualFold(buf, lastDur) {
			fact.WriteFact(buf)
		}

		lastDur = buf
	}
}

func UpdateInterval() {
	/* Config reset-interval */
	if fact.HasResetInterval() {
		fact.WriteFact("/resetint " + fact.FormatResetTime())
	} else {
		fact.WriteFact("/resetint")
	}
}

func checkHours() {
	for glob.ServerRunning {
		time.Sleep(time.Second * 15)

		if cfg.Local.Options.PlayHourEnable {

			graceString := " Server shutting down."
			if fact.NumPlayers > 0 {
				graceString = " Server will shut down in 10 minutes."
			}
			if !WithinHours() && fact.FactIsRunning && fact.FactAutoStart {
				buf := fmt.Sprintf("It is now time for the map to close (%v-%v GMT).%v",
					cfg.Local.Options.PlayStartHour,
					cfg.Local.Options.PlayEndHour,
					graceString)

				fact.FactChat(buf)
				fact.FactChat(buf)
				fact.FactChat(buf)
				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)

				shutTime := time.Now()
				shutTime = shutTime.Add(time.Minute * 10)

				for fact.NumPlayers > 0 {
					if time.Now().UTC().Sub(shutTime) <= time.Second {
						break
					}
					if WithinHours() {
						buf := fmt.Sprintf("Time was adjusted to %v - %v GMT, shutdown timer aborted.",
							cfg.Local.Options.PlayStartHour,
							cfg.Local.Options.PlayEndHour)

						fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, buf)
						break
					}
				}

				if !WithinHours() {
					fact.SetAutolaunch(false, false)
					fact.QuitFactorio("Time is up...")
				}

			} else if WithinHours() && !fact.FactIsRunning && !fact.FactAutoStart {
				buf := fmt.Sprintf("It is now time for the map to open (%v-%v GMT). Server starting.",
					cfg.Local.Options.PlayStartHour,
					cfg.Local.Options.PlayEndHour)

				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
				fact.SetAutolaunch(true, false)
			}
		}

	}
}

func WithinHours() bool {

	if cfg.Local.Options.PlayHourEnable {
		curTime := time.Now().UTC().Hour()

		if cfg.Local.Options.PlayStartHour > cfg.Local.Options.PlayEndHour {
			if curTime >= cfg.Local.Options.PlayStartHour ||
				curTime < cfg.Local.Options.PlayEndHour {
				return true
			}
		} else {
			if curTime >= cfg.Local.Options.PlayStartHour &&
				curTime < cfg.Local.Options.PlayEndHour {
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
			plen := len(password)
			if plen > 3 {
				trimEnd := password[0 : plen-2]
				trimStart := password[2:plen]

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
