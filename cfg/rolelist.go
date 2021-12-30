package cfg

import (
	"ChatWire/botlog"
	"ChatWire/constants"
	"ChatWire/glob"
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
)

func WriteRoleList() bool {
	glob.RoleListLock.Lock()
	defer glob.RoleListLock.Unlock()

	tempPath := constants.RoleListFile + "." + Local.ServerCallsign + ".tmp"
	finalPath := constants.RoleListFile

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	glob.RoleList.Version = "0.0.1"

	if err := enc.Encode(glob.RoleList); err != nil {
		botlog.DoLog("WriteRoleList: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("WriteRoleList: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("WriteRoleList: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		botlog.DoLog("Couldn't rename rolelist file.")
		return false
	}

	return true
}

func CreateRoleList() glob.RoleListData {
	newcfg := glob.RoleListData{Version: "0.0.1"}
	return newcfg
}

func ReadRoleList() bool {
	glob.RoleListLock.Lock()
	defer glob.RoleListLock.Unlock()

	_, err := os.Stat(constants.RoleListFile)
	notfound := os.IsNotExist(err)

	if notfound {
		botlog.DoLog("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateRoleList()
		glob.RoleList = newcfg

		_, err := os.Create(constants.RoleListFile)
		if err != nil {
			botlog.DoLog("Could not create rolelist.")
			return false
		}
		return true
	} else { //Otherwise just read in the config
		file, err := ioutil.ReadFile(constants.RoleListFile)

		if file != nil && err == nil {
			newcfg := CreateRoleList()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				botlog.DoLog("ReadRoleList: Unmashal failure")
				return false
			}

			botlog.DoLog("ReadRoleList: Successfully read.")
			glob.RoleList = newcfg

			return true
		} else {
			botlog.DoLog("ReadRoleList: ReadFile failure")
			return false
		}
	}
}

func UpdateRoleList() {
	glob.RoleListLock.Lock()
	defer glob.RoleListLock.Unlock()
	g := glob.Guild

	if g != nil {
		glob.RoleList.NitroBooster = []string{}
		glob.RoleList.Patreons = []string{}
		glob.RoleList.Moderators = []string{}

		for _, m := range g.Members {
			for _, r := range m.Roles {
				if r == Global.RoleData.NitroRoleID {
					glob.RoleList.NitroBooster = append(glob.RoleList.NitroBooster, m.User.Username)
				} else if r == Global.RoleData.PatreonRoleID {
					glob.RoleList.Patreons = append(glob.RoleList.Patreons, m.User.Username)
				} else if r == Global.RoleData.ModeratorRoleID {
					glob.RoleList.Moderators = append(glob.RoleList.Moderators, m.User.Username)
				}
			}
		}
		if len(glob.RoleList.Patreons) > 0 {
			go WriteRoleList()
		}
	}
}
