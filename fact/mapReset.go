package fact

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/util"
)

func getMapTypeNum(mapt string) int {
	i := 0

	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
		return 0
	}
	for i = 0; i < len(constants.MapTypes); i = i + 1 {
		if strings.EqualFold(constants.MapTypes[i], mapt) {
			return i
		}
	}
	return -1
}

func getMapTypeName(num int) string {

	numMaps := len(constants.MapTypes)
	if num >= 0 && num < numMaps {
		return constants.MapTypes[num]
	}
	return "Error"
}

/* Generate map */
func Map_reset(doReport bool) {

	/* If Factorio is running, and there is a argument... echo it
	 * Otherwise, stop Factorio and generate a new map */
	if FactorioBooted || FactIsRunning {
		QueueReboot = false      //Skip queued reboot
		QueueFactReboot = false  //Skip queued fact reboot
		DoUpdateFactorio = false //Skip queued updates

		SetAutolaunch(false, false)
		QuitFactorio("Server rebooting for map reset!")
	} else {
		return
	}

	/* Wait for server to stop if running */
	WaitFactQuit(false)

	/* Only proceed if we were running a map, and we know our Factorio version. */
	if GameMapPath != "" && FactorioVersion != constants.Unknown {
		quickArchive()
	}

	GenNewMap()

	/* If available, use per-server ping setting... otherwise use global */
	pingstr := ""
	if cfg.Local.Options.ResetPingRole != "" {
		pingstr = fmt.Sprintf("<@&%v>", cfg.Local.Options.ResetPingRole)
	} else if cfg.Global.Options.ResetPingRole != "" {
		pingstr = fmt.Sprintf("<@&%v>", cfg.Global.Options.ResetPingRole)
	}
	LogGameCMS(false, cfg.Global.Discord.AnnounceChannel, pingstr+" Map "+cfg.Local.Callsign+"-"+cfg.Local.Name+" auto-reset.")

	/* Mods queue folder */
	qPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsQueueFolder + "/"

	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	files, err := os.ReadDir(qPath)
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
			if strings.EqualFold(f.Name(), constants.ModSettingsName) {
				err := os.Rename(qPath+f.Name(), modPath+f.Name())
				if err != nil {
					cwlog.DoLogCW(err.Error())
				} else {
					buf := "Installed new mod-settings.dat"
					LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
				}
			}

			if strings.HasSuffix(f.Name(), ".zip") {

				/* Delete mods queued up to be deleted */
				if strings.HasPrefix(f.Name(), "deleteme-") {

					delModName := f.Name()
					err = os.Remove(qPath + delModName)
					if err != nil {

						modName := strings.TrimPrefix(delModName, "deleteme-")
						err = os.Remove(modPath + modName)
						if err != nil {
							buf := fmt.Sprintf("Failed to remove mod: %v", modName)
							LogCMS(cfg.Local.Channel.ChatChannel, buf)
						} else {
							buf := fmt.Sprintf("Removed mod: %v", modName)
							LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
						}
					} else {
						buf := "Mod queue: incorrect file permissions."
						LogCMS(cfg.Local.Channel.ChatChannel, buf)
					}
				} else {

					/* Otherwise, install new mod */
					err := os.Rename(qPath+f.Name(), modPath+f.Name())
					if err != nil {
						msg := fmt.Sprintf("Unable to install mod: %v", f.Name())
						LogCMS(cfg.Local.Channel.ChatChannel, msg)

					} else {
						buf := fmt.Sprintf("Installed mod: %v", f.Name())
						LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
					}
				}
			}
		}
	}

	glob.VoteBox.LastMapChange = time.Now()
	VoidAllVotes() /* Void all votes */
	WriteVotes()

	SetAutolaunch(true, false)
}

