package disc

import (
	"testing"

	"ChatWire/cfg"

	"github.com/bwmarrin/discordgo"
)

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
