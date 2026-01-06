package support

import (
	"math/rand/v2"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func startFactorioUpdateLoop() {
	/****************************
	* Check for Factorio updates
	****************************/
	//Every 30 minutes
	go func() {

		if *glob.ProxyURL != "" {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()

			for glob.ServerRunning {
				<-ticker.C

				if !cfg.Local.Options.AutoUpdate {
					continue
				}

				cTime := time.Now()
				if cTime.Minute() == 15 || cTime.Minute() == 45 {
					checkFactUpdate()
				}

			}
		} else {

			for glob.ServerRunning {
				time.Sleep(time.Minute * 30)
				if cfg.Local.Options.AutoUpdate {
					checkFactUpdate()
				}
				//Add 0 to 5 minutes of sleep
				time.Sleep(time.Microsecond * time.Duration((rand.Float64() * 60000000 * 5.0)))
			}
		}
	}()
}

func checkFactUpdate() {
	glob.ResetUpdateMessage()
	_, msg, err, upToDate := factUpdater.DoQuickLatest(false)
	if msg != "" {
		if !err && !upToDate {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Updated", msg, glob.COLOR_CYAN))
			cwlog.DoLogCW(msg)

			newHist := modupdate.ModHistoryItem{InfoItem: true,
				Name: "Factorio Updated", Notes: "To version: " + fact.NewVersion, Date: time.Now()}
			modupdate.AddModHistory(newHist)
		} else if err && !upToDate {
			//glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel , glob.GetUpdateMessage(), "ERROR", msg, glob.COLOR_RED)
			cwlog.DoLogCW(msg)
		}
	}
	glob.ResetUpdateMessage()
}
