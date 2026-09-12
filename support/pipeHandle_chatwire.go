package support

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ChatWire/banlist"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

const chatWireTag = "[CHATWIRE] "

type chatWireEnvelope struct {
	Version int             `json:"v"`
	Kind    string          `json:"kind"`
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Event   string          `json:"event"`
	OK      bool            `json:"ok"`
	Error   string          `json:"error"`
	Data    json.RawMessage `json:"data"`
}

type chatWireOnlinePlayer struct {
	Name       string `json:"name"`
	ScoreTicks int    `json:"score_ticks"`
	TimeTicks  int    `json:"time_ticks"`
	Type       string `json:"type"`
	AFK        string `json:"afk"`
}

type chatWireOnlineData struct {
	Count   int             `json:"count"`
	Players json.RawMessage `json:"players"`
}

func handleChatWire(input *handleData) bool {
	if !strings.HasPrefix(input.line, chatWireTag) {
		return false
	}

	payload, err := fact.DecodeSoftModPayload(strings.TrimSpace(strings.TrimPrefix(input.line, chatWireTag)))
	if err != nil {
		cwlog.DoLogCW("Invalid SoftMod protocol encoding: %v", err)
		return true
	}
	var envelope chatWireEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		cwlog.DoLogCW("Invalid SoftMod protocol message: %v", err)
		return true
	}
	if envelope.Version != fact.SoftModProtocolVersion {
		cwlog.DoLogCW("Unsupported SoftMod protocol version: %d", envelope.Version)
		return true
	}

	switch envelope.Kind {
	case "response":
		handleChatWireResponse(envelope)
	case "event":
		handleChatWireEvent(envelope.Event, envelope.Data)
	default:
		cwlog.DoLogCW("Invalid SoftMod protocol message kind: %q", envelope.Kind)
	}
	return true
}

func handleChatWireResponse(envelope chatWireEnvelope) {
	if !envelope.OK {
		cwlog.DoLogCW("SoftMod request %s (%s) failed: %s", envelope.ID, envelope.Command, envelope.Error)
		return
	}

	switch envelope.Command {
	case "hello":
		var data struct {
			Version string `json:"softmod_version"`
		}
		if json.Unmarshal(envelope.Data, &data) != nil || data.Version == "" {
			cwlog.DoLogCW("SoftMod hello response omitted its version")
			return
		}
		firstHello := glob.SoftModVersion == constants.Unknown
		glob.SoftModVersion = data.Version
		if firstHello {
			cwlog.DoLogCW("Softmod detected: " + data.Version)
			ConfigSoftMod()
			fact.RequestOnlinePlayers()
		}
	case "status":
		handleChatWireStatus(envelope.Data)
	case "online":
		handleChatWireOnline(envelope.Data)
	}
}

func handleChatWireEvent(event string, data json.RawMessage) {
	switch event {
	case "online":
		handleChatWireOnline(data)
	case "player-level":
		var level struct {
			Name  string `json:"name"`
			Level int    `json:"level"`
		}
		if json.Unmarshal(data, &level) == nil && level.Name != "" {
			fact.PlayerLevelSetFromGame(level.Name, level.Level)
		}
	case "registration":
		var registration struct {
			Type string `json:"type"`
			Name string `json:"name"`
			Code string `json:"code"`
		}
		if json.Unmarshal(data, &registration) == nil {
			handlePlayerRegister(preProcessFactorioOutput(fmt.Sprintf("[ACCESS] %s %s %s", registration.Type, registration.Name, registration.Code)))
		}
	case "message", "activity", "audit", "report", "todo", "error", "log":
		var payload struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(data, &payload) != nil {
			return
		}
		dispatchChatWireTextEvent(event, payload.Text)
	}
}

func dispatchChatWireTextEvent(event, text string) {
	tag := map[string]string{
		"message": "[MSG] ", "activity": "[ACT] ", "audit": "[AUDIT] ",
		"report": "[REPORT] ", "todo": "[TODO] ", "error": "[ERROR] ",
	}[event]
	input := preProcessFactorioOutput(tag + text)
	switch event {
	case "message":
		handleSoftModMsg(input)
	case "activity", "todo", "error":
		handleActMsg(input)
	case "audit":
		handleAuditMsg(input)
	case "report":
		handlePlayerReport(input)
	case "log":
		cwlog.DoLogGame(text)
	}
}

func handleChatWireStatus(raw json.RawMessage) {
	var status struct {
		Tick   uint64  `json:"tick"`
		Speed  float64 `json:"speed"`
		Paused bool    `json:"paused"`
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		cwlog.DoLogCW("Invalid SoftMod status response: %v", err)
		return
	}
	fact.RecordGameTick(status.Tick, status.Speed, status.Paused, time.Now())
}

func handleChatWireOnline(raw json.RawMessage) {
	var data chatWireOnlineData
	if err := json.Unmarshal(raw, &data); err != nil {
		cwlog.DoLogCW("Invalid SoftMod online response: %v", err)
		return
	}

	var onlinePlayers []chatWireOnlinePlayer
	playersJSON := strings.TrimSpace(string(data.Players))
	if playersJSON != "" && playersJSON != "null" && playersJSON != "{}" {
		if err := json.Unmarshal(data.Players, &onlinePlayers); err != nil {
			cwlog.DoLogCW("Invalid SoftMod player array: %v", err)
			return
		}
	}
	players := make([]glob.OnlinePlayerData, 0, len(onlinePlayers))
	for _, player := range onlinePlayers {
		if player.Name == "" {
			continue
		}
		fact.UpdateSeen(player.Name)
		banlist.CheckBanList(player.Name, false)
		players = append(players, glob.OnlinePlayerData{
			Name: player.Name, ScoreTicks: player.ScoreTicks, TimeTicks: player.TimeTicks,
			Level: fact.StringToLevel(player.Type), AFK: player.AFK,
		})
	}

	previous := fact.NumPlayersCurrent()
	fact.SetNumPlayers(len(players))
	fact.OnlinePlayersLock.Lock()
	glob.OnlinePlayers = players
	fact.OnlinePlayersLock.Unlock()
	if previous != len(players) {
		fact.UpdateChannelName()
		if len(players) == 0 {
			fact.DoUpdateChannelNameForce()
		}
	}
}
