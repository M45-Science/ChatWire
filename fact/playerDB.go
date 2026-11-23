package fact

import (
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/util"
	"ChatWire/watcher"
)

var (
	playerListDirtyCh = make(chan struct{}, 1)
	playerChanges     = struct {
		sync.Mutex
		next     uint64
		versions map[string]playerChangeVersions
	}{versions: make(map[string]playerChangeVersions)}
)

type playerChangeVersions struct {
	record uint64
	stats  uint64
}

type playerChange struct {
	player   *glob.PlayerData
	versions playerChangeVersions
}

const playerDBLockTimeout = 5 * time.Second

// PlayerRankCounts counts each known Factorio player once at their current rank.
func PlayerRankCounts() map[string]int {
	counts := make(map[string]int)
	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()
	for _, player := range glob.PlayerList {
		if player != nil {
			counts[LevelToString(player.Level)]++
		}
	}
	return counts
}

func PlayerListDirtySignal() <-chan struct{} {
	return playerListDirtyCh
}

/* Local use only */
func compactNow() int64 {
	t := time.Now()
	return (t.Unix() - constants.SeenEpoch) / constants.SeenDivisor
}

func ExpandTime(input int64) time.Time {
	newTime := (input * constants.SeenDivisor) + constants.SeenEpoch
	out := time.Unix(newTime, 0)
	return out
}

/* Screw fsnotify */
func WatchDatabaseFile() {
	filePath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile

	watcher.Watch(filePath, 5*time.Second, glob.RuntimeContext(), func() {
		time.Sleep(time.Second)
		setPlayerListUpdated()
	})
}

/* Set DB as updated */
func setPlayerListUpdated() {
	glob.PlayerListUpdatedLock.Lock()
	glob.PlayerListUpdated = true
	glob.PlayerListUpdatedLock.Unlock()
}

func markPlayerDirty(pname string, statsOnly bool) {
	pname = strings.ToLower(strings.TrimSpace(pname))
	if pname == "" {
		return
	}
	playerChanges.Lock()
	playerChanges.next++
	versions := playerChanges.versions[pname]
	if statsOnly {
		versions.stats = playerChanges.next
	} else {
		versions.record = playerChanges.next
	}
	playerChanges.versions[pname] = versions
	playerChanges.Unlock()
}

func playerRecordChangePending(pname string) bool {
	playerChanges.Lock()
	pending := playerChanges.versions[strings.ToLower(pname)].record != 0
	playerChanges.Unlock()
	return pending
}

func discardPlayerRecordChange(name string) {
	playerChanges.Lock()
	defer playerChanges.Unlock()
	versions := playerChanges.versions[name]
	if versions.stats == 0 {
		delete(playerChanges.versions, name)
		return
	}
	versions.record = 0
	playerChanges.versions[name] = versions
}

func signalPlayerListDirty() {
	glob.PlayerListDirtyLock.Lock()
	glob.PlayerListDirty = true
	glob.PlayerListDirtyLock.Unlock()
	select {
	case playerListDirtyCh <- struct{}{}:
	default:
	}
}

/* Mark a player record as dirty. */
func SetPlayerListDirty(pname string) {
	markPlayerDirty(pname, false)
	signalPlayerListDirty()
}

// SetPlayerStatsDirty marks player stats as updated (LastSeen / Minutes).
// These changes are saved on a slower cadence than full DB changes.
func SetPlayerStatsDirty(pname string) {
	markPlayerDirty(pname, true)
	glob.PlayerStatsDirtyLock.Lock()
	glob.PlayerStatsDirty = true
	glob.PlayerStatsDirtyLock.Unlock()
}

