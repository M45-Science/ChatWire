package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func newerVersion(currentStr, remoteStr string) (bool, error) {

	cInt, err := versionToInt(currentStr)
	if err != nil {
		return false, err
	}
	rInt, err := versionToInt(remoteStr)
	if err != nil {
		return false, err
	}

	//Major
	if rInt.parts[0] > cInt.parts[0] {
		return true, nil
	} else if rInt.parts[0] < cInt.parts[0] {
		return false, nil
	}

	//Minor
	if rInt.parts[1] > cInt.parts[1] {
		return true, nil
	} else if rInt.parts[1] < cInt.parts[1] {
		return false, nil
	}

	//Patch
	if rInt.parts[2] > cInt.parts[2] {
		return true, nil
	} else if rInt.parts[1] < cInt.parts[1] {
		return false, nil
	}

	return false, nil
}

func versionToInt(data string) (intVersion, error) {
	parts := strings.Split(data, ".")

	var intOut intVersion
	if len(parts) != 3 {
		return intVersion{}, fmt.Errorf("malformed version string")
	}
	for p, part := range parts {
		val, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return intVersion{}, fmt.Errorf("failed to parse version string")
		}
		intOut.parts[p] = int(val)
	}
	return intOut, nil
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
func GetInfoJSONFromZip(zipFilePath string) ([]byte, error) {
	// Open the zip file
	zipReader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("GetInfoJSONFromZip:  to open zip file: %w", err)
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "info.json") {
			if strings.Count(file.Name, "/") < 2 {
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
	}

	return nil, fmt.Errorf("GetInfoJSONFromZip: info.json not found in: " + zipFilePath)
}
