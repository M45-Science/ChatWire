package fact

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"../config"
	"../constants"
	"../glob/"
	"../logs"
)

func GetMapTypeNum(mapt string) int {
	i := 0

	if config.Config.MapGenJson != "" {
		return 0
	}
	for i = 0; i < glob.MaxMapTypes; i = i + 1 {
		if strings.EqualFold(constants.MapTypes[i], mapt) {
			return i
		}
	}
	return -1
}

func GetMapTypeName(num int) string {

	if num < glob.MaxMapTypes && num >= 0 {
		return constants.MapTypes[num]
	}
	return "Error"
}

//Generate map
func Map_reset(data string) {

	if IsFactRunning() {
		if data != "" {
			CMS(config.Config.FactorioChannelID, data)
			WriteFact("/cchat " + data)
			return
		} else {
			CMS(config.Config.FactorioChannelID, "Stopping server, for map reset.")
			SetAutoStart(false)
			QuitFactorio()
		}
	}

	//Wait for server to stop if running
	for IsFactRunning() {
		time.Sleep(10 * time.Second)
	}

	glob.GameMapLock.Lock()
	defer glob.GameMapLock.Unlock()

	version := strings.Split(glob.FactorioVersion, ".")
	vlen := len(version)

	if vlen < 3 {
		logs.Log("Unable to determine factorio version.")
		return
	}

	if glob.GameMapPath != "" && glob.FactorioVersion != constants.Unknown {
		shortversion := strings.Join(version[0:2], ".")

		t := time.Now()
		date := fmt.Sprintf("%02d-%02d-%04d_%02d-%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute())
		newmapname := fmt.Sprintf("%s-%s.zip", config.Config.ChannelName, date)
		newmappath := fmt.Sprintf("%s/%s maps/%s", config.Config.MapArchivePath, shortversion, newmapname)
		newmapurl := fmt.Sprintf("http://m45sci.xyz/u/fact/old-maps/%s%smaps/%s", shortversion, "%20", newmapname)

		from, erra := os.Open(glob.GameMapPath)
		if erra != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to open the map to archive. Details: %s", erra))
			return
		}
		defer from.Close()

		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to create the archive map file. Details: %s", errb))
			return
		}
		defer to.Close()

		_, errc := io.Copy(to, from)
		if errc != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to write the archived map. Details: %s", errc))
			return
		}

		var buf string
		if erra == nil && errb == nil && errc == nil {
			buf = fmt.Sprintf("Map archived as: %s", newmapurl)
		} else {
			buf = "Map archive failed."
			return
		}

		CMS(config.Config.FactorioChannelID, buf)
	}

	t := time.Now()
	ourseed := uint64(t.UnixNano())

	MapPreset := config.Config.MapPreset

	if MapPreset == "Error" {
		CMS(config.Config.FactorioChannelID, "Invalid map preset.")
		return
	}

	CMS(config.Config.FactorioChannelID, "Generating map...")

	//Generate code to make filename
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, ourseed)
	ourcode := fmt.Sprintf("%02d%v", GetMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	filename := config.Config.FactorioLocation + "/saves/" + ourcode + ".zip"

	factargs := []string{"--preset", MapPreset, "--map-gen-seed", fmt.Sprintf("%v", ourseed), "--create", filename}

	//Append map gen if set
	if config.Config.MapGenJson != "" {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, config.Config.MapGenJson)
	}

	//Append map settings if set
	if config.Config.MapSetJson != "" {
		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, config.Config.MapSetJson)
	}

	cmd := exec.Command(config.Config.Executable, factargs...)
	_, aerr := cmd.CombinedOutput()

	if aerr != nil {
		logs.Log(fmt.Sprintf("An error occurred attempting to generate the map. Details: %s", aerr))
		return
	}
	CMS(config.Config.FactorioChannelID, "Rebooting.")
	DoExit()
	return

}
