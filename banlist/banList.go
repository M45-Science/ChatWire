package banlist

import (
	"encoding/json"
	"fmt"
	"io"
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

func CheckBanList(name string) {
	pname := strings.ToLower(name)
	BanListLock.Lock()
	defer BanListLock.Unlock()

	if cfg.Global.Paths.DataFiles.Bans == "" {
		return
	}

	for _, ban := range BanList {
		if strings.EqualFold(ban.UserName, pname) {
			if fact.PlayerLevelGet(ban.UserName, true) < 2 {
				fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, "Warning: [FCL] ban detected for "+pname+": "+ban.Reason)
			}
			break
		}
	}

	if fact.PlayerLevelGet(pname, false) == -1 {
		glob.PlayerListLock.Lock()
		if glob.PlayerList[pname] != nil {
			fact.WriteFact("/ban " + pname + " " + glob.PlayerList[pname].BanReason)
		}
		glob.PlayerListLock.Unlock()
	}
}

func WatchBanFile() {
	for glob.ServerRunning {
		time.Sleep(time.Minute)

		if cfg.Global.Paths.DataFiles.Bans == "" {
			break
		}

		filePath := cfg.Global.Paths.DataFiles.Bans
		initialStat, erra := os.Stat(filePath)

		if erra != nil {
			time.Sleep(time.Minute)
			continue
		}

		for glob.ServerRunning && initialStat != nil {
			time.Sleep(30 * time.Second)

			stat, errb := os.Stat(filePath)
			if errb != nil {
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				ReadBanFile()
				break
			}
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
		//log.Println(err)
		return
	}
	defer file.Close()

	var bData []banDataType

	data, err := io.ReadAll(file)

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
			bData = append(bData, banDataType{UserName: strings.ToLower(name)})
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

				if fact.PlayerLevelGet(aBan.UserName, true) >= 2 {
					buf = buf + fmt.Sprintf("Reg/Mod:Bypass:%v, %v", aBan.UserName, aBan.Reason)
				} else if aBan.Reason != "" {
					buf = buf + aBan.UserName + ": " + aBan.Reason
				} else {
					buf = buf + aBan.UserName
				}
			}
		}

	}
	if oldLen > 0 && strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) && buf != "" {

		fact.CMS(cfg.Global.Discord.ReportChannel, "New FCL bans: "+sclean.TruncateStringEllipsis(sclean.UnicodeCleanup(buf), 1000))
	}
}
