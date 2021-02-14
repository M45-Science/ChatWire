package fact

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"../cfg"
	"../constants"
	"../glob/"
	"../logs"
)

func GetMapTypeNum(mapt string) int {
	i := 0

	if cfg.Local.MapGenPreset != "" {
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

	newstr := data

	//Remove factorio tags
	rega := regexp.MustCompile(`\[/[^][]+\]`) //remove close tags [/color]

	regc := regexp.MustCompile(`\[color=(.*?)\]`) //remove [color=*]
	regd := regexp.MustCompile(`\[font=(.*?)\]`)  //remove [font=*]

	regf := regexp.MustCompile(`\*+`) //Remove discord markdown
	regg := regexp.MustCompile(`\~+`)
	regh := regexp.MustCompile(`\_+`)

	newstr = strings.ReplaceAll(newstr, "\n", "") //replace newline
	newstr = strings.ReplaceAll(newstr, "\r", "") //replace return

	for regc.MatchString(newstr) || regd.MatchString(newstr) {
		//Remove colors/fonts
		newstr = regc.ReplaceAllString(newstr, "")
		newstr = regd.ReplaceAllString(newstr, "")
	}
	for rega.MatchString(newstr) {
		//Filter close tags
		newstr = rega.ReplaceAllString(newstr, "")
	}
	for regf.MatchString(newstr) || regg.MatchString(newstr) || regh.MatchString(newstr) {
		//Filter discord tags
		newstr = regf.ReplaceAllString(newstr, "")
		newstr = regg.ReplaceAllString(newstr, "")
		newstr = regh.ReplaceAllString(newstr, "")
	}

	if IsFactRunning() {
		if newstr != "" {
			CMS(cfg.Local.ChannelData.ChatID, newstr)
			WriteFact("/cchat [color=red](SYSTEM) " + newstr + "[/color]")
			return
		} else {
			CMS(cfg.Local.ChannelData.ChatID, "Stopping server, for map reset.")
			SetAutoStart(false)
			QuitFactorio()
		}
	}

	//Wait for server to stop if running
	for IsFactRunning() {
		time.Sleep(1 * time.Second)
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
		newmapname := fmt.Sprintf("%s-%s.zip", cfg.Local.ServerCallsign+"-"+cfg.Local.Name, date)
		newmappath := fmt.Sprintf("%s%s maps/%s", cfg.Global.PathData.MapArchivePath, shortversion, newmapname)
		newmapurl := fmt.Sprintf("%v%s%smaps/%s", cfg.Global.PathData.ArchiveURL, shortversion, "%20", newmapname)

		from, erra := os.Open(glob.GameMapPath)
		if erra != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to open the map to archive. Details: %s", erra))
			return
		}
		defer from.Close()

		//Make directory if it does not exist
		newdir := fmt.Sprintf("%s%s maps/", cfg.Global.PathData.MapArchivePath, shortversion)
		os.MkdirAll(newdir, os.ModePerm)

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

		CMS(cfg.Local.ChannelData.ChatID, buf)
	}

	t := time.Now()
	ourseed := uint64(t.UnixNano())

	MapPreset := cfg.Local.MapPreset

	if MapPreset == "Error" {
		CMS(cfg.Local.ChannelData.ChatID, "Invalid map preset.")
		return
	}

	CMS(cfg.Local.ChannelData.ChatID, "Generating map...")

	//Generate code to make filename
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, ourseed)
	ourcode := fmt.Sprintf("%02d%v", GetMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	filename := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + "/saves/" + ourcode + ".zip"

	factargs := []string{"--preset", MapPreset, "--map-gen-seed", fmt.Sprintf("%v", ourseed), "--create", filename}

	//Append map gen if set
	if cfg.Local.MapGenPreset != "" {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, cfg.Global.PathData.MapGenPath+cfg.Local.MapGenPreset+"-set.json")
	}

	//Append map settings if set
	if cfg.Local.MapGenPreset != "" {
		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, cfg.Global.PathData.MapGenPath+cfg.Local.MapGenPreset+"-gen.json")
	}

	cmd := exec.Command(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactorioHomePrefix+cfg.Local.ServerCallsign+"/"+cfg.Global.PathData.FactorioBinary, factargs...)
	_, aerr := cmd.CombinedOutput()

	if aerr != nil {
		logs.Log(fmt.Sprintf("An error occurred attempting to generate the map. Details: %s", aerr))
		return
	}
	CMS(cfg.Local.ChannelData.ChatID, "Rebooting.")
	CMS(cfg.Global.DiscordData.AnnounceChannelID, "Map on server: "+cfg.Local.ServerCallsign+"-"+cfg.Local.Name+" has been reset.")
	DoExit()
	return

}
