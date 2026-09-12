package user

import (
	"fmt"
	"testing"

	"ChatWire/cfg"
	"ChatWire/glob"
)

func TestPlayerListEmbedsPreservePlayerDetails(t *testing.T) {
	oldLocal := cfg.Local
	t.Cleanup(func() { cfg.Local = oldLocal })
	cfg.Local.Callsign = "a"
	cfg.Local.Name = "test"

	embeds := playerListEmbeds([]glob.OnlinePlayerData{{
		Name: "Alice", ScoreTicks: 7200, TimeTicks: 3600, Level: 2, AFK: "3m",
	}})
	if len(embeds) != 1 || len(embeds[0].Fields) != 1 {
		t.Fatalf("embeds = %#v", embeds)
	}
	field := embeds[0].Fields[0]
	if field.Name != "Alice • Regular" || field.Value != "Score **2.0 h**\nOnline **1m0s**\nAFK **3m**" {
		t.Fatalf("field = %#v", field)
	}
}

func TestPlayerListEmbedsPaginateAtDiscordFieldLimit(t *testing.T) {
	players := make([]glob.OnlinePlayerData, 26)
	for i := range players {
		players[i].Name = fmt.Sprintf("player-%d", i)
	}
	embeds := playerListEmbeds(players)
	if len(embeds) != 2 || len(embeds[0].Fields) != 25 || len(embeds[1].Fields) != 1 {
		t.Fatalf("field counts = %d/%d across %d embeds", len(embeds[0].Fields), len(embeds[1].Fields), len(embeds))
	}
}
