package fact

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Distortions81/M45-ChatWire/cfg"
	"github.com/Distortions81/M45-ChatWire/constants"
	"github.com/Distortions81/M45-ChatWire/glob"
	"github.com/Distortions81/M45-ChatWire/sclean"
	"github.com/fsnotify/fsnotify"
)

func WatchDatabaseFile() {

	go func() {
		for {
			time.Sleep(time.Second)
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				log.Println(fmt.Sprintf("fsnotify error: %v", err))
			}

			done := make(chan bool)

			go func() {
				for {
					select {

					case event := <-watcher.Events:
						if event.Op&fsnotify.Write == fsnotify.Write {
							SetPlayerListUpdated()
							done <- true
							return
						}

					case err := <-watcher.Errors:
						log.Println(fmt.Sprintf("fsnotify error: %v", err))
						done <- true
						return
					}
				}
			}()

			if err := watcher.Add(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName); err != nil {
				log.Println(fmt.Sprintf("fsnotify error: %v", err))
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

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

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

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

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

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

	t := time.Now()

	WhitelistPlayer(pname, level)

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i].Name == pname {

			glob.PlayerList[i].LastSeen = t.Unix()

			if glob.PlayerList[i].Level != level {
				SetPlayerListDirty()
			} else {
				SetPlayerListSeenDirty()
			}

			glob.PlayerList[i].Level = level
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

		SetPlayerListDirty()
	}

	return false
}

//Expects locked db, only used for LoadPlayers
func AddPlayer(pname string, level int, id string, creation int64, seen int64) {
	if pname == "" {
		return
	}

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

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
		pname = strings.ReplaceAll(pname, ",", "") //remove comma
		pname = strings.ReplaceAll(pname, ":", "") //replace colon
		pname = sclean.StripControlAndSubSpecial(pname)
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

	pname = strings.ReplaceAll(pname, ",", "") //remove comma
	pname = strings.ReplaceAll(pname, ":", "") //replace colon
	pname = sclean.StripControlAndSubSpecial(pname)

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
		log.Println("Couldn't read db file, skipping...")
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
					pname = strings.ReplaceAll(pname, ",", "") //remove comma
					pname = strings.ReplaceAll(pname, ":", "") //replace colon
					pname = sclean.StripControlAndSubSpecial(pname)
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
		log.Println("Couldn't open db file, skipping...")
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

		if glob.PlayerList[i].Level >= 0 {

			//Filter comma from names, just in case
			name := strings.ReplaceAll(glob.PlayerList[i].Name, ",", "") //remove comma
			name = strings.ReplaceAll(name, ":", "")                     //replace colon
			name = sclean.StripControlAndSubSpecial(name)
			buffer = buffer + fmt.Sprintf("%s,%d,%s,%v,%v:", name, glob.PlayerList[i].Level, glob.PlayerList[i].ID, glob.PlayerList[i].Creation, glob.PlayerList[i].LastSeen)
		}
	}
	glob.PlayerListLock.RUnlock()

	nfilename := fmt.Sprintf("pdb-%s.tmp", cfg.Local.ServerCallsign)
	err = ioutil.WriteFile(nfilename, []byte(buffer), 0644)

	if err != nil {
		log.Println("Couldn't write db temp file.")
		return
	}

	oldName := nfilename
	newName := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName
	err = os.Rename(oldName, newName)

	if err != nil {
		log.Println("Couldn't rename db temp file.")
		return
	}

}
