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

	if cfg.Global.Paths.DataFiles.Bans == "" {
		return
	}

	for _, ban := range BanList {
		if strings.EqualFold(ban.UserName, player) {
			if fact.PlayerLevelGet(ban.UserName, false) < 2 {
				fact.WriteFact("/ban " + ban.UserName + " [auto] " + ban.Reason)
			}
			break
		}
	}
}

func WatchBanFile() {
	for glob.ServerRunning {
		time.Sleep(time.Second * 15)

		if cfg.Global.Paths.DataFiles.Bans == "" {
			break
		}

		filePath := cfg.Global.Paths.DataFiles.Bans
		initialStat, erra := os.Stat(filePath)

		if erra != nil {
			//cwlog.DoLogCW("watchBanFile: stat")
			time.Sleep(time.Minute)
			continue
		}

		for glob.ServerRunning && initialStat != nil {
			stat, errb := os.Stat(filePath)
			if errb != nil {
				//cwlog.DoLogCW("watchBanFile: restat")
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				ReadBanFile()
				break
			}

			time.Sleep(30 * time.Second)
		}
	}
}

func ReadBanFile() {
	BanListLock.Lock()
	defer BanListLock.Unlock()

	if cfg.Global.Paths.DataFiles.Bans == "" {
		return
	}

	file, err := os.Open(cfg.Global.Paths.DataFiles.Bans)

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
				if strings.EqualFold(bBan.UserName, aBan.UserName) {
					found = true
					break
				}
			}
			if !found {
				BanList = append(BanList, aBan)
				if buf != "" {
					buf = buf + ", "
				}

				if fact.PlayerLevelGet(aBan.UserName, false) >= 2 {
					buf = buf + fmt.Sprintf("Reg/Mod:Bypass:%v", aBan.UserName, aBan.Reason)
				} else if aBan.Reason != "" {
					buf = buf + aBan.UserName + ": " + aBan.Reason
				} else {
					buf = buf + aBan.UserName
				}
			}
		}

	}
	if oldLen > 0 && strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) && buf != "" {

		fact.CMS(cfg.Global.Discord.ReportChannel, "New bans: "+sclean.TruncateStringEllipsis(sclean.StripControlAndSubSpecial(buf), 1000))
	}
}