func GenNewMap() string {
	SetResetDate()

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	cfg.Local.Options.SkipReset = false //Turn off skip reset
	cfg.WriteLCfg()

	genpath := util.GetSavesFolder()
	flist, err := filepath.Glob(genpath + "/gen-*.zip")
	if err != nil {
		panic(err)
	}
	for _, f := range flist {
		if err := os.Remove(f); err != nil {
			cwlog.DoLogCW("Failed to delete: " + f)
		}
	}

	t := time.Now()
	ourseed := int(t.UnixNano() - constants.CWEpoch)
	cfg.Local.Options.Speed = 1
	haveSeed := false

	//Use seed if specified, then clear it
	if cfg.Local.Settings.Seed > 0 {
		haveSeed = true
		ourseed = cfg.Local.Settings.Seed
		cfg.Local.Settings.Seed = 0
		cfg.WriteLCfg()

		msg := fmt.Sprintf("Using custom map seed: %v", cfg.Local.Settings.Seed)
		LogGameCMS(false, cfg.Local.Channel.ChatChannel, msg)
	}

	MapPreset := cfg.Local.Settings.MapPreset

	if strings.EqualFold(MapPreset, "error") {
		cwlog.DoLogCW("Invalid map preset.")
		return "Invalid map preset"
	}

	/* Generate code to make filename */
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, uint64(ourseed))
	ourcode := fmt.Sprintf("%02d%v", getMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	sName := "gen-" + ourcode + ".zip"

	filename := util.GetSavesFolder() +
		"/" + sName
	factargs := []string{"--create", filename}

	if haveSeed { //If we have a custom seed, use it, otherwise let factorio randomize
		factargs = append(factargs, "--map-gen-seed", fmt.Sprintf("%v", ourseed))
	}

	/* Append map gen if set */
	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-gen.json")

		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-set.json")
	} else {
		factargs = append(factargs, "--preset")
		factargs = append(factargs, MapPreset)
	}

	if cfg.Local.Settings.Scenario != "" || strings.EqualFold(cfg.Local.Settings.Scenario, "none") {
		cfg.Local.Settings.NewMap = true
	}

	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", GetFactorioBinary(), strings.Join(factargs, " "))
	cwlog.DoLogCW(lbuf)

	cmd := exec.Command(GetFactorioBinary(), factargs...)
	_, aerr := cmd.CombinedOutput()

	if aerr != nil {
		buf := fmt.Sprintf("An error occurred attempting to generate the map: %s", aerr)
		cwlog.DoLogCW(buf)
		LogCMS(cfg.Local.Channel.ChatChannel, buf)
		return aerr.Error()
	}

	return sName
}

func quickArchive() {
	version := strings.Split(FactorioVersion, ".")
	shortversion := strings.Join(version[0:2], ".")

	t := time.Now()
	date := t.Format("2006-01-02")
	newmapname := fmt.Sprintf("%v-%v.zip", cfg.Local.Callsign+"-"+cfg.Local.Name, date)
	newmappath := fmt.Sprintf("%v%v%v%v%v", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix, "/", newmapname)
	newmapurl := fmt.Sprintf("https://%v%v%v%v%v%v",
		cfg.Global.Paths.URLs.Domain,
		cfg.Global.Paths.URLs.PathPrefix,
		cfg.Global.Paths.URLs.ArchivePath,
		url.PathEscape(shortversion+constants.ArchiveFolderSuffix),
		"/",
		url.PathEscape(newmapname))

	from, erra := os.Open(GameMapPath)
	if erra != nil {

		buf := fmt.Sprintf("An error occurred when attempting to read the map to archive: %s", erra)
		cwlog.DoLogCW(cfg.Local.Channel.ChatChannel, buf)
		return
	}
	defer from.Close()

	/* Attach map, send to chat */
	dData := &discordgo.MessageSend{Files: []*discordgo.File{
		{Name: newmapname, Reader: from, ContentType: "application/zip"}}}
	if disc.DS != nil {
		_, err := disc.DS.ChannelMessageSendComplex(cfg.Local.Channel.ChatChannel, dData)

		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}

	_, err := from.Seek(0, io.SeekStart)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	/* Make directory if it does not exist */
	newdir := fmt.Sprintf("%v%v%v/", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix)
	err = os.MkdirAll(newdir, os.ModePerm)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
	if errb != nil {
		buf := fmt.Sprintf("An error occurred when attempting to create the map archive file: %s", errb)
		LogCMS(cfg.Local.Channel.ChatChannel, buf)
		return
	}
	defer to.Close()

	_, errc := io.Copy(to, from)
	if errc != nil {
		buf := fmt.Sprintf("An error occurred when attempting to write the map archive file: %s", errc)
		LogCMS(cfg.Local.Channel.ChatChannel, buf)
		return
	}

	buf := fmt.Sprintf("Map archived as: %s", newmapurl)
	LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
}
