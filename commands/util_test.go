package commands

import (
	"strings"
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

func TestFormatBusyMessageIncludesCommandMetadata(t *testing.T) {
	startedAt := time.Now().Add(-(time.Hour + 2*time.Minute + 3*time.Second))
	msg := formatBusyMessage(activeCommandState{
		Name:      "sync-mods",
		User:      "alice",
		StartedAt: startedAt,
	})

	for _, part := range []string{
		`"sync-mods"`,
		"alice",
		"1 hour 2 minutes 3 seconds",
	} {
		if !strings.Contains(msg, part) {
			t.Fatalf("expected busy message to contain %q, got %q", part, msg)
		}
	}
}

func TestFormatBusyMessageFallsBackWithoutMetadata(t *testing.T) {
	msg := formatBusyMessage(activeCommandState{})
	want := "Another command is already in progress. Please wait and try again."
	if msg != want {
		t.Fatalf("expected %q, got %q", want, msg)
	}
}

func TestFormatBusyElapsedRoundsToSecondsAndLimitsToThreeFields(t *testing.T) {
	got := formatBusyElapsed(time.Hour + 2*time.Minute + 3*time.Second + 400*time.Millisecond)
	want := "1 hour 2 minutes 3 seconds"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestIsCommandLockExemptAllowsChatwireForceReboot(t *testing.T) {
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "chatwire",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "action", Type: discordgo.ApplicationCommandOptionString, Value: "force-reboot"},
				},
			},
		},
	}

	if !isCommandLockExempt(i) {
		t.Fatal("expected /chatwire force-reboot to bypass command lock")
	}
}

func TestIsCommandLockExemptRejectsOtherChatwireActions(t *testing.T) {
	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Type: discordgo.InteractionApplicationCommand,
			Data: discordgo.ApplicationCommandInteractionData{
				Name: "chatwire",
				Options: []*discordgo.ApplicationCommandInteractionDataOption{
					{Name: "action", Type: discordgo.ApplicationCommandOptionString, Value: "reload-config"},
				},
			},
		},
	}

	if isCommandLockExempt(i) {
		t.Fatal("expected non-force-reboot /chatwire actions to remain locked")
	}
}
