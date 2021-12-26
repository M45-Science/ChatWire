package fact

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/glob"
	"ChatWire/sclean"
)

//Screw fsnotify
func WatchDatabaseFile() {

	filePath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName

	initialStat, err := os.Stat(filePath)
	if err != nil {
		return
	}

	for glob.ServerRunning {
		stat, err := os.Stat(filePath)
		if err != nil {
			return
		}

		if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
			SetPlayerListUpdated()
			initialStat, err = os.Stat(filePath)
			if err != nil {
				return
			}
		}

		time.Sleep(1 * time.Second)
	}
}

func IsPlayerListUpdated() bool {
	glob.PlayerListUpdatedLock.Lock()
	reply := glob.PlayerListUpdated
	glob.PlayerListUpdatedLock.Unlock()

	return reply
}

func SetPlayerListUpdated() {
	glob.PlayerListUpdatedLock.Lock()
	glob.PlayerListUpdated = true
	glob.PlayerListUpdatedLock.Unlock()
}

func SetPlayerListDirty() {
	glob.PlayerListDirtyLock.Lock()
	glob.PlayerListDirty = true
	glob.PlayerListDirtyLock.Unlock()
}

func SetPlayerListSeenDirty() {
	glob.PlayerListSeenDirtyLock.Lock()
	glob.PlayerListSeenDirty = true
	glob.PlayerListSeenDirtyLock.Unlock()
}

//Always marks db dirty, important
func PlayerSetID(pname string, id string, level int) bool {

	if id == "" || pname == "" {
		return false
	}

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
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

	//Not in list, add them
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

//Faster
func UpdateSeen(pname string) {
	if pname == "" {
		return
	}

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
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

func PlayerLevelSet(pname string, level int) bool {
	if pname == "" {
		return false
	}

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
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
		return true
	}

	//Not in list, add them
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

//Expects locked db, only used for LoadPlayers
func AddPlayer(pname string, level int, id string, creation int64, seen int64) {
	if pname == "" {
		return
	}

	if glob.PlayerList[pname] != nil {
		if level <= -254 {
			glob.PlayerList[pname].Level = level
		} else if level == -1 && glob.PlayerList[pname].Level != -1 {
			glob.PlayerList[pname].Level = level
			WriteFact(fmt.Sprintf("/ban %s", pname))
		} else if level > glob.PlayerList[pname].Level {
			glob.PlayerList[pname].Level = level
			WhitelistPlayer(pname, level)
		}
		if creation > glob.PlayerList[pname].Creation {
			glob.PlayerList[pname].Creation = creation
		}
		if seen > glob.PlayerList[pname].LastSeen {
			glob.PlayerList[pname].LastSeen = seen
		}
		if id != "" && id != glob.PlayerList[pname].ID {
			glob.PlayerList[pname].ID = id
		}
		return
	}

	//Not in list, add them
	t := time.Now()
	newplayer := glob.PlayerData{

		Name:     pname,
		Level:    level,
		ID:       "",
		LastSeen: t.Unix(),
		Creation: t.Unix(),
	}
	glob.PlayerList[pname] = &newplayer
	//SetPlayerListDirty()
	WhitelistPlayer(pname, level)
}

func PlayerLevelGet(pname string) int {
	if pname == "" {
		return 0
	}

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	if glob.PlayerList[pname] != nil {

		//Found in list
		glob.PlayerList[pname].LastSeen = t.Unix()
		level := glob.PlayerList[pname].Level
		SetPlayerListSeenDirty()
		return level
	}

	//Not in list, add them
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

func LoadPlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	filedata, err := ioutil.ReadFile(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName)
	if err != nil {
		botlog.DoLog("Couldn't read db file, skipping...")
		return
	}

	if filedata != nil {
		dblines := strings.Split(string(filedata), ":")
		numlines := len(dblines)

		//Upgrade existing
		if dblines[0] == "db-v0.03" {

			glob.PlayerListLock.Lock()
			for pos := 0; pos < numlines; pos++ {
				items := strings.Split(string(dblines[pos]), ",")
				numitems := len(items)
				if numitems == 5 {
					pname := items[0]
					playerlevel, _ := strconv.Atoi(items[1])
					pid := items[2]
					creation, _ := strconv.ParseInt(items[3], 10, 64)
					seen, _ := strconv.ParseInt(items[4], 10, 64)
					AddPlayer(pname, playerlevel, pid, creation, seen)
				}
			}
			glob.PlayerListLock.Unlock()

		}
	}
}

func WritePlayers() {
	//Write to file
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	buffer := ""

	fo, err := os.Create(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName)
	if err != nil {
		botlog.DoLog("Couldn't open db file, skipping...")
		return
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	buffer = buffer + "db-v0.03:"
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		buffer = buffer + fmt.Sprintf("%s,%d,%s,%v,%v:", player.Name, player.Level, player.ID, player.Creation, player.LastSeen)
	}
	glob.PlayerListLock.RUnlock()

	nfilename := fmt.Sprintf("pdb-%s.tmp", cfg.Local.ServerCallsign)
	err = ioutil.WriteFile(nfilename, []byte(buffer), 0644)

	if err != nil {
		botlog.DoLog("Couldn't write db temp file.")
		return
	}

	oldName := nfilename
	newName := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName
	err = os.Rename(oldName, newName)

	if err != nil {
		botlog.DoLog("Couldn't rename db temp file.")
		return
	}

}
