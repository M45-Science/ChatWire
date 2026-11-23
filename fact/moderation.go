package fact

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/util"
)

var errPlayerLevelConflict = errors.New("player level changed on another map; review the current level and try again")

type nativeModerationAction struct {
	banned   bool
	reason   string
	revision uint64
}

var pendingNativeModeration = struct {
	sync.Mutex
	actions map[string]nativeModerationAction
}{actions: make(map[string]nativeModerationAction)}

func validUnbannedLevel(level int) bool {
	return level == 0 || level == 1 || level == 2 || level == 3 || level == 255
}

// SetPlayerLevelByModerator is shared by Discord and HTTP adapters.
func SetPlayerLevelByModerator(name string, level int, reason, actor string) error {
	if !validUnbannedLevel(level) && level != -1 && level != -255 {
		return errors.New("invalid player level")
	}
	if len(reason) > 500 {
		return errors.New("ban reason is too long")
	}
	if reason == "" {
		reason = "No reason given"
	}
	name = strings.ToLower(strings.TrimSpace(name))
	if level == -1 {
		reason = fmt.Sprintf("%s -- %s %s", reason, actor, time.Now().Format("01-02-2006"))
	}
	expected := playerLevelRevision(name)
	previous, saved, err := commitPlayerLevel(name, playerLevelChange{level: level, reason: reason, expected: &expected})
	if err != nil {
		if errors.Is(err, errPlayerLevelConflict) {
			LoadPlayers(false, false, false)
		}
		return err
	}
	applyModerationToGame(name, previous, saved)
	AutoPromoteFromLevel(name, false, false, previous.Level)
	return nil
}

// ApplyCommunityBan rechecks the Regular exemption against the shared record.
func ApplyCommunityBan(name, reason string) (bool, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	previous, saved, err := commitPlayerLevel(name, playerLevelChange{level: -1, reason: reason, source: moderationCommunity})
	if err != nil || !saved.Banned() {
		return false, err
	}
	applyModerationToGame(name, previous, saved)
	return true, nil
}

func applyAutomaticBan(name, reason string) {
	name = strings.ToLower(strings.TrimSpace(name))
	previous, saved, err := commitPlayerLevel(name, playerLevelChange{level: -1, reason: reason})
	if err != nil {
		cwlog.DoLogCW("Automatic ban for %q could not be saved: %v", name, err)
		return
	}
	applyModerationToGame(name, previous, saved)
}

// RecordNativeModeration accepts an untagged Factorio action. Tagged replies
// go to the reconciliation code and never enter this write path.
func RecordNativeModeration(name string, banned bool, reason string, revision uint64) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if !validModerationName(name) || len(reason) > constants.MaxBanReasonLength {
		return errors.New("invalid native moderation event")
	}
	pendingNativeModeration.Lock()
	defer pendingNativeModeration.Unlock()
	action := nativeModerationAction{banned: banned, reason: reason, revision: revision}
	pendingNativeModeration.actions[name] = action
	return processNativeModeration(name, action)
}

// RetryNativeModeration keeps a native action pending across transient DB errors.
// The original revision stays attached so retries cannot overtake a newer ban.
func RetryNativeModeration() {
	pendingNativeModeration.Lock()
	defer pendingNativeModeration.Unlock()
	for name, action := range pendingNativeModeration.actions {
		if err := processNativeModeration(name, action); err != nil && !errors.Is(err, errPlayerLevelConflict) {
			cwlog.DoLogCW("Native moderation retry for %q: %v", name, err)
		}
		return
	}
}

func processNativeModeration(name string, action nativeModerationAction) error {
	level := -1
	if !action.banned {
		level = 0
		action.reason = ""
	}
	_, saved, err := commitPlayerLevel(name, playerLevelChange{
		level: level, reason: action.reason, expected: &action.revision, source: moderationNative,
	})
	if errors.Is(err, errPlayerLevelConflict) {
		LoadPlayers(false, false, false)
	}
	if err == nil || errors.Is(err, errPlayerLevelConflict) {
		delete(pendingNativeModeration.actions, name)
		rememberBanState(name, &saved, true)
		ReconcilePlayerBans()
	}
	return err
}

func validModerationName(name string) bool {
	return name != "" && len(name) <= constants.MaxNameLength && !strings.ContainsAny(name, " \t\r\n\x00")
}

func playerLevelRevision(name string) uint64 {
	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()
	if player := glob.PlayerList[name]; player != nil {
		return player.LevelRevision
	}
	return 0
}

type moderationSource uint8

const (
	moderationCommand moderationSource = iota
	moderationNative
	moderationCommunity
	playerRegistration
)

