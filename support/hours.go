package support

import (
	"fmt"
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
)

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
					if time.Now().After(shutTime) {
						break
					}
					if WithinHours() {
						buf := fmt.Sprintf("Time was adjusted to %v - %v GMT, shutdown timer aborted.",
							cfg.Local.Options.PlayStartHour,
							cfg.Local.Options.PlayEndHour)

						fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, buf)
						break
					}
					time.Sleep(time.Second)
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
