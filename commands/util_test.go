package commands

import (
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

func TestSlashCommandReturnsImmediatelyWhenBusy(t *testing.T) {
	commandLock.Lock()
	defer commandLock.Unlock()

	done := make(chan struct{})
	go func() {
		SlashCommand(nil, &discordgo.InteractionCreate{})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("SlashCommand blocked while another command was in progress")
	}
}
