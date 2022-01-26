package disc

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
)

/* Discord role member-lists */
var RoleListLock sync.Mutex
var RoleList RoleListData
var RoleListUpdated bool

/* Cache a list of users with specific Discord roles */
func WriteRoleList() bool {
	RoleListLock.Lock()
	defer RoleListLock.Unlock()

	tempPath := constants.RoleListFile + "." + cfg.Local.ServerCallsign + ".tmp"
	finalPath := constants.RoleListFile

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	RoleList.Version = "0.0.1"

	if err := enc.Encode(RoleList); err != nil {
		cwlog.DoLogCW("Writecfg.RoleList: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("Writecfg.RoleList: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("Writecfg.RoleList: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename RoleList file.")
		return false
	}

	return true
}

/* Create a new RoleList */
func CreateRoleList() RoleListData {
	newcfg := RoleListData{Version: "0.0.1"}
	return newcfg
}

/* Read in cached list of Discord users with specific roles */
func ReadRoleList() bool {
	RoleListLock.Lock()
	defer RoleListLock.Unlock()

	_, err := os.Stat(constants.RoleListFile)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateRoleList()
		RoleList = newcfg

		_, err := os.Create(constants.RoleListFile)
		if err != nil {
			cwlog.DoLogCW("Could not create RoleList.")
			return false
		}
		return true
	} else { /* Otherwise just read in the config */
		file, err := ioutil.ReadFile(constants.RoleListFile)

		if file != nil && err == nil {
			newcfg := CreateRoleList()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				cwlog.DoLogCW("Readcfg.RoleList: Unmarshal failure")
				return false
			}

			cwlog.DoLogCW("Readcfg.RoleList: Successfully read.")
			RoleList = newcfg

			return true
		} else {
			cwlog.DoLogCW("Readcfg.RoleList: ReadFile failure")
			return false
		}
	}
}

/* Check with Discord, get updated list of users */
func UpdateRoleList() {
	RoleListLock.Lock()
	defer RoleListLock.Unlock()
	g := Guild

	if g != nil {

		foundChange := false

		for _, m := range g.Members {
			for _, r := range m.Roles {
				if r == cfg.Global.RoleData.NitroRoleID {
					foundN := false
					for _, u := range RoleList.NitroBooster {
						if u == m.User.Username {
							foundN = true
							break
						}
					}
					if !foundN {
						foundChange = true
						RoleList.NitroBooster = append(RoleList.NitroBooster, m.User.Username)
					}
				} else if r == cfg.Global.RoleData.PatreonRoleID {
					foundP := false
					for _, u := range RoleList.Patreons {
						if u == m.User.Username {
							foundP = true
							break
						}
					}
					if !foundP {
						foundChange = true
						RoleList.Patreons = append(RoleList.Patreons, m.User.Username)
					}
				} else if r == cfg.Global.RoleData.ModeratorRoleID {
					foundM := false
					for _, u := range RoleList.Moderators {
						if u == m.User.Username {
							foundM = true
							break
						}
					}
					if !foundM {
						foundChange = true
						RoleList.Moderators = append(RoleList.Moderators, m.User.Username)
					}
				}
			}
		}
		if foundChange {
			RoleListUpdated = true
			go WriteRoleList()
		}
	}
}
