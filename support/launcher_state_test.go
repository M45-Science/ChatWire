package support

import (
	"testing"
	"time"

	"ChatWire/glob"
)

func TestWaitForDiscordReturnsImmediatelyWhenDisabled(t *testing.T) {
	oldNoDiscord := glob.NoDiscord
	disabled := true
	glob.NoDiscord = &disabled
	t.Cleanup(func() { glob.NoDiscord = oldNoDiscord })

	started := time.Now()
	waitForDiscord()
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("waitForDiscord took %v with Discord disabled", elapsed)
	}
}
