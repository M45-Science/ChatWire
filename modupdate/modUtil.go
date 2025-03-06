package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func checkVersion(eo int, currentStr, remoteStr string) (bool, error) {

	cInt, err := versionToInt(currentStr)
	if err != nil {
		return false, err
	}
	rInt, err := versionToInt(remoteStr)
	if err != nil {
		return false, err
	}

	// Compare major versions
	if rInt.parts[0] != cInt.parts[0] {
		return compareVersions(eo, cInt.parts[0], rInt.parts[0])
	}

	// Compare minor versions
	if rInt.parts[1] != cInt.parts[1] {
		return compareVersions(eo, cInt.parts[1], rInt.parts[1])
	}

	// Compare patch versions
	if rInt.parts[2] != cInt.parts[2] {
		return compareVersions(eo, cInt.parts[2], rInt.parts[2])
	}

	// If they are equal
	return compareVersions(eo, 0, 0)
}

// Helper function to compare based on eo
func compareVersions(eo, current, remote int) (bool, error) {
	switch eo {
	case EO_LESS:
		return current < remote, nil
	case EO_LESSEQ:
		return current <= remote, nil
	case EO_EQUAL:
		return current == remote, nil
	case EO_GREATEREQ:
		return current >= remote, nil
	case EO_GREATER:
		return current > remote, nil
	default:
		return false, fmt.Errorf("invalid comparison operation")
	}
}

func versionToInt(data string) (intVersion, error) {
	parts := strings.Split(data, ".")
	numParts := len(parts)
	//For 2 digit versions
	if numParts == 2 {
		data = data + ".0"
		numParts++
	}

	var intOut intVersion
	if numParts != 3 {
		return intVersion{}, fmt.Errorf("malformed version string: " + data)
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
	if strings.HasPrefix(modName, "base") ||
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

func ParseEquality(input string) int {
	switch input {
	case "<":
		return EO_LESS
	case "<=":
		return EO_LESSEQ
	case "=":
		return EO_EQUAL
	case ">=":
		return EO_GREATEREQ
	case ">":
		return EO_GREATER
	default:
		return EO_ERROR
	}
}

func CheckSHA1(data []byte, checkHash string) bool {

	hash := sha1.New()
	hash.Write([]byte(data))
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return strings.EqualFold(hashString, checkHash)
}