type playerLevelChange struct {
	level     int
	reason    string
	discordID string
	expected  *uint64
	source    moderationSource
}

func commitPlayerLevel(name string, change playerLevelChange) (previous, saved glob.PlayerData, err error) {
	if !validModerationName(name) {
		return previous, saved, errors.New("valid player name required")
	}
	if len(change.reason) > constants.MaxBanReasonLength {
		return previous, saved, errors.New("ban reason is too long")
	}
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	path := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
	lock, err := util.AcquireFileLock(path+".lock", cfg.Local.Callsign, playerDBLockTimeout)
	if err != nil {
		return previous, saved, err
	}
	defer lock.Release()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return previous, saved, err
	}
	players, _, err := decodePlayerDatabase(data)
	if err != nil {
		return previous, saved, err
	}
	if stored := players[name]; stored != nil {
		previous = *stored
	}
	saved = previous
	currentRevision := previous.LevelRevision
	if change.source == moderationNative {
		currentRevision = previous.BanRevision
	}
	if change.expected != nil && *change.expected != currentRevision {
		return previous, saved, errPlayerLevelConflict
	}
	if change.source == moderationCommunity && previous.Level >= 2 {
		return previous, saved, nil
	}
	if change.source == playerRegistration {
		if previous.Banned() {
			return previous, saved, errors.New("banned players cannot register")
		}
		if previous.ID != "" && previous.ID != "0" && previous.ID != change.discordID {
			return previous, saved, errors.New("player is already linked to another Discord account")
		}
	}
	// An unban on an already-unbanned record must preserve its current rank.
	level := change.level
	updateReason := change.source == moderationNative && level == -1 && change.reason != "" && change.reason != previous.BanReason
	updateID := change.source == playerRegistration && change.discordID != previous.ID
	if (previous.Level == level && !updateReason && !updateID) ||
		(change.source == moderationNative && level == 0 && previous.Level >= 0) {
		return previous, saved, nil
	}
	saved.Level = level
	if level == -1 {
		if change.reason != "" {
			saved.BanReason = change.reason
		}
		saved.Creation, saved.LastSeen = 0, 0
	} else {
		saved.LastSeen = compactNow()
		if level <= -254 {
			saved.ID = "0"
		}
	}
	if change.source == playerRegistration {
		saved.ID = change.discordID
		if saved.Creation == 0 {
			saved.Creation = saved.LastSeen
		}
	}
	if !advanceLevelRevision(&previous, &saved) {
		return previous, previous, errors.New("player level revision exhausted")
	}
	players[name] = &saved
	encoded, err := encodePlayerDatabase(players, cfg.Global.Paths.DataFiles.DBFormat)
	if err != nil {
		return previous, previous, err
	}
	if err := util.WriteBytesAtomic(path, encoded, 0644); err != nil {
		return previous, previous, err
	}

	glob.PlayerListLock.Lock()
	local := glob.PlayerList[name]
	if local == nil {
		copy := saved
		copy.Name = name
		glob.PlayerList[name] = &copy
		syncPlayerAdminAccess(name, 0, saved.Level)
		WhitelistPlayer(name, saved.Level)
	} else {
		applyCommittedPlayerLevel(name, local, &saved)
		if change.source == playerRegistration {
			local.ID = saved.ID
			local.Creation, local.LastSeen = saved.Creation, saved.LastSeen
		}
	}
	glob.PlayerListLock.Unlock()
	rememberBanState(name, &saved, false)
	return previous, saved, nil
}

func applyCommittedPlayerLevel(name string, local, saved *glob.PlayerData) {
	previousLevel := local.Level
	local.Level, local.BanReason = saved.Level, saved.BanReason
	local.LevelRevision, local.BanRevision, local.BanRetained = saved.LevelRevision, saved.BanRevision, saved.BanRetained
	local.AlreadyBanned = saved.Banned()
	if saved.Level == -1 {
		local.Creation, local.LastSeen = saved.Creation, saved.LastSeen
	} else if saved.Level <= -254 {
		local.ID = "0"
	}
	syncPlayerAdminAccess(name, previousLevel, saved.Level)
	WhitelistPlayer(name, saved.Level)
}

func applyModerationToGame(name string, previous, saved glob.PlayerData) {
	rememberBanState(name, &saved, false)
	if TaggedBanSyncEnabled() {
		ReconcilePlayerBans()
	} else if saved.Banned() && saved.Level == -1 {
		WriteBan(name, saved.BanReason)
	} else if previous.Banned() && !saved.Banned() && saved.Level >= 0 {
		WriteUnban(name)
	}
}
