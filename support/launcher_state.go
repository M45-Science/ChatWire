package support

import (
	"time"

	"ChatWire/cwlog"
)

var (
	BotIsReady bool
)

// Wait for a moment, so we don't lose factorio booting message on first connect.
func waitForDiscord() {
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
