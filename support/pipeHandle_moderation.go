package support

import (
	"fmt"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

func handleActMsg(input *handleData) bool {
	/******************
	 * ACT AREA
	 * Used for logs, and to attempt to warn of potential griefing
	 ******************/

	if strings.HasPrefix(input.line, "[ERROR]") {
		cwlog.DoLogCW(input.line)
		return true

	} else if strings.HasPrefix(input.line, "[ACT]") || strings.HasPrefix(input.line, "[TODO]") {
		cwlog.DoLogGame(input.line)
		if input.wordListLen > 2 {
			pname, action := input.wordList[1], input.wordList[2]
			if pname != "" {
				fact.UpdateSeen(pname)
				if input.wordList[0] == "[ACT]" {
					detail := strings.TrimSpace(strings.Join(input.wordList[3:], " "))
					scoreActivity(pname, action, detail)
				}
			}
		}
		return true
	}

	return false
}

// Score only explicit actions. Planner rectangles accompany entity-level
// records and cancellations never add suspicion or trigger an alert.
func scoreActivity(player, action, detail string) {
	var delta int64
	switch action {
	case "placed-ghost":
		delta = -2
	case "mined-ghost", "placed":
		delta = -1
	case "mined":
		delta = 1
	case "decon":
		if detail == "" || strings.HasPrefix(detail, "[") {
			return
		}
		delta = 2
	case "undecon":
	default:
		return
	}
	glob.PlayerListLock.Lock()
	p := glob.PlayerList[strings.ToLower(player)]
	if p == nil || p.Name == "" {
		glob.PlayerListLock.Unlock()
		return
	}
	if p.Level >= 2 {
		p.SusScore = 0
		glob.PlayerListLock.Unlock()
		return
	}
	now := time.Now()
	mapActivityAlerts.mu.Lock()
	session := mapActivityAlerts.sessions[strings.ToLower(player)]
	qualified, context := session.warningScore(now, action, delta, p)
	mapActivityAlerts.mu.Unlock()
	glob.PlayerListLock.Unlock()
	if !cfg.Global.Options.ShutupSusWarn {
		mapActivityAlerts.record(now, player, action, detail, qualified, context)
	}
}

func (s *activitySession) warningScore(now time.Time, action string, delta int64, p *glob.PlayerData) (bool, string) {
	bonus := s.observe(now, action)
	p.SusScore += delta
	thresh := int64(constants.SusWarningThresh)
	if p.Level > 0 {
		thresh += 1000
	}
	qualified := delta > 0 && p.SusScore+bonus > thresh
	if qualified {
		extraReview := p.SusScore <= thresh && bonus > 0
		p.SusScore = 0
		if bonus > 0 {
			s.bonusUsed = true
		}
		if extraReview {
			return true, fmt.Sprintf("Since joining %s ago: %d mined, %d placed.", now.Sub(s.joined).Round(time.Second), s.mined, s.placed)
		}
	}
	return qualified, ""
}

func startActivityAlertLoop() {
	go func() {
		ctx := glob.RuntimeContext()
		for waitForContext(ctx, time.Second) {
			for _, alert := range mapActivityAlerts.due(time.Now()) {
				glob.PlayerListLock.RLock()
				p := glob.PlayerList[strings.ToLower(alert.player)]
				eligible := p != nil && p.Level >= 0 && p.Level < 2
				glob.PlayerListLock.RUnlock()
				if eligible && !cfg.Global.Options.ShutupSusWarn {
					reportActivityAlert(alert)
				}
			}
		}
	}()
}

func reportActivityAlert(alert activityAlert) {
	message := alert.message
	serverTag := fmt.Sprintf("[%s-%s]\n", cfg.Local.Callsign, cfg.Local.Name)
	if cfg.Local.Channel.ChatChannel != "" {
		serverTag = "<#" + cfg.Local.Channel.ChatChannel + "> " + serverTag
	}
	// Only the configured alert role should be able to notify Discord users.
	text := serverTag + strings.ReplaceAll(message, "@", "@\u200b")
	if url := cfg.GetGameLogURL(); url != "" {
		text += "\nLog: " + url
	}
	reportChannel := cfg.Global.Discord.ReportChannel
	if reportChannel != "" {
		reportText := text
		if role := cfg.Global.Discord.SusPingRole; alert.pingRole && role != "" {
			reportText += "\n<@&" + role + ">"
		}
		fact.LogCMS(reportChannel, reportText)
	} else {
		cwlog.DoLogCW("%s", text)
	}
	// A map-channel copy must not ping the same role a second time.
	if mapChannel := cfg.Local.Channel.ChatChannel; !alert.reportOnly && mapChannel != "" && mapChannel != reportChannel {
		fact.CMS(mapChannel, text)
	}
	if alert.gameMessage != "" {
		fact.FactChat("%s", alert.gameMessage)
	}
}

func handleBan(input *handleData) bool {
	/******************
	 * BAN
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[BAN]") {

		glob.PlayerListWriteLock.Lock()
		defer glob.PlayerListWriteLock.Unlock()

		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			trustname := input.noDatestampList[1]
			fact.SetLastBan(trustname)

			if strings.Contains(input.noDatestamp, "was banned by") {
				if reason, ok := banReasonFromLine(input.noDatestamp); ok {
					fact.PlayerSetBanReason(trustname, reason, false)
				} else {
					fact.PlayerLevelSet(trustname, -1, false)
				}
			}

			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.CurrentGametime(), strings.Join(input.noDatestampList[1:], " ")))
			fact.RequestOnlinePlayers()
		}
		return true
	}
	return false
}

func banReasonFromLine(line string) (string, bool) {
	if _, reason, ok := strings.Cut(line, "Reason:"); ok {
		return strings.TrimSpace(reason), true
	}
	return "", false
}

func handleUnBan(input *handleData) bool {
	/******************
	 * UNBAN
	 ******************/
	if strings.HasPrefix(input.noDatestamp, "[UNBANNED]") {
		cwlog.DoLogGame(input.noDatestamp)

		if input.noDatestampListLen > 1 {
			trustname := input.noDatestampList[1]
			fact.SetLastBan("")

			if strings.Contains(input.noDatestamp, "was unbanned by") {
				if fact.PlayerLevelGet(trustname, true) < 0 {
					fact.PlayerLevelSet(trustname, 0, false)
				}
			}

			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, fmt.Sprintf("`%v` %s", fact.CurrentGametime(), strings.Join(input.noDatestampList[1:], " ")))
		}
		return true
	}
	return false
}
