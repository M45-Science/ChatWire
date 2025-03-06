package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/factUpdater"
	"ChatWire/util"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const (
	modPortalURL = "https://mods.factorio.com/api/mods/%v/full"
)

func CheckModUpdates(verbose bool) string {
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	//Read mods directory
	modList, err := os.ReadDir(modPath)
	if err != nil {
		return "CheckModUpdates: Unable to read mods dir: " + err.Error()
	}

	//Find all mods, read info.json inside
	var fileModList []modZipInfo
	for _, mod := range modList {
		if strings.HasSuffix(mod.Name(), ".zip") {
			modInfo := ReadModZipInfo(mod.Name())
			if modInfo == nil {
				continue
			}
			fileModList = append(fileModList, *modInfo)
		}
	}

	//Read mods-list.json
	jsonFileList, err := GetGameMods()
	if err != nil {
		return ("CheckModUpdates: Unable to mods list: " + err.Error())
	}

	//Check both lists, save any that are enabled so we have the mod version + details
	var installedMods []modZipInfo
	found := false
	for _, fmod := range fileModList {
		//Check if the mod is disabled
		for _, jmod := range jsonFileList.Mods {
			if IsBaseMod(jmod.Name) {
				continue
			}

			if jmod.Name == fmod.Name && jmod.Enabled {
				found = true
				installedMods = append(installedMods, fmod)
				break
			}
		}
		//Also include mods that are not in the mods-list.json (not disabled)
		if !found {
			installedMods = append(installedMods, fmod)
		}
	}

	if len(installedMods) == 0 {
		return "The game has no installed mods to update."
	}

	detailList := []modPortalFullData{}
	for _, item := range installedMods {
		URL := fmt.Sprintf(modPortalURL, item.Name)
		data, _, err := factUpdater.HttpGet(URL)
		if err != nil {
			cwlog.DoLogCW("Mod info request failed: " + err.Error())
			continue
		}
		newInfo := modPortalFullData{}
		err = json.Unmarshal(data, &newInfo)
		if err != nil {
			cwlog.DoLogCW("Mod info unmarshal failed: " + err.Error())
			continue
		}
		detailList = append(detailList, newInfo)
	}

	buf := ""
	for _, dItem := range detailList {
		for _, iItem := range installedMods {
			if dItem.Name == iItem.Name {
				found = false
				newestVersion := iItem.Version
				for _, release := range dItem.Releases {
					isNewer, err := newerVersion(newestVersion, release.Version)
					if err != nil {
						continue
					}
					if isNewer {
						newestVersion = release.Version
						found = true
					}
				}
				if found {
					if buf != "" {
						buf = buf + ", "
					}
					if verbose {
						buf = buf + iItem.Title + ": " + iItem.Version + " -> " + newestVersion
					} else {
						buf = buf + iItem.Name + "-" + newestVersion
					}
				}
			}
		}
	}

	if buf == "" {
		return "All installed mods are up to date."
	} else {
		return buf
	}
}

func ConfigGameMods(controlList []string, setState bool) (*modListData, error) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/" + constants.ModListName

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	serverMods := modListData{}
	err = json.Unmarshal(data, &serverMods)
	if err != nil {
		return nil, err
	}

	if len(controlList) > 0 {
		for s, serverMod := range serverMods.Mods {
			if strings.EqualFold(serverMod.Name, "base") {
				continue
			}
			for _, controlMod := range controlList {
				if strings.EqualFold(serverMod.Name, controlMod) {
					serverMods.Mods[s].Enabled = setState

					cwlog.DoLogCW(util.BoolToString(setState) + " " + serverMod.Name)
				}
			}
		}

		outbuf := new(bytes.Buffer)
		enc := json.NewEncoder(outbuf)
		enc.SetIndent("", "\t")

		if err := enc.Encode(serverMods); err != nil {
			return nil, err
		}

		err = os.WriteFile(path, outbuf.Bytes(), 0644)
		cwlog.DoLogCW("Wrote " + constants.ModListName)
	}
	return &serverMods, err
}
