package disc

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync"
)

/* Discord role member-lists */
var RoleListLock sync.Mutex
var RoleList RoleListData

//Cache a list of users with specific Discord roles
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
		botlog.DoLog("Writecfg.RoleList: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("Writecfg.RoleList: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("Writecfg.RoleList: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		botlog.DoLog("Couldn't rename RoleList file.")
		return false
	}

	return true
}

//Create a new RoleList
func CreateRoleList() RoleListData {
	newcfg := RoleListData{Version: "0.0.1"}
	return newcfg
}

//Read in cached list of Discord users with specific roles
func ReadRoleList() bool {
	RoleListLock.Lock()
	defer RoleListLock.Unlock()

	_, err := os.Stat(constants.RoleListFile)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateRoleList()
		RoleList = newcfg

		_, err := os.Create(constants.RoleListFile)
		if err != nil {
			botlog.DoLog("Could not create RoleList.")
			return false
		}
		return true
	} else { //Otherwise just read in the config
		file, err := ioutil.ReadFile(constants.RoleListFile)

		if file != nil && err == nil {
			newcfg := CreateRoleList()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				botlog.DoLog("Readcfg.RoleList: Unmashal failure")
				return false
			}

			botlog.DoLog("Readcfg.RoleList: Successfully read.")
			RoleList = newcfg

			return true
		} else {
			botlog.DoLog("Readcfg.RoleList: ReadFile failure")
			return false
		}
	}
}

//Check with Discord, get updated list of users
func UpdateRoleList() {
	RoleListLock.Lock()
	defer RoleListLock.Unlock()
	g := Guild

	if g != nil {
		RoleList.NitroBooster = []string{}
		RoleList.Patreons = []string{}
		RoleList.Moderators = []string{}

		for _, m := range g.Members {
			for _, r := range m.Roles {
				if r == cfg.Global.RoleData.NitroRoleID {
					RoleList.NitroBooster = append(RoleList.NitroBooster, m.User.Username)
				} else if r == cfg.Global.RoleData.PatreonRoleID {
					RoleList.Patreons = append(RoleList.Patreons, m.User.Username)
				} else if r == cfg.Global.RoleData.ModeratorRoleID {
					RoleList.Moderators = append(RoleList.Moderators, m.User.Username)
				}
			}
		}
		if len(RoleList.Patreons) > 0 {
			go WriteRoleList()
		}
	}
}
