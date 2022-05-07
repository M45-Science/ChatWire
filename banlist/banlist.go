package banlist

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
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
		if strings.EqualFold(ban.UserName, player) {
			fact.WriteFact("/ban " + ban.UserName + " [auto] " + ban.Reason)
			break
		}
	}
}

func WatchBanFile() {
	for glob.ServerRunning {

		if cfg.Global.PathData.BanFile == "" {
			break
		}

		filePath := cfg.Global.PathData.BanFile
		initialStat, erra := os.Stat(filePath)

		if erra != nil {
			cwlog.DoLogCW("watchBanFile: stat")
			continue
		}

		time.Sleep(time.Second)

		for glob.ServerRunning && initialStat != nil {
			stat, errb := os.Stat(filePath)
			if errb != nil {
				cwlog.DoLogCW("watchBanFile: restat")
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				go ReadBanFile()
				break
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func ReadBanFile() {
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
		//log.Println(err)
		return
	}

	/* This area deals with 'array of strings' format */
	var names []string
	err = json.Unmarshal(data, &names)
	if err != nil {
		fmt.Print("")
		//Ignore error
	}

	for _, name := range names {
		if name != "" {
			bData = append(bData, banDataType{UserName: name})
		}
	}

	/* Standard format bans */
	err = json.Unmarshal(data, &bData)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	oldLen := len(BanList)
	buf := ""
	for _, aBan := range bData {
		found := false
		if aBan.UserName != "" {
			for _, bBan := range BanList {
				if bBan.UserName == aBan.UserName {
					found = true
					break
				}
			}
			if !found {
				BanList = append(BanList, aBan)
				if buf != "" {
					buf = buf + ", "
				}
				if aBan.Reason != "" {
					buf = buf + aBan.UserName + ": " + aBan.Reason
				} else {
					buf = buf + aBan.UserName
				}
			}
		}

	}
	if oldLen > 0 && cfg.Local.ReportNewBans && buf != "" {
		fact.CMS(cfg.Global.DiscordData.ReportChannelID, "New bans: "+sclean.TruncateStringEllipsis(sclean.StripControlAndSubSpecial(buf), 500))
	}
}
