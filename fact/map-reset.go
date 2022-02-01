package fact

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/sclean"
)

func GetMapTypeNum(mapt string) int {
	i := 0

	if cfg.Local.MapGenPreset != "" {
		return 0
	}
	for i = 0; i < len(constants.MapTypes); i = i + 1 {
		if strings.EqualFold(constants.MapTypes[i], mapt) {
			return i
		}
	}
	return -1
}

func GetMapTypeName(num int) string {

	numMaps := len(constants.MapTypes)
	if num >= 0 && num < numMaps {
		return constants.MapTypes[num]
	}
	return "Error"
}

/* Generate map */
func Map_reset(data string) {

	/* Prevent another map reset from accidentally running at the same time */
	GameMapLock.Lock()
	defer GameMapLock.Unlock()

	/* If Factorio is running, and there is a argument... echo it
	 * Otherwise, stop Factorio and generate a new map */
	if IsFactRunning() {
		if data != "" {
			CMS(cfg.Local.ChannelData.ChatID, sclean.EscapeDiscordMarkdown(data))
			WriteFact("/cchat " + AddFactColor("red", data))
			return
		} else {
			CMS(cfg.Local.ChannelData.ChatID, "Stopping server, for map reset.")

			cfg.Local.SlowConnect.DefaultSpeed = 1.0
			cfg.Local.SlowConnect.ConnectSpeed = 0.5
			SetAutoStart(false)
			QuitFactorio()
		}
	}

	/* Wait for server to stop if running */
	for x := 0; x < constants.MaxFactorioCloseWait && IsFactRunning(); x++ {
		time.Sleep(time.Millisecond * 100)
	}

	/* Get Factorio version, for archive folder name */
	version := strings.Split(FactorioVersion, ".")
	vlen := len(version)
	if vlen < 3 {
		cwlog.DoLogCW("Unable to determine Factorio version.")
		return
	}

	/* Only proceed if we were running a map, and we know our Factorio version. */
	if GameMapPath != "" && FactorioVersion != constants.Unknown {
		shortversion := strings.Join(version[0:2], ".")

		t := time.Now()
		date := t.Format("2006-01-02")
		newmapname := fmt.Sprintf("%v-%v.zip", sclean.AlphaNumOnly(cfg.Local.ServerCallsign)+"-"+cfg.Local.Name, date)
		newmappath := fmt.Sprintf("%v%v maps/%v", cfg.Global.PathData.MapArchivePath, shortversion, newmapname)
		newmapurl := fmt.Sprintf("%v%v/%v", cfg.Global.PathData.ArchiveURL, url.PathEscape(shortversion+" maps"), url.PathEscape(newmapname))

		from, erra := os.Open(GameMapPath)
		if erra != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to open the map to archive. Details: %s", erra))
			return
		}
		defer from.Close()

		/* Make directory if it does not exist */
		newdir := fmt.Sprintf("%v%v maps/", cfg.Global.PathData.MapArchivePath, shortversion)
		err := os.MkdirAll(newdir, os.ModePerm)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}

		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the archive map file. Details: %s", errb))
			return
		}
		defer to.Close()

		_, errc := io.Copy(to, from)
		if errc != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to write the archived map. Details: %s", errc))
			return
		}

		var buf string
		if erra == nil && errb == nil && errc == nil {
			buf = fmt.Sprintf("Map archived as: %s", newmapurl)
			CMS(cfg.Local.ChannelData.ChatID, buf)
		} else {
			buf = "Map archive failed."
			CMS(cfg.Local.ChannelData.ChatID, buf)
			return
		}
	}

	t := time.Now()
	ourseed := uint64(t.UnixNano())

	MapPreset := cfg.Local.MapPreset

	if MapPreset == "Error" {
		CMS(cfg.Local.ChannelData.ChatID, "Invalid map preset.")
		return
	}

	CMS(cfg.Local.ChannelData.ChatID, "Generating map...")
	/* Delete old sav-* map to save space */
	DeleteOldSav()

	/* Generate code to make filename */
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, ourseed)
	ourcode := fmt.Sprintf("%02d%v", GetMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	filename := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath + "/gen-" + ourcode + ".zip"

	factargs := []string{"--map-gen-seed", fmt.Sprintf("%v", ourseed), "--create", filename}

	/* Append map gen if set */
	if cfg.Local.MapGenPreset != "" {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.MapGenPath+"/"+cfg.Local.MapGenPreset+"-gen.json")

		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.MapGenPath+"/"+cfg.Local.MapGenPreset+"-set.json")
	} else {
		factargs = append(factargs, "--preset")
		factargs = append(factargs, MapPreset)
	}

	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", GetFactorioBinary(), strings.Join(factargs, " "))
	cwlog.DoLogCW(lbuf)

	cmd := exec.Command(GetFactorioBinary(), factargs...)
	_, aerr := cmd.CombinedOutput()

	if aerr != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred attempting to generate the map. Details: %s", aerr))
		return
	}
	CMS(cfg.Local.ChannelData.ChatID, "Rebooting.")

	/* If available, use per-server ping setting... otherwise use global */
	pingstr := ""
	if cfg.Local.ResetPingString != "" {
		pingstr = cfg.Local.ResetPingString
	} else if cfg.Global.ResetPingString != "" {
		pingstr = cfg.Global.ResetPingString
	}
	CMS(cfg.Global.DiscordData.AnnounceChannelID, pingstr+" Map on server: "+cfg.Local.ServerCallsign+"-"+cfg.Local.Name+" has been reset.")

	/* Mods queue folder */
	qPath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" +
		constants.ModsQueueFolder + "/"
	modPath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" +
		constants.ModsFolder + "/"
	files, err := ioutil.ReadDir(qPath)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	_, err = os.Stat(qPath)
	notfound := os.IsNotExist(err)

	if notfound {
		_, err = os.Create(qPath)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	} else {
		for _, f := range files {
			if f.Name() == "mod-settings.dat" {
				err := os.Rename(qPath+f.Name(), modPath+f.Name())
				if err != nil {
					cwlog.DoLogCW(err.Error())
				}
			}

			if strings.HasSuffix(f.Name(), ".zip") {

				/* Delete mods queued up to be deleted */
				if strings.HasPrefix(f.Name(), "deleteme-") {

					err = os.Remove(modPath + strings.TrimPrefix(f.Name(), "deleteme-"))
					if err != nil {
						cwlog.DoLogCW(err.Error())
					}
					err = os.Remove(qPath + f.Name())
					if err != nil {
						cwlog.DoLogCW(err.Error())
					}
				} else {

					/* Otherwise, install new mod */
					err := os.Rename(qPath+f.Name(), modPath+f.Name())
					if err != nil {
						cwlog.DoLogCW(err.Error())
					}
				}
			}
		}
	}

	glob.VoteBoxLock.Lock()
	glob.VoteBox.LastRewindTime = time.Now()
	VoidAllVotes()     /* Void all votes */
	ResetTotalVotes()  /* New map, reset player's vote limits */
	WriteRewindVotes() /* Save to file before exiting */
	glob.VoteBoxLock.Unlock()
	DoExit(true)
}
