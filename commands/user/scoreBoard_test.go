package user

import (
	"fmt"
	"strings"
	"testing"
)

func TestFormatScoreboardIncludesSinglePlayer(t *testing.T) {
	got := formatScoreboard([]scoreData{{Name: "alice", Score: 60}})
	if !strings.Contains(got, "# 1:") || !strings.Contains(got, "alice") {
		t.Fatalf("single-player scoreboard omitted its player: %q", got)
	}
}

func TestFormatScoreboardLimitsResults(t *testing.T) {
	scores := make([]scoreData, 41)
	for i := range scores {
		scores[i] = scoreData{Name: fmt.Sprintf("player-%d", i), Score: int64(i)}
	}

	got := formatScoreboard(scores)
	if count := strings.Count(got, "\n"); count != 40 {
		t.Fatalf("scoreboard contains %d entries, want 40", count)
	}
	if strings.Contains(got, "player-40") {
		t.Fatal("scoreboard included a player beyond the 40-player limit")
	}
}
