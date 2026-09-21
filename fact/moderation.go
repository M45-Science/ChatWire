package fact

import (
	"ChatWire/glob"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SetPlayerLevelByModerator is shared by Discord and HTTP adapters.
func SetPlayerLevelByModerator(name string, level int, reason, actor string) error {
	switch level {
	case -255, -1, 0, 1, 2, 3, 255:
	default:
		return errors.New("invalid player level")
	}
	if len(reason) > 500 {
		return errors.New("ban reason is too long")
	}
	if reason == "" {
		reason = "No reason given"
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return errors.New("player name required")
	}
	old := PlayerLevelGet(name, false)
	glob.PlayerListLock.RLock()
	playerName := ""
	if p := glob.PlayerList[name]; p != nil {
		playerName = p.Name
	}
	glob.PlayerListLock.RUnlock()
	if playerName == "" {
		return errors.New("player not found")
	}
	if level >= 0 && old == -1 {
		WriteUnban(name)
	}
	if level == -1 && old != -1 {
		reason = fmt.Sprintf("%s -- %s %s", reason, actor, time.Now().Format("01-02-2006"))
		WriteBan(name, reason)
		glob.PlayerListLock.Lock()
		if p := glob.PlayerList[name]; p != nil {
			p.BanReason = reason
		}
		glob.PlayerListLock.Unlock()
	}
	PlayerLevelSet(playerName, level, true)
	AutoPromoteFromLevel(playerName, false, false, old)
	WritePlayers()
	return nil
}
