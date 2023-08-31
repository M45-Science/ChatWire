package disc

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
)

/* Discord role member-lists */
var RoleList RoleListData
var RoleListUpdated bool

/* Cache a list of players with specific Discord roles */
func WriteRoleList() bool {

	tempPath := constants.RoleListFile + "." + cfg.Local.Callsign + ".tmp"
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

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

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

/* Read in cached list of Discord players with specific roles */
func ReadRoleList() bool {

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
		file, err := os.ReadFile(constants.RoleListFile)

		if file != nil && err == nil {
			newcfg := CreateRoleList()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				cwlog.DoLogCW("Readcfg.RoleList: Unmarshal failure")
				return false
			}

			//cwlog.DoLogCW("Readcfg.RoleList: Successfully read.")
			RoleList = newcfg

			return true
		} else {
			cwlog.DoLogCW("Readcfg.RoleList: ReadFile failure")
			return false
		}
	}
}

/* Check with Discord, get updated list of players */
func UpdateRoleList() {
	g := Guild

	if g != nil {

		foundChange := false

		for _, m := range g.Members {
			for _, r := range m.Roles {
				if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Nitro) {
					foundN := false
					for _, u := range RoleList.NitroBooster {
						if strings.EqualFold(u, m.User.Username) {
							foundN = true
							break
						}
					}
					if !foundN {
						foundChange = true
						RoleList.NitroBooster = append(RoleList.NitroBooster, m.User.Username)
					}
				} else if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Patreon) {
					foundP := false
					for _, u := range RoleList.Patreons {
						if strings.EqualFold(u, m.User.Username) {
							foundP = true
							break
						}
					}
					if !foundP {
						foundChange = true
						RoleList.Patreons = append(RoleList.Patreons, m.User.Username)
					}
				} else if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Supporter) {
					foundP := false
					for _, u := range RoleList.Supporters {
						if strings.EqualFold(u, m.User.Username) {
							foundP = true
							break
						}
					}
					if !foundP {
						foundChange = true
						RoleList.Supporters = append(RoleList.Supporters, m.User.Username)
					}
				} else if strings.EqualFold(r, cfg.Global.Discord.Roles.RoleCache.Moderator) {
					foundM := false
					for _, u := range RoleList.Moderators {
						if strings.EqualFold(u, m.User.Username) {
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
			WriteRoleList()
		}
	}
}
