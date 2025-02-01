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
	Revoked  bool   `json:"-"`
}

func CheckBanList(name string, doWarn bool) bool {
	pname := strings.ToLower(name)
	BanListLock.Lock()
	defer BanListLock.Unlock()

	for _, ban := range BanList {
		if ban.Revoked {
			continue
		}
		if strings.EqualFold(ban.UserName, pname) {
			warn := "Warning: [FCL] ban found for '" + pname + "': " + ban.Reason

			if fact.PlayerLevelGet(ban.UserName, true) < 2 {
				if cfg.Global.Options.FCLWarnOnly {
					if doWarn {
						fact.CMS(cfg.Local.Channel.ChatChannel, warn)
					}
				} else {
					fact.WriteBan(pname, "[FCL] "+ban.Reason)
					return true
				}
			} else if cfg.Global.Options.FCLWarnRegulars {
				if doWarn {
					fact.CMS(cfg.Local.Channel.ChatChannel, warn)
				}
			}

			return false
		}
	}

	if fact.PlayerLevelGet(pname, false) == -1 {
		glob.PlayerListLock.Lock()
		if glob.PlayerList[pname] != nil {
			fact.WriteBan(pname, glob.PlayerList[pname].BanReason)
		}
		glob.PlayerListLock.Unlock()
		return true
	}

	return false
}

func WatchBanFile() {
	for glob.ServerRunning {
		time.Sleep(10 * time.Second)

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
			time.Sleep(5 * time.Second)

			stat, errb := os.Stat(filePath)
			if errb != nil {
				break
			}

			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				ReadBanFile(false)
				break
			}
		}
	}
}

func ReadBanFile(firstboot bool) {
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

	var newBans []banDataType
	for _, name := range names {
		if name != "" {
			newBans = append(newBans, banDataType{UserName: strings.ToLower(name)})
		}
	}

	/* Standard format bans */
	err = json.Unmarshal(data, &newBans)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	//Empty, just return
	if len(newBans) <= 0 {
		return
	}
	revBuf := ""
	//Detect removed bans
	for o, oldBan := range BanList {
		if oldBan.UserName == "" {
			continue
		}
		found := false
		for _, newBan := range newBans {
			if strings.EqualFold(oldBan.UserName, newBan.UserName) {
				found = true
				break
			}
		}
		if !found {
			if revBuf != "" {
				revBuf = revBuf + ", "
			} else {
				revBuf = "REVOKED bans: "
			}
			revBuf = revBuf + oldBan.UserName
			BanList[o].Revoked = true
		}

	}

	banBuf := ""
	for _, newBan := range newBans {
		found := false
		if newBan.UserName == "" {
			continue
		}
		for o, oldban := range BanList {
			if strings.EqualFold(oldban.UserName, newBan.UserName) {
				if oldban.Revoked {
					if banBuf != "" {
						banBuf = banBuf + ", "
					}
					banBuf = banBuf + "REINSTATED Ban: " + newBan.UserName
					if newBan.Reason != "" {
						banBuf = banBuf + " -- " + newBan.Reason
					}
					BanList[o].Revoked = false
				}
				found = true
				break
			}
		}
		if !found {
			BanList = append(BanList, newBan)
			if banBuf != "" {
				banBuf = banBuf + ", "
			} else {
				banBuf = "NEW bans: "
			}

			level := fact.PlayerLevelGet(newBan.UserName, false)
			if level >= 2 {
				levelName := fact.LevelToString(level)
				banBuf = banBuf + fmt.Sprintf("M45 Level: %v -- Bypassing FCL ban for: ", levelName)
			}

			if newBan.Reason != "" {
				banBuf = banBuf + newBan.UserName + " -- " + newBan.Reason
			} else {
				banBuf = banBuf + newBan.UserName
			}
		}
	}

	if firstboot { //Don't show on first boot
		return
	}

	if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) && banBuf != "" {
		fact.CMS(cfg.Global.Discord.ReportChannel, "[FCL] "+sclean.TruncateStringEllipsis(sclean.UnicodeCleanup(banBuf), 1000))
	}
	if strings.EqualFold(cfg.Global.PrimaryServer, cfg.Local.Callsign) && revBuf != "" {
		fact.CMS(cfg.Global.Discord.ReportChannel, "[FCL] "+sclean.TruncateStringEllipsis(sclean.UnicodeCleanup(revBuf), 1000))
	}
}
