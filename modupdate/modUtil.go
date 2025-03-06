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

func GetModList() (modListData, error) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/" + constants.ModListName

	data, err := os.ReadFile(path)
	if err != nil {
		return modListData{}, err
	}

	serverMods := modListData{}
	err = json.Unmarshal(data, &serverMods)
	if err != nil {
		return modListData{}, err
	}

	return serverMods, nil
}

func ModInfoRead(modName string, rawData []byte) *modZipInfo {
	var err error
	if rawData == nil {
		path := cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			cfg.Global.Paths.Folders.Mods + "/" + modName

		rawData, err = os.ReadFile(path)
		if err != nil {
			cwlog.DoLogCW("ReadModZipInfo: " + err.Error())
			return nil
		}
	}
	jsonData := GetInfoJsonFromZip(rawData)

	modData := modZipInfo{}
	err = json.Unmarshal(jsonData, &modData)
	if err != nil {
		cwlog.DoLogCW("ReadModZipInfo: Unmarshal failure: " + err.Error())
		buf := fmt.Sprintf("%v", modData)
		cwlog.DoLogCW(buf)
		return nil
	}

	return &modData
}

func GetInfoJsonFromZip(data []byte) []byte {
	// Create a reader from the byte array
	byteReader := bytes.NewReader(data)

	// Create a zip reader
	zipReader, err := zip.NewReader(byteReader, int64(len(data)))
	if err != nil {
		return nil
	}

	for _, file := range zipReader.File {
		if strings.HasSuffix(file.Name, "info.json") {
			if strings.Count(file.Name, "/") < 2 {
				f, err := file.Open()
				if err != nil {
					return nil
				}
				defer f.Close()

				var buf bytes.Buffer
				_, err = io.Copy(&buf, f)
				if err != nil {
					return nil
				}

				return buf.Bytes()
			}
		}
	}

	return nil
}
