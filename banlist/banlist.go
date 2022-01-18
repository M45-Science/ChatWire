package banlist

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/glob"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"
)

var BanList []banDataType
var BanListLock sync.Mutex

type banDataType struct {
	UserName string `json:"username"`
	Reason   string `json:"reason,omitempty"`
}

func CheckBanList(player string) {
	BanListLock.Lock()
	defer BanListLock.Unlock()

	if cfg.Global.PathData.BanFile == "" {
		return
	}

	for _, ban := range BanList {
		if ban.UserName == player {
			fact.WriteFact("/ban " + ban.UserName + "[auto] " + ban.Reason)
			break
		}
	}
}

func WatchBanFile() {
	for glob.ServerRunning {
		time.Sleep(time.Second * 5)

		if cfg.Global.PathData.BanFile == "" {
			break
		}

		filePath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.DBFileName
		initialStat, erra := os.Stat(filePath)

		if erra != nil {
			botlog.DoLog("watchBanFile: stat")
			continue
		}

		for glob.ServerRunning && initialStat != nil {
			stat, errb := os.Stat(filePath)
			if errb != nil {
				botlog.DoLog("watchBanFile: restat")
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				go readBanFile()
				break
			}

			time.Sleep(1 * time.Second)
		}
	}
}

func readBanFile() {
	BanListLock.Lock()
	defer BanListLock.Unlock()

	if cfg.Global.PathData.BanFile == "" {
		return
	}

	file, err := os.Open(cfg.Global.PathData.BanFile)

	if err != nil {
		log.Println(file, err)
		return
	}
	defer file.Close()

	var bData []banDataType

	data, err := ioutil.ReadAll(file)

	if err != nil {
		log.Println(err)
		return
	}

	/* This area deals with 'array of strings' format */
	var names []string
	json.Unmarshal(data, &names)

	for _, name := range names {
		if name != "" {
			bData = append(bData, banDataType{UserName: name})
		}
	}

	/* Standard format bans */
	json.Unmarshal(data, &bData)
}
