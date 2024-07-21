package fact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

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

func CompactTime(input int64) int64 {
	return (input - constants.SeenEpoch) / constants.SeenDivisor
}

/* Screw fsnotify */
func WatchDatabaseFile() {
	for glob.ServerRunning {
		time.Sleep(time.Second * 5)

		filePath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
		initialStat, erra := os.Stat(filePath)

		if erra != nil {
			cwlog.DoLogCW("WatchDatabaseFile: stat")
			time.Sleep(time.Minute)
			continue
		}

		for glob.ServerRunning && initialStat != nil {
			time.Sleep(5 * time.Second)

			stat, errb := os.Stat(filePath)
			if errb != nil {
				cwlog.DoLogCW("WatchDatabaseFile: restat")
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				SetPlayerListUpdated()
				break
			}
		}
	}
}

/* Check if DB has been updated */
func IsPlayerListUpdated() bool {
	glob.PlayerListUpdatedLock.Lock()
	reply := glob.PlayerListUpdated
	glob.PlayerListUpdatedLock.Unlock()

	return reply
}

/* Set DB as updated */
func SetPlayerListUpdated() {
	glob.PlayerListUpdatedLock.Lock()
	glob.PlayerListUpdated = true
	glob.PlayerListUpdatedLock.Unlock()
}

/* Mark DB dirty */
func SetPlayerListDirty() {
	glob.PlayerListDirtyLock.Lock()
	glob.PlayerListDirty = true
	glob.PlayerListDirtyLock.Unlock()
}

/* Mark DB as SeenDirty (low priority) */
func SetPlayerListSeenDirty() {
	glob.PlayerListSeenDirtyLock.Lock()
	glob.PlayerListSeenDirty = true
	glob.PlayerListSeenDirtyLock.Unlock()
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
			WriteFact("/ban " + pname + " " + reason)
		}

		glob.PlayerList[pname].Level = -1
		if reason != "" {
			glob.PlayerList[pname].BanReason = reason
		}
		glob.PlayerList[pname].LastSeen = 0
		glob.PlayerList[pname].Creation = 0
		glob.PlayerList[pname].AlreadyBanned = true

		SetPlayerListDirty()
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
		WriteFact("/ban " + pname + " " + reason)
	}

	SetPlayerListDirty()
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

		SetPlayerListDirty()
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

	SetPlayerListDirty()
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

		SetPlayerListSeenDirty()
		return
	}
}

/* Set player level, add to db if not found */
func PlayerLevelSet(pname string, level int, modifyOnly bool) bool {
	if pname == "" || len(pname) > constants.MaxNameLength {
		return false
	}

	pname = strings.ToLower(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {

		glob.PlayerList[pname].LastSeen = compactNow()

		if glob.PlayerList[pname].Level != level {
			glob.PlayerList[pname].Level = level
			SetPlayerListDirty()
			WhitelistPlayer(pname, level)
		} else {
			SetPlayerListSeenDirty()
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

	SetPlayerListDirty()
	WhitelistPlayer(pname, level)

	return false
}

/*************************************************
 * Expects locked db, only used for LoadPlayers()
 *************************************************/
func AddPlayer(iname string, level int, id string, creation int64, seen int64, reason string, susScore int64, mins int64, doBan bool) bool {
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
			if reason != "" {
				glob.PlayerList[pname].BanReason = reason
				WriteFact("/ban " + pname + " " + reason)
			} else {
				WriteFact("/ban " + pname)
			}
			didBan = true
		} else if level >= 0 && glob.PlayerList[pname].Level == -1 { //Unbanned
			WriteFact("/unban " + pname)
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
		if reason != "" {
			WriteFact("/ban " + pname + " " + reason)
		} else {
			WriteFact("/ban " + pname)
		}
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
		SetPlayerListSeenDirty()
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

	SetPlayerListDirty()
	return 0
}

/* Load database */
func LoadPlayers(bootMode, minimize bool) {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	didBan := false

	filedata, err := os.ReadFile(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile)
	if err != nil {
		cwlog.DoLogCW("Couldn't read db file, skipping...")
		return
	}

	if filedata != nil {

		if strings.HasSuffix(cfg.Global.Paths.DataFiles.DBFile, ".json") {

			var tempData = make(map[string]*glob.PlayerData)
			err = json.Unmarshal(filedata, &tempData)
			if err != nil {
				cwlog.DoLogCW(err.Error())
			}

			banCount := 0
			doBan := true
			//Add name back in, makes db file smaller
			glob.PlayerListLock.Lock()
			var removed int
			for pname := range tempData {
				//DB cleaning
				if minimize {
					//Get rid of new/deleted
					if tempData[pname].Level == 0 || tempData[pname].Level == -255 {
						removed++
						defer delete(tempData, pname)
						continue
					}
					//Delete uneeded data from member/reg/moderator
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
				didBan = false
				if tempData[pname].Level == 2 && tempData[pname].Minutes > constants.VeteranThresh {
					tempData[pname].Level = 3
				}
				if bootMode {
					didBan = AddPlayer(pname, tempData[pname].Level, tempData[pname].ID, tempData[pname].Creation, tempData[pname].LastSeen, tempData[pname].BanReason, tempData[pname].SusScore, tempData[pname].Minutes, false)
				} else if !bootMode {
					didBan = AddPlayer(pname, tempData[pname].Level, tempData[pname].ID, tempData[pname].Creation, tempData[pname].LastSeen, tempData[pname].BanReason, tempData[pname].SusScore, tempData[pname].Minutes, doBan)
				}
				if didBan {
					banCount++
				}
			}
			if removed > 0 {
				fmt.Printf("Removed: %v entries.\n", removed)
			}
			glob.PlayerListLock.Unlock()
		}
	}
}

/* Save database */
func WritePlayers() {
	/* Write to file */
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	fo, err := os.Create(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile)
	if err != nil {
		cwlog.DoLogCW("Couldn't open db file, skipping...")
		return
	}
	/*  close fo on exit and check for its returned error */
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	glob.PlayerListLock.RLock()

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	if err := enc.Encode(glob.PlayerList); err != nil {
		cwlog.DoLogCW("WritePlayers: enc.Encode failure")
		return
	}
	glob.PlayerListLock.RUnlock()

	nfilename := fmt.Sprintf("pdb-%s.tmp", cfg.Local.Callsign)
	err = os.WriteFile(nfilename, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("Couldn't write db temp file.")
		return
	}

	oldName := nfilename
	newName := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
	err = os.Rename(oldName, newName)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename db temp file.")
		return
	}

}
