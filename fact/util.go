package fact

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"
)

/* Prgram shutdown */
func DoExit(delay bool) {

	/* Show stats */
	tnow := time.Now()
	tnow = tnow.Round(time.Second)
	mm := GetManMinutes()
	cwlog.DoLogCW(fmt.Sprintf("Stats: Man-hours: %.4f, Activity index: %.4f, Uptime: %v", float64(mm)/60.0, float64(mm)/tnow.Sub(glob.Uptime.Round(time.Second)).Minutes(), tnow.Sub(glob.Uptime.Round(time.Second)).String()))

	time.Sleep(3 * time.Second)
	/* This kills all loops! */
	glob.ServerRunning = false

	cwlog.DoLogCW("CW closing, load/save db, and waiting for locks...")
	LoadPlayers()
	WritePlayers()

	time.Sleep(1 * time.Second)

	/* File locks */
	glob.PlayerListWriteLock.Lock()
	glob.RecordPlayersWriteLock.Lock()

	cwlog.DoLogCW("Closing log files.")
	glob.GameLogDesc.Close()
	glob.CWLogDesc.Close()

	_ = os.Remove("cw.lock")
	/* Logs are closed, don't report */

	if disc.DS != nil {
		disc.DS.Close()
	}

	fmt.Println("Goodbye.")
	if delay {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
	}
	os.Exit(1)
}

func AddFactColor(color string, text string) string {
	text = sclean.StripControlAndSubSpecial(text)
	text = sclean.RemoveDiscordMarkdown(text)
	text = sclean.RemoveFactorioTags(text)

	color = sclean.StripControlAndSubSpecial(color)
	color = sclean.RemoveFactorioTags(color)
	color = strings.TrimSpace(color)
	color = strings.ToLower(color)

	return fmt.Sprintf("[color=%v]%v[/color]", color, text)
}

/* Generate full path to Factorio binary */
func GetFactorioBinary() string {
	bloc := ""
	if strings.HasPrefix(cfg.Global.PathData.FactorioBinary, "/") {
		/* Absolute path */
		bloc = cfg.Global.PathData.FactorioBinary
	} else {
		/* Relative path */
		bloc = cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.FactorioBinary
	}
	return bloc
}

/* Write a Discord message to the buffer */
func CMS(channel string, text string) {

	/* Split at newlines, so we can batch neatly */
	lines := strings.Split(text, "\n")

	disc.CMSBufferLock.Lock()

	for _, line := range lines {

		if len(line) <= 2000 {
			var item disc.CMSBuf
			item.Channel = channel
			item.Text = line

			disc.CMSBuffer = append(disc.CMSBuffer, item)
		} else {
			cwlog.DoLogCW("CMS: Line too long! Discarding...")
		}
	}

	disc.CMSBufferLock.Unlock()
}

/* Log AND send this message to Discord */
func LogCMS(channel string, text string) {
	cwlog.DoLogCW(text)
	CMS(channel, text)
}
