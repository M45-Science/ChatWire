package disc

import (
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// TestCreateCommand registers a simple command and deletes it again when
// CW_TEST_TOKEN, CW_TEST_GUILD and CW_TEST_APP are set.
func TestCreateCommand(t *testing.T) {
	token := os.Getenv("CW_TEST_TOKEN")
	guild := os.Getenv("CW_TEST_GUILD")
	app := os.Getenv("CW_TEST_APP")
	if token == "" || guild == "" || app == "" {
		t.Skip("CW_TEST_TOKEN/GUILD/APP not set")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		t.Fatalf("session create: %v", err)
	}
	defer s.Close()

	cmd, err := s.ApplicationCommandCreate(app, guild, &discordgo.ApplicationCommand{
		Name:        "cwtest",
		Description: "temporary test command",
	})
	if err != nil {
		t.Fatalf("create command: %v", err)
	}

	if err := s.ApplicationCommandDelete(app, guild, cmd.ID); err != nil {
		t.Fatalf("delete command: %v", err)
	}
}
