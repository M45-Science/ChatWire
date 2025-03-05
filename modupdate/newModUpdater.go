package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/util"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const modPortalURL = "https://mods.factorio.com/api/mods/%v/full"

func CheckModUpdates() {
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	//Read mods directory
	modList, err := os.ReadDir(modPath)
	if err != nil {
		cwlog.DoLogCW("CheckModUpdates: Unable to read mods dir: " + err.Error())
		return
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
		cwlog.DoLogCW("CheckModUpdates: Unable to mods list: " + err.Error())
		return
	}

	//Check both lists, save any that are enabled so we have the mod version + details
	var finalModList []modZipInfo
	for _, jmod := range jsonFileList.Mods {
		if IsBaseMod(jmod.Name) {
			continue
		}
		for _, fmod := range fileModList {
			if jmod.Name == fmod.Name && jmod.Enabled {
				finalModList = append(finalModList, fmod)
			}
		}
	}

	//Print out data
	var buf string
	for _, item := range finalModList {
		if buf != "" {
			buf = buf + ", "
		}
		buf = buf + fmt.Sprintf("%v: %v", item.Name, item.Version)
	}

	if buf == "" {
		buf = "No enabled game mods found."
	}
	cwlog.DoLogCW(buf)
}

func GetInfoJSONFromZip(zipFilePath string) ([]byte, error) {
	// Open the zip file
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("GetInfoJSONFromZip:  to open zip file: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "info.json") {
			f, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("GetInfoJSONFromZip: failed to open file in zip: %w", err)
			}
			defer f.Close()

			var buf bytes.Buffer
			_, err = io.Copy(&buf, f)
			if err != nil {
				return nil, fmt.Errorf("GetInfoJSONFromZip: failed to read file content: %w", err)
			}

			return buf.Bytes(), nil
		}
	}

	return nil, fmt.Errorf("GetInfoJSONFromZip: info.json not found in: " + zipFilePath)
}

func ReadModZipInfo(modName string) *modZipInfo {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/" + modName

	data, err := GetInfoJSONFromZip(path)
	if err != nil {
		cwlog.DoLogCW("ReadModZipInfo: " + err.Error())
		return nil
	}

	modData := modZipInfo{}
	err = json.Unmarshal(data, &modData)
	if err != nil {
		cwlog.DoLogCW("ReadModZipInfo: Unmarshal failure: " + err.Error())
		return nil
	}

	return &modData
}

func IsBaseMod(modName string) bool {
	if strings.EqualFold(modName, "base") ||
		strings.EqualFold(modName, "elevated-rails") ||
		strings.EqualFold(modName, "quality") ||
		strings.EqualFold(modName, "space-age") {
		return true
	}
	return false
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

func GetGameMods() (*modListData, error) {
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

	return &serverMods, nil
}
