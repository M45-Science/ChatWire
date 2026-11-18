package support

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/util"
	"github.com/bwmarrin/discordgo"
)

// Each server owns its own daily files, avoiding concurrent writers across processes.
type mapMinutes struct {
	Name    string
	Channel string
	Minutes int
}

func popularityDir() string {
	return filepath.Join(cfg.Global.Paths.Folders.ServersRoot, "map-popularity")
}
func popularityFile(day time.Time, callsign string) string {
	return filepath.Join(popularityDir(), day.UTC().Format("2006-01-02"), filepath.Base(callsign)+".json")
}
func recordMapMinutes(now time.Time, count int) error {
	if count <= 0 {
		return nil
	}
	path := popularityFile(now, cfg.Local.Callsign)
	var entry mapMinutes
	data, err := os.ReadFile(path)
	if err == nil {
		if err = json.Unmarshal(data, &entry); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	entry.Name, entry.Channel = cfg.Local.Name, cfg.Local.Channel.ChatChannel
	entry.Minutes += count
	data, err = json.Marshal(entry)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".popularity-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func popularityRanking(now, start time.Time) []mapMinutes {
	totals := map[string]mapMinutes{}
	for day := start; !day.After(now); day = day.AddDate(0, 0, 1) {
		files, err := filepath.Glob(filepath.Join(popularityDir(), day.Format("2006-01-02"), "*.json"))
		if err != nil {
			continue
		}
		for _, path := range files {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var entry mapMinutes
			if json.Unmarshal(data, &entry) != nil || entry.Minutes <= 0 {
				continue
			}
			key := filepath.Base(path)
			total := totals[key]
			total.Minutes += entry.Minutes
			total.Name, total.Channel = entry.Name, entry.Channel
			totals[key] = total
		}
	}
	result := make([]mapMinutes, 0, len(totals))
	for _, entry := range totals {
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Minutes != result[j].Minutes {
			return result[i].Minutes > result[j].Minutes
		}
		return result[i].Name < result[j].Name
	})
	return result
}
func popularMapsDescription(now time.Time) string {
	now = now.UTC()
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	week := day.AddDate(0, 0, -(int(day.Weekday())+6)%7)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var b strings.Builder
	b.WriteString("Combined player hours · UTC calendar periods · updated every 15 minutes\n")
	for _, period := range []struct {
		name  string
		start time.Time
	}{{"Today", day}, {"This week", week}, {"This month", month}} {
		fmt.Fprintf(&b, "\n**%s**\n", period.name)
		rows := popularityRanking(day, period.start)
		if len(rows) == 0 {
			b.WriteString("No activity recorded yet.\n")
			continue
		}
		if len(rows) > 10 {
			rows = rows[:10]
		}
		for i, row := range rows {
			name := strings.NewReplacer("\n", " ", "\r", " ", "@", "@\u200b").Replace(row.Name)
			if row.Channel != "" {
				name = "<#" + row.Channel + ">"
			}
			fmt.Fprintf(&b, "%d. %s — %.1f hours\n", i+1, name, float64(row.Minutes)/60)
		}
	}
	return b.String()
}

type popularityMessage struct{ Channel, ID string }

const popularMapsChannelName = "popular-maps"

// resolvePopularMapsChannel reuses a saved or configured channel. If neither
// exists, it searches the guild before creating one, so a failed state write
// or a restart cannot normally leave duplicate channels.
func resolvePopularMapsChannel(configured, saved, guild string, list func(string) ([]*discordgo.Channel, error), create func(string) (*discordgo.Channel, error)) (string, error) {
	if configured != "" {
		return configured, nil
	}
	if saved != "" {
		return saved, nil
	}
	if guild == "" {
		return "", fmt.Errorf("Discord guild is not configured")
	}
	channels, err := list(guild)
	if err != nil {
		return "", fmt.Errorf("list channels: %w", err)
	}
	for _, channel := range channels {
		if channel.Name == popularMapsChannelName && channel.Type == discordgo.ChannelTypeGuildText {
			return channel.ID, nil
		}
	}
	channel, err := create(guild)
	if err != nil {
		return "", fmt.Errorf("create #%s (requires Manage Channels): %w", popularMapsChannelName, err)
	}
	if channel == nil || channel.ID == "" {
		return "", fmt.Errorf("Discord returned an empty channel")
	}
	return channel.ID, nil
}

func savePopularityMessage(path string, saved popularityMessage) error {
	data, err := json.Marshal(saved)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".message-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func publishPopularMaps(now time.Time) error {
	if disc.DS == nil {
		return nil
	}
	if err := os.MkdirAll(popularityDir(), 0755); err != nil {
		return err
	}
	lock, err := util.AcquireFileLock(filepath.Join(popularityDir(), "report.lock"), cfg.Local.Callsign, 5*time.Second)
	if err != nil {
		return err
	}
	defer lock.Release()
	path := filepath.Join(popularityDir(), "message.json")
	var saved popularityMessage
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, &saved); err != nil {
			return fmt.Errorf("read report state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	channel, err := resolvePopularMapsChannel(cfg.Global.Discord.PopularMapsChannel, saved.Channel, cfg.Global.Discord.Guild,
		func(guild string) ([]*discordgo.Channel, error) { return disc.DS.GuildChannels(guild) },
		func(guild string) (*discordgo.Channel, error) {
			return disc.DS.GuildChannelCreate(guild, popularMapsChannelName, discordgo.ChannelTypeGuildText)
		})
	if err != nil {
		return err
	}
	if channel != saved.Channel {
		saved = popularityMessage{Channel: channel}
		if err := savePopularityMessage(path, saved); err != nil {
			return err
		}
	}

	embed := &discordgo.MessageEmbed{Title: "Most popular maps", Description: popularMapsDescription(now), Color: glob.COLOR_CYAN}
	var msg *discordgo.Message
	if saved.Channel == channel && saved.ID != "" {
		msg, err = disc.DS.ChannelMessageEditEmbed(channel, saved.ID, embed)
		if err != nil {
			var restErr *discordgo.RESTError
			if !errors.As(err, &restErr) || restErr.Response == nil || restErr.Response.StatusCode != http.StatusNotFound {
				return err
			}
			if restErr.Message != nil && restErr.Message.Code == discordgo.ErrCodeUnknownChannel && cfg.Global.Discord.PopularMapsChannel == "" {
				if err := savePopularityMessage(path, popularityMessage{}); err != nil {
					return err
				}
				return fmt.Errorf("popular maps channel was deleted; will recreate it on the next attempt")
			}
			msg, err = disc.DS.ChannelMessageSendEmbed(channel, embed)
			if err != nil {
				return err
			}
		}
	} else {
		msg, err = disc.DS.ChannelMessageSendEmbed(channel, embed)
		if err != nil {
			return err
		}
	}
	if saved.Channel == channel && saved.ID == msg.ID {
		return nil
	}
	return savePopularityMessage(path, popularityMessage{channel, msg.ID})
}
func startPopularMapsLoop() {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		ctx := glob.RuntimeContext()
		var nextReport time.Time
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				if !isIdle() {
					if err := recordMapMinutes(now, fact.NumPlayersCurrent()); err != nil {
						cwlog.DoLogCW("Popular maps: %v", err)
					}
				}
				if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) && disc.DS != nil && !now.Before(nextReport) {
					if err := publishPopularMaps(now); err != nil {
						cwlog.DoLogCW("Popular maps: %v", err)
						nextReport = now.Add(time.Minute)
					} else {
						nextReport = now.Add(15 * time.Minute)
					}
				}
			}
		}
	}()
}
