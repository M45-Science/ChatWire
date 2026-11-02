package moderator

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

func TestSetScheduleAcceptsLongResetIntervals(t *testing.T) {
	for _, months := range []int{3, 6} {
		t.Run(strconv.Itoa(months)+" months", func(t *testing.T) {
			oldLocal, oldSession, oldRunning := cfg.Local, disc.DS, fact.FactIsRunning
			t.Cleanup(func() {
				cfg.Local, disc.DS, fact.FactIsRunning = oldLocal, oldSession, oldRunning
			})
			t.Chdir(t.TempDir())
			disc.DS, fact.FactIsRunning = nil, false
			cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
			cfg.Local.Options.NextReset = time.Time{}
			cfg.Local.Options.ResetHour = 0
			cfg.Local.Channel.ChatChannel = ""

			interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				Type: discordgo.InteractionApplicationCommand,
				Data: discordgo.ApplicationCommandInteractionData{
					Options: []*discordgo.ApplicationCommandInteractionDataOption{{
						Name: "interval-months", Type: discordgo.ApplicationCommandOptionInteger, Value: float64(months),
					}},
				},
			}}
			earliest := time.Now().UTC().AddDate(0, months, 0).Round(time.Second)
			SetSchedule(nil, interaction)
			latest := time.Now().UTC().AddDate(0, months, 0).Round(time.Second)
			if cfg.Local.Options.ResetInterval.Months != months {
				t.Fatalf("interval = %+v; want %d months", cfg.Local.Options.ResetInterval, months)
			}
			if date := cfg.Local.Options.NextReset; date.Before(earliest) || date.After(latest) {
				t.Fatalf("next reset = %v; want between %v and %v", date, earliest, latest)
			}
			data, err := os.ReadFile(constants.CWLocalConfig)
			if err != nil {
				t.Fatal(err)
			}
			var stored struct {
				Options struct {
					ResetInterval cfg.ResetInterval
					NextReset     time.Time
				}
			}
			if err := json.Unmarshal(data, &stored); err != nil {
				t.Fatal(err)
			}
			if stored.Options.ResetInterval.Months != months || !stored.Options.NextReset.Equal(cfg.Local.Options.NextReset) {
				t.Fatalf("stored schedule = %+v; want %d months and %v", stored.Options, months, cfg.Local.Options.NextReset)
			}
		})
	}
}
