package support

import (
	"time"

	"ChatWire/cwlog"
	"ChatWire/glob"
)

var (
	BotIsReady bool
)

// Wait for a moment, so we don't lose factorio booting message on first connect.
func waitForDiscord() {
	if glob.NoDiscord != nil && *glob.NoDiscord {
		return
	}
	if BotIsReady {
		return
	}
	for x := 0; x <= 10; x++ {
		if BotIsReady {
			return
		}
		cwlog.DoLogCW("Waiting for Discord...")
		time.Sleep(time.Second)
	}
}
