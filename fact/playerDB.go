package fact

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/sclean"
)

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
			stat, errb := os.Stat(filePath)
			if errb != nil {
				cwlog.DoLogCW("WatchDatabaseFile: restat")
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				SetPlayerListUpdated()
				break
			}

			time.Sleep(5 * time.Second)
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

/* Get playerID (Discord), add to db if not found */
func PlayerSetID(pname string, id string, level int) bool {

	if id == "" || pname == "" {
		return false
	}

	pname = strings.ToLower(pname)

	pname = strings.ReplaceAll(pname, ",", "") /* remove comma */
	pname = strings.ReplaceAll(pname, ":", "") /* replace colon */
	id = strings.ReplaceAll(id, ",", "")       /* remove comma */
	id = strings.ReplaceAll(id, ":", "")       /* replace colon */
	pname = sclean.StripControlAndSubSpecial(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	if glob.PlayerList[pname] != nil {
		glob.PlayerList[pname].ID = id
		glob.PlayerList[pname].Level = level
		glob.PlayerList[pname].LastSeen = t.Unix()

		SetPlayerListDirty()
		return true
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    level,
		ID:       id,
		LastSeen: t.Unix(),
		Creation: t.Unix(),
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

	pname = strings.ReplaceAll(pname, ",", "") /* remove comma */
	pname = strings.ReplaceAll(pname, ":", "") /* replace colon */
	pname = sclean.StripControlAndSubSpecial(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	if glob.PlayerList[pname] != nil {
		glob.PlayerList[pname].LastSeen = t.Unix()

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

	pname = strings.ReplaceAll(pname, ",", "") /* remove comma */
	pname = strings.ReplaceAll(pname, ":", "") /* replace colon */
	pname = sclean.StripControlAndSubSpecial(pname)

	t := time.Now()

	WhitelistPlayer(pname, level)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	if glob.PlayerList[pname] != nil {

		glob.PlayerList[pname].LastSeen = t.Unix()

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
		LastSeen: t.Unix(),
		Creation: t.Unix(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty()

	return false
}

/*************************************************
 * Expects locked db, only used for LoadPlayers()
 *************************************************/
func AddPlayer(pname string, level int, id string, creation int64, seen int64) {
	if pname == "" {
		return
	}

	pname = strings.ToLower(pname)

	if glob.PlayerList[pname] != nil {
		if level <= -254 {
			glob.PlayerList[pname].Level = level
			/*Clear discord ID on delete*/
			glob.PlayerList[pname].ID = "0"
		} else if level == -1 && glob.PlayerList[pname].Level != -1 {
			glob.PlayerList[pname].Level = level

			/* Use discordid as a sneaky way to pass ban reason */
			idReason := id
			reason := "Banned on a different server."
			if sclean.AlphaOnly(idReason) != "" {
				reason = idReason
			}
			WriteFact(fmt.Sprintf("/ban %v %v", pname, reason))

		} else if level > glob.PlayerList[pname].Level {
			glob.PlayerList[pname].Level = level
			WhitelistPlayer(pname, level)
		}
		if creation > 0 {
			glob.PlayerList[pname].Creation = creation
		}
		if seen > glob.PlayerList[pname].LastSeen {
			glob.PlayerList[pname].LastSeen = seen
		}
		if id != "" {
			glob.PlayerList[pname].ID = id
		}
		return
	}

	/* Not in list, add them */
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    level,
		ID:       id,
		LastSeen: seen,
		Creation: creation,
	}
	glob.PlayerList[pname] = &newplayer
	WhitelistPlayer(pname, level)
}

/* Get player level, add to db if not found */
func PlayerLevelGet(pname string, modifyOnly bool) int {
	if pname == "" {
		return 0
	}

	pname = strings.ToLower(pname)

	pname = strings.ReplaceAll(pname, ",", "") /* remove comma */
	pname = strings.ReplaceAll(pname, ":", "") /* replace colon */
	pname = sclean.StripControlAndSubSpecial(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	if glob.PlayerList[pname] != nil {

		/* Found in list */
		glob.PlayerList[pname].LastSeen = t.Unix()
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
		LastSeen: t.Unix(),
		Creation: t.Unix(),
	}
	glob.PlayerList[pname] = &newplayer

	SetPlayerListDirty()
	return 0
}

/* Load database */
func LoadPlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	filedata, err := ioutil.ReadFile(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile)
	if err != nil {
		cwlog.DoLogCW("Couldn't read db file, skipping...")
		return
	}

	if filedata != nil {
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
					seen, _ := strconv.ParseInt(items[4], 10, 64)
					AddPlayer(pname, playerlevel, pid, creation, seen)
				} else if pos != 0 && pos != dblen-1 {
					cwlog.DoLogCW(fmt.Sprintf("Invalid db line %v:, skipping...", pos))
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

	buffer := ""

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

	buffer = buffer + "db-v0.03:"
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {

		/* Don't bother saving new players that are not registered */
		if player.Level != 0 || len(player.ID) > 1 {
			buffer = buffer + fmt.Sprintf("%s,%d,%s,%v,%v:", strings.ToLower(player.Name), player.Level, player.ID, player.Creation, player.LastSeen)
		}
	}
	glob.PlayerListLock.RUnlock()

	nfilename := fmt.Sprintf("pdb-%s.tmp", cfg.Local.Callsign)
	err = ioutil.WriteFile(nfilename, []byte(buffer), 0644)

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
