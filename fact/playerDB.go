package fact

import (
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
		versions map[string]uint64
	}{versions: make(map[string]uint64)}
)

const playerDBLockTimeout = 5 * time.Second

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

func markPlayerDirty(pname string) {
	pname = strings.ToLower(strings.TrimSpace(pname))
	if pname == "" {
		return
	}
	playerChanges.Lock()
	playerChanges.next++
	playerChanges.versions[pname] = playerChanges.next
	playerChanges.Unlock()
}

func playerChangePending(pname string) bool {
	playerChanges.Lock()
	_, ok := playerChanges.versions[strings.ToLower(pname)]
	playerChanges.Unlock()
	return ok
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
	markPlayerDirty(pname)
	signalPlayerListDirty()
}

// SetPlayerStatsDirty marks player stats as updated (LastSeen / Minutes).
// These changes are saved on a slower cadence than full DB changes.
func SetPlayerStatsDirty(pname string) {
	markPlayerDirty(pname)
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

/* Get playerID (Discord), add to db if not found */
func PlayerSetID(pname string, id string, level int) bool {

	if id == "" || pname == "" || len(pname) > constants.MaxNameLength {
		return false
	}

	pname = strings.ToLower(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {
		glob.PlayerList[pname].ID = id
		glob.PlayerList[pname].Level = level
		glob.PlayerList[pname].LastSeen = compactNow()

		SetPlayerListDirty(pname)
		return true
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    level,
		ID:       id,
		LastSeen: compactNow(),
		Creation: compactNow(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty(pname)
	return false
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
	return playerLevelSet(pname, level, modifyOnly, true)
}

// PlayerLevelSetFromGame records an in-game level change. SoftMod emits the
// same message for both a real promotion and a ChatWire-requested assignment,
// so matching levels must be ignored to avoid echo writes across servers.
func PlayerLevelSetFromGame(pname string, level int) bool {
	return playerLevelSet(pname, level, false, false)
}

func playerLevelSet(pname string, level int, modifyOnly, touchUnchanged bool) bool {
	if pname == "" || len(pname) > constants.MaxNameLength {
		return false
	}

	pname = strings.ToLower(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {
		if glob.PlayerList[pname].Level != level {
			glob.PlayerList[pname].LastSeen = compactNow()
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

	if modifyOnly {
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
	WhitelistPlayer(pname, level)

	return false
}

/*************************************************
 * Expects locked db, only used for LoadPlayers()
 *************************************************/
func addPlayer(iname string, level int, id string, creation int64, seen int64, reason string, susScore int64, mins int64, doBan bool) bool {
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
		} else if level >= 0 && glob.PlayerList[pname].Level == -1 { //Unbanned
			WriteUnban(pname)
		}

		if level != glob.PlayerList[pname].Level {
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

func snapshotPlayerChanges() (map[string]*glob.PlayerData, map[string]uint64) {
	changes := make(map[string]*glob.PlayerData)
	versions := make(map[string]uint64)

	glob.PlayerListLock.RLock()
	playerChanges.Lock()
	for pname, version := range playerChanges.versions {
		versions[pname] = version
		if player := glob.PlayerList[pname]; player != nil {
			copy := *player
			changes[pname] = &copy
		} else {
			changes[pname] = nil
		}
	}
	playerChanges.Unlock()
	glob.PlayerListLock.RUnlock()

	return changes, versions
}

func completePlayerChanges(versions map[string]uint64) {
	playerChanges.Lock()
	defer playerChanges.Unlock()
	for pname, version := range versions {
		if playerChanges.versions[pname] == version {
			delete(playerChanges.versions, pname)
		}
	}
}

func mergePlayerChanges(players, changes map[string]*glob.PlayerData) {
	for pname, player := range changes {
		if player == nil {
			delete(players, pname)
			continue
		}
		players[pname] = player
	}
}

/* Load database */
func LoadPlayers(bootMode, minimize, clearBans bool) {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	didBan := false
	dbPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile

	filedata, err := os.ReadFile(dbPath)
	if err != nil {
		cwlog.DoLogCW("Couldn't read db file, path: %v", dbPath)
		return
	}

	if filedata != nil {

		var tempData map[string]*glob.PlayerData
		tempData, _, err = decodePlayerDatabase(filedata)
		if err != nil {
			cwlog.DoLogCW("LoadPlayers: " + err.Error())
			return
		}

		banCount := 0
		doBan := true
		changedNames := make(map[string]struct{})
		levelChanges := make(map[string]int)
		//Add name back in, makes db file smaller
		glob.PlayerListLock.Lock()
		var removed int

		for pname := range tempData {
			if !bootMode && !minimize && !clearBans && playerChangePending(pname) {
				continue
			}
			previousLevel := 0
			previousPlayer := glob.PlayerList[pname]
			if previousPlayer != nil {
				previousLevel = previousPlayer.Level
			}

			if clearBans {
				if tempData[pname].Level < 0 {
					removed++
					delete(tempData, pname)
					delete(glob.PlayerList, pname)
					changedNames[pname] = struct{}{}
					continue
				}
			}
			//DB cleaning
			if minimize {
				//Get rid of new/deleted
				if tempData[pname].Level == 0 || tempData[pname].Level == -255 {
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
			if bootMode {
				didBan = addPlayer(pname, tempData[pname].Level, tempData[pname].ID, tempData[pname].Creation, tempData[pname].LastSeen, tempData[pname].BanReason, tempData[pname].SusScore, tempData[pname].Minutes, false)
			} else {
				didBan = addPlayer(pname, tempData[pname].Level, tempData[pname].ID, tempData[pname].Creation, tempData[pname].LastSeen, tempData[pname].BanReason, tempData[pname].SusScore, tempData[pname].Minutes, doBan)
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
	}
}

/* Save database */
func WritePlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	finalPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
	changes, versions := snapshotPlayerChanges()
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
	mergePlayerChanges(players, changes)

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
	completePlayerChanges(versions)
}
