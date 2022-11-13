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

	if pname == "" {
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

	if id == "" || pname == "" {
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
	if pname == "" {
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
	if pname == "" {
		return false
	}

	pname = strings.ToLower(pname)
	WhitelistPlayer(pname, level)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {

		glob.PlayerList[pname].LastSeen = compactNow()

		if glob.PlayerList[pname].Level != level {
			SetPlayerListDirty()
		} else {
			SetPlayerListSeenDirty()
		}

		glob.PlayerList[pname].Level = level
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
		ID:       "",
		LastSeen: compactNow(),
		Creation: compactNow(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty()

	return false
}

/*************************************************
 * Expects locked db, only used for LoadPlayers()
 *************************************************/
func AddPlayer(pname string, level int, id string, creation int64, seen int64, reason string) {
	if pname == "" {
		return
	}

	pname = strings.ToLower(pname)

	if glob.PlayerList[pname] != nil {
		if level <= -254 { //Delete
			glob.PlayerList[pname].Level = level
			/*Clear discord ID on delete*/
			glob.PlayerList[pname].ID = "0"
		} else if level == -1 && glob.PlayerList[pname].Level >= 0 { //Banned
			glob.PlayerList[pname].Level = level
			if reason != "" {
				glob.PlayerList[pname].BanReason = reason
				WriteFact("/ban " + pname + " " + reason)
			} else {
				WriteFact("/ban " + pname)
			}
		} else if level >= 0 && glob.PlayerList[pname].Level == -1 { //Unbanned
			glob.PlayerList[pname].Level = level
			WriteFact("/unban " + pname)
		} else if level > glob.PlayerList[pname].Level { //Promoted
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
		if id != "" { //Registered
			glob.PlayerList[pname].ID = id
		}
		return
	}

	/* Don't load level 0 players that are not registered */
	if level != 0 || id != "" {
		/* Not in list, add them */
		newplayer := glob.PlayerData{

			Name:      pname,
			Level:     level,
			ID:        id,
			BanReason: reason,
			LastSeen:  seen,
			Creation:  creation,
		}

		glob.PlayerList[pname] = &newplayer
		WhitelistPlayer(pname, level)
	}

}

/* Get player level, add to db if not found */
func PlayerLevelGet(pname string, modifyOnly bool) int {
	if pname == "" {
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
func LoadPlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	filedata, err := os.ReadFile(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile)
	if err != nil {
		cwlog.DoLogCW("Couldn't read db file, skipping...")
		return
	}

	if filedata != nil {

		if strings.HasSuffix(cfg.Global.Paths.DataFiles.DBFile, ".dat") {
			dblines := strings.Split(string(filedata), ":")
			dblen := len(dblines)

			/* Upgrade existing */
			if strings.EqualFold(dblines[0], "db-v0.03") {

				glob.PlayerListLock.Lock()
				for pos, line := range dblines {
					items := strings.Split(string(line), ",")
					numitems := len(items)
					if numitems == 5 {
						pname := strings.ToLower(items[0])
						playerlevel, _ := strconv.Atoi(items[1])
						pid := items[2]
						creation, _ := strconv.ParseInt(items[3], 10, 64)
						creation = CompactTime(creation)
						seen, _ := strconv.ParseInt(items[4], 10, 64)
						seen = CompactTime(seen)

						if playerlevel != 0 || len(pid) > 1 {
							AddPlayer(pname, playerlevel, pid, creation, seen, "")
						}
					} else if pos != 0 && pos != dblen-1 {
						cwlog.DoLogCW(fmt.Sprintf("Invalid db line %v:, skipping...", pos))
					}
				}
				cfg.Global.Paths.DataFiles.DBFile = "playerdb.json"
				cfg.WriteGCfg()
				glob.PlayerListLock.Unlock()

			}
		} else if strings.HasSuffix(cfg.Global.Paths.DataFiles.DBFile, ".json") {

			var tempData = make(map[string]*glob.PlayerData)
			err = json.Unmarshal(filedata, &tempData)
			if err != nil {
				cwlog.DoLogCW(err.Error())
			}

			//Add name back in, makes db file smaller
			glob.PlayerListLock.Lock()
			for pname := range tempData {
				tempData[pname].Name = pname
				if tempData[pname].Level != 0 || tempData[pname].ID != "" {
					AddPlayer(pname, tempData[pname].Level, tempData[pname].ID, tempData[pname].Creation, tempData[pname].LastSeen, tempData[pname].BanReason)
				}
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
