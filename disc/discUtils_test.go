package disc

import (
	"errors"
	"testing"

	"ChatWire/cfg"

	"github.com/bwmarrin/discordgo"
)

func drainCMSChan() {
	for {
		select {
		case <-CMSChan:
		default:
			return
		}
	}
}

func TestRoleChecksMatchConfiguredRoleCache(t *testing.T) {
	cfg.Global.Discord.Roles.RoleCache.Admin = "admin-role"
	cfg.Global.Discord.Roles.RoleCache.Moderator = "mod-role"
	cfg.Global.Discord.Roles.RoleCache.Supporter = "supporter-role"
	cfg.Global.Discord.Roles.RoleCache.Patreon = "patreon-role"
	cfg.Global.Discord.Roles.RoleCache.Regular = "regular-role"
	cfg.Global.Discord.Roles.RoleCache.Veteran = "veteran-role"
	cfg.Global.Discord.Roles.RoleCache.Member = "member-role"

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				Roles: []string{"admin-role", "mod-role", "supporter-role", "regular-role", "veteran-role", "member-role"},
			},
		},
	}

	if !CheckAdmin(i) {
		t.Fatal("expected admin role to match")
	}
	if !CheckModerator(i) {
		t.Fatal("expected moderator role to match")
	}
	if !CheckSupporter(i) {
		t.Fatal("expected supporter role to match")
	}
	if !CheckRegular(i) {
		t.Fatal("expected regular role to match")
	}
	if !CheckVeteran(i) {
		t.Fatal("expected veteran role to match")
	}
	if !CheckMember(i) {
		t.Fatal("expected member role to match")
	}
}

func TestCheckSupporterMatchesPatreonRole(t *testing.T) {
	cfg.Global.Discord.Roles.RoleCache.Supporter = "supporter-role"
	cfg.Global.Discord.Roles.RoleCache.Patreon = "patreon-role"

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				Roles: []string{"patreon-role"},
			},
		},
	}

	if !CheckSupporter(i) {
		t.Fatal("expected patreon role to satisfy supporter check")
	}
}

func TestRoleChecksFailWithoutMatchingRole(t *testing.T) {
	cfg.Global.Discord.Roles.RoleCache.Admin = "admin-role"
	cfg.Global.Discord.Roles.RoleCache.Regular = "regular-role"

	i := &discordgo.InteractionCreate{
		Interaction: &discordgo.Interaction{
			Member: &discordgo.Member{
				Roles: []string{"different-role"},
			},
		},
	}

	if CheckAdmin(i) {
		t.Fatal("expected admin role check to fail")
	}
	if CheckRegular(i) {
		t.Fatal("expected regular role check to fail")
	}
}

func TestQueueDiscordMessageQueuesStatusText(t *testing.T) {
	drainCMSChan()
	t.Cleanup(drainCMSChan)

	QueueDiscordMessage("chan-1", "Factorio is online.")

	select {
	case queued := <-CMSChan:
		if queued.Channel != "chan-1" {
			t.Fatalf("expected queued channel chan-1, got %q", queued.Channel)
		}
		if queued.Text != "Factorio is online." {
			t.Fatalf("expected queued text %q, got %q", "Factorio is online.", queued.Text)
		}
	default:
		t.Fatal("expected status text to be queued")
	}
}

func TestSmartWriteDiscordQueuesWhenDiscordOffline(t *testing.T) {
	drainCMSChan()
	t.Cleanup(drainCMSChan)

	DS = nil

	if msg := SmartWriteDiscord("chan-1", "boot message"); msg != nil {
		t.Fatal("expected no direct Discord message while offline")
	}

	select {
	case queued := <-CMSChan:
		if queued.Channel != "chan-1" || queued.Text != "boot message" {
			t.Fatalf("unexpected queued message: %+v", queued)
		}
	default:
		t.Fatal("expected offline send to be queued")
	}
}

func TestSmartWriteDiscordRequeuesRetryableSendError(t *testing.T) {
	drainCMSChan()
	t.Cleanup(drainCMSChan)

	DS = &discordgo.Session{}
	oldSender := discordTextSender
	discordTextSender = func(ch string, text string) (*discordgo.Message, error) {
		return nil, errors.New("websocket is not open")
	}
	t.Cleanup(func() {
		discordTextSender = oldSender
		DS = nil
	})

	if msg := SmartWriteDiscord("chan-1", "retry me"); msg != nil {
		t.Fatal("expected no direct Discord message on retryable error")
	}

	select {
	case queued := <-CMSChan:
		if queued.Channel != "chan-1" || queued.Text != "retry me" {
			t.Fatalf("unexpected queued message: %+v", queued)
		}
	default:
		t.Fatal("expected retryable send error to requeue message")
	}
}

func TestSmartWriteDiscordDoesNotRequeueNonRetryableError(t *testing.T) {
	drainCMSChan()
	t.Cleanup(drainCMSChan)

	DS = &discordgo.Session{}
	oldSender := discordTextSender
	discordTextSender = func(ch string, text string) (*discordgo.Message, error) {
		return nil, errors.New("missing access")
	}
	t.Cleanup(func() {
		discordTextSender = oldSender
		DS = nil
	})

	if msg := SmartWriteDiscord("chan-1", "do not retry"); msg != nil {
		t.Fatal("expected no direct Discord message on failed send")
	}

	select {
	case queued := <-CMSChan:
		t.Fatalf("did not expect non-retryable error to requeue, got %+v", queued)
	default:
	}
}