func PlayerSetBanReason(pname string, reason string, doban bool) bool {

	if pname == "" || len(pname) > constants.MaxNameLength || len(reason) > constants.MaxBanReasonLength {
		return false
	}

	pname = strings.ToLower(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {

		if doban && !glob.PlayerList[pname].AlreadyBanned {
			WriteBan(pname, reason)
		}

		glob.PlayerList[pname].Level = -1
		if reason != "" {
			glob.PlayerList[pname].BanReason = reason
		}
		glob.PlayerList[pname].LastSeen = 0
		glob.PlayerList[pname].Creation = 0
		glob.PlayerList[pname].AlreadyBanned = true

		SetPlayerListDirty(pname)
		return true
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:          pname,
		Level:         -1,
		ID:            "",
		BanReason:     reason,
		AlreadyBanned: true,
		LastSeen:      compactNow(),
		Creation:      compactNow(),
	}
	glob.PlayerList[pname] = &newplayer

	if doban {
		WriteBan(pname, reason)
	}

	SetPlayerListDirty(pname)
	return false
}

// RegisterPlayer persists the Discord linkage and its rank before granting roles.
func RegisterPlayer(name, discordID string, level int) error {
	name = strings.ToLower(strings.TrimSpace(name))
	if discordID == "" || !validUnbannedLevel(level) {
		return errors.New("valid Discord ID and player level required")
	}
	expected := playerLevelRevision(name)
	_, _, err := commitPlayerLevel(name, playerLevelChange{
		level: level, discordID: discordID, expected: &expected, source: playerRegistration,
	})
	if errors.Is(err, errPlayerLevelConflict) {
		LoadPlayers(false, false, false)
	}
	return err
}

/* Saw player (low priority) */
func UpdateSeen(pname string) {
	if pname == "" || len(pname) > constants.MaxNameLength {
		return
	}
	pname = strings.ToLower(pname)
	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {
		glob.PlayerList[pname].LastSeen = compactNow()

		SetPlayerStatsDirty(pname)
		return
	}
}

/* Set player level, add to db if not found */
func PlayerLevelSet(pname string, level int, modifyOnly bool) bool {
	return playerLevelSet(pname, level, modifyOnly, true, nil)
}

// PlayerLevelSetFromGame records an in-game rank change. Moderation has its own
// event, and legacy SoftMod may still echo an unchanged rank assignment.
func PlayerLevelSetFromGame(pname string, level int) bool {
	return PlayerLevelSetFromGameAtRevision(pname, level, nil)
}

func PlayerLevelSetFromGameAtRevision(pname string, level int, expected *uint64) bool {
	if !validUnbannedLevel(level) {
		return false
	}
	return playerLevelSet(pname, level, false, false, expected)
}

func playerLevelSet(pname string, level int, modifyOnly, touchUnchanged bool, expected *uint64) bool {
	if pname == "" || len(pname) > constants.MaxNameLength {
		return false
	}

	pname = strings.ToLower(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {
		if (!touchUnchanged && glob.PlayerList[pname].Level < 0) ||
			(expected != nil && *expected != glob.PlayerList[pname].LevelRevision) {
			return false
		}
		if glob.PlayerList[pname].Level != level {
			glob.PlayerList[pname].LastSeen = compactNow()
			syncPlayerAdminAccess(pname, glob.PlayerList[pname].Level, level)
			glob.PlayerList[pname].Level = level
			SetPlayerListDirty(pname)
			WhitelistPlayer(pname, level)
		} else if touchUnchanged {
			glob.PlayerList[pname].LastSeen = compactNow()
			SetPlayerStatsDirty(pname)
		}

		/* Delete discord id upon delete */
		if level <= -255 {
			glob.PlayerList[pname].ID = "0"
		}
		return true
	}

	if modifyOnly || (expected != nil && *expected != 0) {
		return false
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    level,
		LastSeen: compactNow(),
		Creation: compactNow(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty(pname)
	syncPlayerAdminAccess(pname, 0, level)
	WhitelistPlayer(pname, level)

	return false
}

/*************************************************
 * Expects locked db, only used for LoadPlayers()
 *************************************************/
func addPlayer(iname string, level int, id string, creation int64, seen int64, reason string, susScore int64, mins int64, doBan, doUnban bool) bool {
	if iname == "" || len(iname) > constants.MaxNameLength {
		return false
	}

	didBan := false
	pname := strings.ToLower(iname)

	if glob.PlayerList[pname] != nil {
		if level <= -254 { //Delete
			/*Clear discord ID on delete*/
			glob.PlayerList[pname].ID = "0"
		} else if level == -1 && glob.PlayerList[pname].Level >= 0 && doBan { //Banned
			WriteBan(pname, reason)
			didBan = true
		} else if level >= 0 && glob.PlayerList[pname].Level == -1 && doUnban { //Unbanned
			WriteUnban(pname)
		}

		if level != glob.PlayerList[pname].Level {
			syncPlayerAdminAccess(pname, glob.PlayerList[pname].Level, level)
			glob.PlayerList[pname].Level = level
			WhitelistPlayer(pname, level)
		}
		if creation > 0 { //Add creation date
			glob.PlayerList[pname].Creation = creation
		}
		if seen > glob.PlayerList[pname].LastSeen { //Update last seen
			glob.PlayerList[pname].LastSeen = seen
			WhitelistPlayer(pname, level)
		}
		if level >= 0 && id != "" { //Registered, don't keep for banned
			glob.PlayerList[pname].ID = id
		}
		//Don't keep sus score for regulars, admins and banned.
		if level < 2 && level >= 0 && susScore != 0 {
			glob.PlayerList[pname].SusScore = susScore
		}
		//Don't keep playtime for banned
		if level >= 0 && mins > 0 && mins > glob.PlayerList[pname].Minutes {
			glob.PlayerList[pname].Minutes = mins
		}
		return didBan
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{
		Name:      pname,
		Level:     level,
		ID:        id,
		BanReason: reason,
		LastSeen:  seen,
		Creation:  creation,
		Minutes:   mins,
		SusScore:  susScore,
	}

	glob.PlayerList[pname] = &newplayer

	if level == -1 && doBan {
		WriteBan(pname, reason)
		didBan = true
	}
	syncPlayerAdminAccess(pname, 0, level)
	WhitelistPlayer(pname, level)

	return didBan
}

/* Get player level, add to db if not found */
func PlayerLevelGet(pname string, modifyOnly bool) int {
	if pname == "" || len(pname) > constants.MaxNameLength {
		return 0
	}

	pname = strings.ToLower(pname)
	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {

		/* Found in list */
		glob.PlayerList[pname].LastSeen = compactNow()
		level := glob.PlayerList[pname].Level
		SetPlayerStatsDirty(pname)
		return level
	}

	if modifyOnly {
		return 0
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    0,
		ID:       "",
		LastSeen: compactNow(),
		Creation: compactNow(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty(pname)
	return 0
}

func snapshotPlayerChanges() map[string]playerChange {
	glob.PlayerListLock.RLock()
	playerChanges.Lock()
	changes := make(map[string]playerChange, len(playerChanges.versions))
	for pname, versions := range playerChanges.versions {
		change := playerChange{versions: versions}
		if player := glob.PlayerList[pname]; player != nil {
			copy := *player
			change.player = &copy
		}
		changes[pname] = change
	}
	playerChanges.Unlock()
	glob.PlayerListLock.RUnlock()

	return changes
}

func completePlayerChanges(changes map[string]playerChange) {
	playerChanges.Lock()
	defer playerChanges.Unlock()
	for pname, change := range changes {
		pending := playerChanges.versions[pname]
		if pending.record == change.versions.record {
			pending.record = 0
		}
		if pending.stats == change.versions.stats {
			pending.stats = 0
		}
		if pending == (playerChangeVersions{}) {
			delete(playerChanges.versions, pname)
		} else {
			playerChanges.versions[pname] = pending
		}
	}
}

func mergePlayerChanges(players map[string]*glob.PlayerData, changes map[string]playerChange) map[string]bool {
	conflicts := make(map[string]bool)
	for pname, change := range changes {
		player := change.player
		previous := players[pname]
		if change.versions.record == 0 {
			// Playtime and last-seen updates must preserve bans and deletions
			// already written by another instance.
			if player != nil && previous != nil && previous.Level >= 0 {
				previous.LastSeen = max(previous.LastSeen, player.LastSeen)
				previous.Minutes = max(previous.Minutes, player.Minutes)
			}
			continue
		}
		if player == nil {
			if previous == nil || previous.LevelRevision == 0 {
				delete(players, pname)
			}
			continue
		}
		if previous == nil {
			previous = &glob.PlayerData{}
		}
		next := *player
		if !advanceLevelRevision(previous, &next) {
			conflicts[pname] = true
			continue
		}
		next.LastToolHint = max(previous.LastToolHint, next.LastToolHint)
		players[pname] = &next
	}
	return conflicts
}

func advanceLevelRevision(previous, next *glob.PlayerData) bool {
	if next.LevelRevision != previous.LevelRevision {
		return false
	}
	next.BanRevision = previous.BanRevision
	next.BanRetained = next.Level <= -254 && previous.Banned()
	if next.Level == previous.Level && next.BanReason == previous.BanReason && next.ID == previous.ID {
		return true
	}
	if previous.LevelRevision == math.MaxUint64 {
		return false
	}
	next.LevelRevision++
	if next.Banned() != previous.Banned() || next.Banned() && next.BanReason != previous.BanReason {
		next.BanRevision = next.LevelRevision
	}
	return true
}

/* Load database */
func LoadPlayers(bootMode, minimize, clearBans bool) bool {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	didBan := false
	dbPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile

	filedata, err := os.ReadFile(dbPath)
	if err != nil {
		cwlog.DoLogCW("Couldn't read db file, path: %v", dbPath)
		return false
	}

	if filedata != nil {

		var tempData map[string]*glob.PlayerData
		tempData, _, err = decodePlayerDatabase(filedata)
		if err != nil {
			cwlog.DoLogCW("LoadPlayers: " + err.Error())
			return false
		}

		banCount := 0
		doBan := true
		legacySync := !bootMode && !minimize && !clearBans && !TaggedBanSyncEnabled()
		changedNames := make(map[string]struct{})
		levelChanges := make(map[string]int)
		//Add name back in, makes db file smaller
		glob.PlayerListLock.Lock()
		var removed int

		for pname := range tempData {
			stored := tempData[pname]
			if !minimize && !clearBans {
				rememberBanState(pname, stored, false)
			}
			previousLevel := 0
			previousPlayer := glob.PlayerList[pname]
			if !minimize && !clearBans && playerRecordChangePending(pname) {
				if stored.LevelRevision == 0 || (previousPlayer != nil && stored.LevelRevision <= previousPlayer.LevelRevision) {
					continue
				}
				discardPlayerRecordChange(pname)
			}
			if previousPlayer != nil {
				previousLevel = previousPlayer.Level
			}

			if clearBans {
				if tempData[pname].Level < 0 {
					removed++
					if stored.LevelRevision > 0 || stored.Level == -1 {
						glob.PlayerList[pname] = &glob.PlayerData{Name: pname, LevelRevision: stored.LevelRevision, BanRevision: stored.BanRevision}
						changedNames[pname] = struct{}{}
						continue
					}
					delete(tempData, pname)
					delete(glob.PlayerList, pname)
					changedNames[pname] = struct{}{}
					continue
				}
			}
			//DB cleaning
			if minimize {
				//Get rid of new/deleted
				if (tempData[pname].Level == 0 || tempData[pname].Level == -255) && stored.LevelRevision == 0 {
					removed++
					delete(tempData, pname)
					delete(glob.PlayerList, pname)
					changedNames[pname] = struct{}{}
					continue
				}
				changedNames[pname] = struct{}{}
				//Delete unneeded data from member/reg/moderator
				if tempData[pname].Level > 0 {
					tempData[pname].SusScore = 0
					tempData[pname].BanReason = ""
					tempData[pname].SpamScore = 0
				}
				//Check discord id, fixed if needed.
				ID, err := strconv.ParseUint(tempData[pname].ID, 10, 64)
				//There are some old DBs that had ban reasons in the ID field, fix them.
				if ID == 0 || err != nil {
					tempData[pname].BanReason = tempData[pname].ID
					tempData[pname].ID = ""
				}
				//Delete id "0"
				if tempData[pname].ID == "0" {
					tempData[pname].ID = ""
				}
			}

			if banCount > 5 {
				doBan = false
			}
			//Autopromote to veteran
			if tempData[pname].Level == 2 && tempData[pname].Minutes > constants.VeteranThresh {
				tempData[pname].Level = 3
				changedNames[pname] = struct{}{}
			}
			if !bootMode && !minimize && !clearBans && (previousPlayer == nil || previousLevel != tempData[pname].Level) {
				levelChanges[pname] = previousLevel
			}
			didBan = addPlayer(pname, stored.Level, stored.ID, stored.Creation, stored.LastSeen, stored.BanReason, stored.SusScore, stored.Minutes, legacySync && doBan, legacySync)
			if loaded := glob.PlayerList[strings.ToLower(pname)]; loaded != nil {
				loaded.LevelRevision, loaded.BanRevision, loaded.BanRetained = stored.LevelRevision, stored.BanRevision, stored.BanRetained
				loaded.BanReason = stored.BanReason
				loaded.AlreadyBanned = stored.Banned()
				if stored.Level >= 0 {
					loaded.ID = stored.ID
				}
				loaded.LastToolHint = max(loaded.LastToolHint, stored.LastToolHint)
			}
			if didBan {
				banCount++
			}
		}
		if removed > 0 {
			cwlog.DoLogCW("Removed: %v entries.\n", removed)
		}
		glob.PlayerListLock.Unlock()
		for pname, previousLevel := range levelChanges {
			syncOnlinePlayerLevel(pname, previousLevel, tempData[pname].Level)
		}
		for pname := range changedNames {
			SetPlayerListDirty(pname)
		}
		ReconcilePlayerBans()
	}
	return true
}

/* Save database */
func WritePlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	finalPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
	changes := snapshotPlayerChanges()
	if len(changes) == 0 {
		return
	}

	lock, err := util.AcquireFileLock(finalPath+".lock", cfg.Local.Callsign, playerDBLockTimeout)
	if err != nil {
		cwlog.DoLogCW("WritePlayers: " + err.Error())
		signalPlayerListDirty()
		return
	}
	defer func() {
		if err := lock.Release(); err != nil {
			cwlog.DoLogCW("WritePlayers: release database lock: %v", err)
		}
	}()

	players := make(map[string]*glob.PlayerData)
	filedata, err := os.ReadFile(finalPath)
	if err != nil && !os.IsNotExist(err) {
		cwlog.DoLogCW("WritePlayers: " + err.Error())
		signalPlayerListDirty()
		return
	}
	if len(filedata) > 0 {
		if players, _, err = decodePlayerDatabase(filedata); err != nil {
			cwlog.DoLogCW("WritePlayers: " + err.Error())
			signalPlayerListDirty()
			return
		}
	}
	conflicts := mergePlayerChanges(players, changes)

	data, err := encodePlayerDatabase(players, cfg.Global.Paths.DataFiles.DBFormat)
	if err != nil {
		cwlog.DoLogCW("WritePlayers: " + err.Error())
		signalPlayerListDirty()
		return
	}
	if err := util.WriteBytesAtomic(finalPath, data, 0644); err != nil {
		cwlog.DoLogCW("WritePlayers: " + err.Error())
		signalPlayerListDirty()
		return
	}
	glob.PlayerListLock.Lock()
	var rankAcks []desiredRank
	var rankRepairs []desiredRank
	for name, change := range changes {
		current, saved := glob.PlayerList[name], players[name]
		if change.versions.record == 0 || current == nil || saved == nil {
			continue
		}
		if conflicts[name] {
			playerChanges.Lock()
			unchanged := playerChanges.versions[name].record == change.versions.record
			playerChanges.Unlock()
			if !unchanged {
				continue
			}
			if current.Level != saved.Level {
				rankRepairs = append(rankRepairs, desiredRank{Name: name, Level: saved.Level, Revision: saved.LevelRevision})
			}
			applyCommittedPlayerLevel(name, current, saved)
		} else {
			if current.LevelRevision != saved.LevelRevision && saved.Level >= 0 {
				rankAcks = append(rankAcks, desiredRank{Name: name, Level: saved.Level, Revision: saved.LevelRevision})
			}
			current.LevelRevision, current.BanRevision, current.BanRetained = saved.LevelRevision, saved.BanRevision, saved.BanRetained
		}
		rememberBanState(name, saved, false)
	}
	completePlayerChanges(changes)
	glob.PlayerListLock.Unlock()
	for _, rank := range rankAcks {
		acknowledgePlayerRank(rank)
	}
	for _, rank := range rankRepairs {
		syncOnlinePlayerLevel(rank.Name, -1, rank.Level)
	}
}
