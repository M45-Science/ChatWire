package disc

import (
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
)

// TestCreateChannel connects to Discord using credentials from environment
// variables and creates then deletes a temporary channel. It is skipped when
// CW_TEST_TOKEN or CW_TEST_GUILD is not set.
func TestCreateChannel(t *testing.T) {
	token := os.Getenv("CW_TEST_TOKEN")
	guild := os.Getenv("CW_TEST_GUILD")
	if token == "" || guild == "" {
		t.Skip("CW_TEST_TOKEN or CW_TEST_GUILD not set")
	}

	s, err := discordgo.New("Bot " + token)
	if err != nil {
		t.Fatalf("session create: %v", err)
	}
	defer s.Close()

	name := "cw-test-" + strconv.FormatInt(time.Now().Unix(), 10)
	ch, err := s.GuildChannelCreate(guild, name, discordgo.ChannelTypeGuildText)
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if _, err := s.ChannelDelete(ch.ID); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
}
