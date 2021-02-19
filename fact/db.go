package fact

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"../cfg"
	"../constants"
	"../glob"
	"../logs"
	"github.com/fsnotify/fsnotify"
)

func WatchDatabaseFile() {

	go func() {
		for {
			time.Sleep(time.Second)
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				logs.Log(fmt.Sprintf("fsnotify error: %v", err))
			}

			done := make(chan bool)

			go func() {
				for {
					select {

					case event := <-watcher.Events:
						if event.Op&fsnotify.Write == fsnotify.Write {

							//New thread, so we don't miss events
							go func() {
								//logs.Log("Database updated, marking for read...")
								SetPlayerListUpdated()
							}()

							done <- true
							return
						}

					case err := <-watcher.Errors:
						logs.Log(fmt.Sprintf("fsnotify error: %v", err))
					}
				}
			}()

			if err := watcher.Add(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName); err != nil {
				logs.Log(fmt.Sprintf("fsnotify error: %v", err))
			}

			<-done
			watcher.Close()
		}

	}()
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

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {
			glob.PlayerList[i].ID = id
			glob.PlayerList[i].Level = level
			glob.PlayerList[i].LastSeen = t.Unix()

			SetPlayerListDirty()
			return true
		}
	}

	//Not in list, add them
	if glob.PlayerListMax < constants.MaxPlayers { //Don't go over max
		glob.PlayerList[glob.PlayerListMax].Name = pname
		glob.PlayerList[glob.PlayerListMax].Level = level
		glob.PlayerList[glob.PlayerListMax].ID = id
		glob.PlayerList[glob.PlayerListMax].LastSeen = t.Unix()
		glob.PlayerList[glob.PlayerListMax].Creation = t.Unix()
		glob.PlayerListMax++
	}

	SetPlayerListDirty()
	return false
}

//Faster
func UpdateSeen(pname string) {
	if pname == "" {
		return
	}

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {
			glob.PlayerList[i].LastSeen = t.Unix()

			SetPlayerListSeenDirty()
			return
		}
	}
}

func PlayerLevelSet(pname string, level int) bool {
	if pname == "" {
		return false
	}

	t := time.Now()

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {

			glob.PlayerList[i].LastSeen = t.Unix()
			SetPlayerListSeenDirty()

			if glob.PlayerList[i].Level < level && level > 0 {
				WhitelistPlayer(pname, level)
			}

			//If level didn't change, don't bother
			if glob.PlayerList[i].Level != level {
				glob.PlayerList[i].Level = level

				//Don't care about writing out new users often
				if level != 0 {
					SetPlayerListDirty()
				} else {
					SetPlayerListSeenDirty()
				}
			}

			return true
		}
	}

	//Not in list, add them
	if glob.PlayerListMax < constants.MaxPlayers { //Don't go over max
		glob.PlayerList[glob.PlayerListMax].Name = pname
		glob.PlayerList[glob.PlayerListMax].Level = level
		glob.PlayerList[glob.PlayerListMax].LastSeen = t.Unix()
		glob.PlayerList[glob.PlayerListMax].Creation = t.Unix()
		glob.PlayerListMax++

		WhitelistPlayer(pname, level)
	}

	//Don't care about writing out new users often
	if level != 0 {
		SetPlayerListDirty()
	} else {
		SetPlayerListSeenDirty()
	}
	return false
}

//Expects locked db, only used for LoadPlayers
func AddPlayer(pname string, level int, id string, creation int64, seen int64) {
	if pname == "" {
		return
	}

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {

			if level <= -254 {
				glob.PlayerList[i].Level = level
			} else if level == -1 && glob.PlayerList[i].Level != -1 {
				glob.PlayerList[i].Level = level
				WriteFact(fmt.Sprintf("/ban %s", pname))
			} else if level > glob.PlayerList[i].Level {
				glob.PlayerList[i].Level = level
				WhitelistPlayer(pname, level)
			}
			if creation > glob.PlayerList[i].Creation {
				glob.PlayerList[i].Creation = creation
			}
			if seen > glob.PlayerList[i].LastSeen {
				glob.PlayerList[i].LastSeen = seen
			}
			if id != "" && id != glob.PlayerList[i].ID {
				glob.PlayerList[i].ID = id
			}
			return
		}
	}

	//Not in list, add them
	if glob.PlayerListMax < constants.MaxPlayers { //Don't go over max
		glob.PlayerList[glob.PlayerListMax].Name = pname
		glob.PlayerList[glob.PlayerListMax].Level = level
		glob.PlayerList[glob.PlayerListMax].Creation = creation
		glob.PlayerList[glob.PlayerListMax].LastSeen = seen
		glob.PlayerList[glob.PlayerListMax].ID = id
		glob.PlayerListMax++

		WhitelistPlayer(pname, level)
	}
}

func PlayerLevelGet(pname string) int {
	if pname == "" {
		return 0
	}

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	t := time.Now()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {

			//Found in list
			glob.PlayerList[i].LastSeen = t.Unix()
			level := glob.PlayerList[i].Level
			SetPlayerListSeenDirty()
			return level
		}
	}

	//Not in list, add them
	if glob.PlayerListMax < constants.MaxPlayers { //Don't go over max
		glob.PlayerList[glob.PlayerListMax].Name = pname
		glob.PlayerList[glob.PlayerListMax].Level = 0
		glob.PlayerList[glob.PlayerListMax].LastSeen = t.Unix()
		glob.PlayerList[glob.PlayerListMax].Creation = t.Unix()
		glob.PlayerListMax++
	}

	//Don't care about writing out new users often
	SetPlayerListSeenDirty()
	return 0
}

func LoadPlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	filedata, err := ioutil.ReadFile(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName)
	if err != nil {
		logs.Log("Couldn't read db file, skipping...")
		return
	}

	if filedata != nil {
		dblines := strings.Split(string(filedata), ":")
		numlines := len(dblines)

		//Upgrade exsisting
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

		} else if dblines[0] == "db-v0.02" {

			glob.PlayerListLock.Lock()
			for pos := 0; pos < numlines; pos++ {

				items := strings.Split(string(dblines[pos]), ",")
				numitems := len(items)
				if numitems == 2 {
					pname := items[0]
					playerlevel, _ := strconv.Atoi(items[1])
					AddPlayer(pname, playerlevel, "", 0, 0)
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
		logs.Log("Couldn't open db file, skipping...")
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
	for i := 0; i < glob.PlayerListMax; i++ {

		if glob.PlayerList[i].Level > 0 {

			//Filter comma from names, just in case
			name := strings.ReplaceAll(glob.PlayerList[i].Name, ",", "") //remove comma
			name = strings.ReplaceAll(name, ":", "")                     //replace colon
			name = strings.ReplaceAll(name, "\n", "")                    //replace newline
			name = strings.ReplaceAll(name, "\r", "")                    //replace return
			buffer = buffer + fmt.Sprintf("%s,%d,%s,%v,%v:", name, glob.PlayerList[i].Level, glob.PlayerList[i].ID, glob.PlayerList[i].Creation, glob.PlayerList[i].LastSeen)
		}
	}
	glob.PlayerListLock.RUnlock()

	nfilename := fmt.Sprintf("pdb-%s.tmp", cfg.Local.ServerCallsign)
	err = ioutil.WriteFile(nfilename, []byte(buffer), 0644)

	if err != nil {
		logs.Log("Couldn't write db temp file.")
		return
	}

	oldName := nfilename
	newName := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName
	err = os.Rename(oldName, newName)

	if err != nil {
		logs.Log("Couldn't rename db temp file.")
		return
	}

}
