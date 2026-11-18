package support

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"github.com/bwmarrin/discordgo"
)

func leaderboardMapLabel(name, channel string) string {
	if _, err := strconv.ParseUint(channel, 10, 64); err == nil && channel != "0" {
		return "<#" + channel + ">"
	}
	name = strings.Join(strings.Fields(name), " ")
	if chars := []rune(name); len(chars) > 40 {
		name = string(chars[:39]) + "…"
	}
	return strings.ReplaceAll(name, "@", "@\u200b")
}

const recentResetLimit = 3

func popularMapsEmbed(now time.Time) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title: "Most popular maps", Description: popularMapsDescription(now), Color: glob.COLOR_CYAN,
	}
}

type recentReset struct {
	label string
	at    time.Time
}

func recentResetSummary(now time.Time) string {
	root := cfg.Global.Paths.Folders.ServersRoot
	paths, err := filepath.Glob(filepath.Join(root, cfg.Global.Paths.ChatWirePrefix+"*", constants.CWLocalConfig))
	if err != nil {
		cwlog.DoLogCW("Popular maps: finding servers: %v", err)
		return "Reset history unavailable."
	}
	// Search the retained logs newest day first. Once three maps have been found,
	// finish that day's logs so all resets on it are ranked by their actual time.
	const logLayout = "cw-2-January-2006.log"
	serversByDay := make(map[time.Time][]string)
	partial := false
	for _, path := range paths {
		logDir := filepath.Join(filepath.Dir(path), "audit-log")
		files, err := os.ReadDir(logDir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			cwlog.DoLogCW("Popular maps: listing %s: %v", logDir, err)
			partial = true
			continue
		}
		for _, file := range files {
			day, err := time.Parse(logLayout, file.Name())
			if err == nil && !day.After(now) && !file.IsDir() {
				serversByDay[day] = append(serversByDay[day], path)
			}
		}
	}
	days := make([]time.Time, 0, len(serversByDay))
	for day := range serversByDay {
		days = append(days, day)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].After(days[j]) })
	var lines []string
	seen := make(map[string]bool)
	for _, day := range days {
		var resets []recentReset
		for _, path := range serversByDay[day] {
			if seen[path] {
				continue
			}
			logPath := filepath.Join(filepath.Dir(path), "audit-log", day.Format(logLayout))
			at, err := lastResetInLog(logPath, day, now)
			if err != nil {
				cwlog.DoLogCW("Popular maps: reading %s: %v", logPath, err)
				partial = true
				continue
			}
			if !at.IsZero() {
				seen[path] = true
				resets = append(resets, recentReset{resetMapLabel(path), at})
			}
		}
		sort.SliceStable(resets, func(i, j int) bool { return resets[i].at.After(resets[j].at) })
		for _, reset := range resets[:min(len(resets), recentResetLimit-len(lines))] {
			lines = append(lines, fmt.Sprintf("%d. %s — <t:%d:R>", len(lines)+1, reset.label, reset.at.Unix()))
		}
		if len(lines) == recentResetLimit {
			break
		}
	}
	if partial {
		lines = append(lines, "Some reset history is unavailable.")
	}
	if len(lines) == 0 {
		return "No resets recorded."
	}
	return strings.Join(lines, "\n")
}

func resetMapLabel(path string) string {
	var server struct {
		Name    string
		Channel struct{ ChatChannel string }
	}
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &server)
	}
	if server.Name == "" {
		server.Name = filepath.Base(filepath.Dir(path))
	}
	return leaderboardMapLabel(server.Name, server.Channel.ChatChannel)
}

// The lifecycle logs this event after it has generated and installed the reset save.
// Reading it also covers resets by servers still running an older ChatWire.
var completedResetLine = regexp.MustCompile(`^ *([0-9]{1,2}): *([0-9]{1,2})\. *([0-9]{1,2}): +lifecycle\.go: *[0-9]+: lifecycle: follow-up action executed kind=map-reset elapsed=`)

func lastResetInLog(path string, day, now time.Time) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()
	var latest time.Time
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4096), 4<<20)
	for scanner.Scan() {
		match := completedResetLine.FindSubmatch(scanner.Bytes())
		if match == nil {
			continue
		}
		hour, _ := strconv.Atoi(string(match[1]))
		minute, _ := strconv.Atoi(string(match[2]))
		second, _ := strconv.Atoi(string(match[3]))
		// CW log filenames use UTC dates, but individual lines use the host's local clock.
		for _, boundary := range []time.Time{day, day.Add(24*time.Hour - time.Nanosecond)} {
			y, m, d := boundary.Local().Date()
			at := time.Date(y, m, d, hour, minute, second, 0, time.Local)
			if at.UTC().Truncate(24*time.Hour).Equal(day) && !at.After(now) && at.After(latest) {
				latest = at
			}
		}
	}
	return latest, scanner.Err()
}
