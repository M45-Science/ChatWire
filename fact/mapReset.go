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
)

func getMapTypeNum(mapt string) int {
	i := 0

	for i = 0; i < len(constants.MapTypes); i = i + 1 {
		if strings.EqualFold(constants.MapTypes[i], mapt) {
			return i
		}
	}
	return -1
}

/* Generate map */
func Map_reset(doReport bool) error {
	SetAutolaunch(false, false)
	return submitLifecycleRequestAndWait(Request{
		Kind:      ActionMapReset,
		Reason:    "Server rebooting for map reset!",
		RequestID: fmt.Sprintf("map-reset-%d", time.Now().UnixNano()),
	})
}

func mapResetAfterStop(doReport bool) error {
	/* Only proceed if we were running a map, and we know our Factorio version. */
	if GameMapPath != "" && FactorioVersion != constants.Unknown {
		quickArchive()
	}

	if _, err := GenNewMap(); err != nil {
		msg := fmt.Sprintf("Map reset failed: %v", err)
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		return err
	}

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

	_, err := os.Stat(qPath)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(qPath, os.ModePerm); err != nil {
				cwlog.DoLogCW(err.Error())
			}
		} else {
			cwlog.DoLogCW(err.Error())
		}
	} else {
		files, err := os.ReadDir(qPath)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
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
	VoidAllVotes()
	WriteVotes()
	return nil
}

type mapCreateSettings struct {
	preset             string
	generator          string
	mapGenSettingsPath string
	mapSettingsPath    string
}

func (s mapCreateSettings) usesGenerator() bool {
	return s.generator != "" && s.mapGenSettingsPath != "" && s.mapSettingsPath != ""
}

func resolveMapCreateSettings(mapGenerator string, mapPreset string) mapCreateSettings {
	settings := mapCreateSettings{}

	if preset, ok := normalizeMapPreset(mapPreset); ok {
		settings.preset = preset
	} else if strings.TrimSpace(mapPreset) != "" {
		cwlog.DoLogCW("GenNewMap: invalid map preset %q, using Factorio default map creation.", mapPreset)
	}

	generator := strings.TrimSpace(mapGenerator)
	if generator == "" || strings.EqualFold(generator, "none") {
		return settings
	}

	genSettingsPath, mapSettingsPath := cfg.GetMapGeneratorFiles(generator)
	if !pathExists(genSettingsPath) || !pathExists(mapSettingsPath) {
		cwlog.DoLogCW("GenNewMap: map generator %q is unavailable; generating without map generator. map-gen-settings=%q map-settings=%q", generator, genSettingsPath, mapSettingsPath)
		return settings
	}

	settings.generator = generator
	settings.mapGenSettingsPath = genSettingsPath
	settings.mapSettingsPath = mapSettingsPath
	return settings
}

func normalizeMapPreset(mapPreset string) (string, bool) {
	mapPreset = strings.TrimSpace(mapPreset)
	if mapPreset == "" {
		return "", false
	}

	for _, preset := range constants.MapTypes {
		if strings.EqualFold(preset, mapPreset) {
			return preset, true
		}
	}
	return "", false
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func buildNewMapArgs(filename string, haveSeed bool, seed int, settings mapCreateSettings) []string {
	factargs := []string{"--create", filename}

	if haveSeed {
		factargs = append(factargs, "--map-gen-seed", fmt.Sprintf("%v", seed))
	}

	if settings.usesGenerator() {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, settings.mapGenSettingsPath)

		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, settings.mapSettingsPath)
	} else if settings.preset != "" {
		factargs = append(factargs, "--preset")
		factargs = append(factargs, settings.preset)
	}

	return factargs
}

func GenNewMap() (string, error) {
	SetResetDate()

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	cfg.Local.Options.SkipReset = false //Turn off skip reset
	cfg.WriteLCfg()

	genpath := cfg.GetSavesFolder()
	flist, err := filepath.Glob(genpath + "/gen-*.zip")
	if err != nil {
		cwlog.DoLogCW(fmt.Sprintf("mapReset: failed to list generated maps: %v", err))
		return "", err
	}
	for _, f := range flist {
		if err := os.Remove(f); err != nil {
			cwlog.DoLogCW("Failed to delete: " + f)
		}
	}

	t := time.Now()
	ourseed := int(t.UnixNano() - constants.CWEpoch)
	cfg.Local.Options.Speed = 1
	cfg.Local.Settings.AutoPause = true
	haveSeed := false

	//Use seed if specified, then clear it
	if cfg.Local.Settings.Seed > 0 {
		haveSeed = true
		origSeed := cfg.Local.Settings.Seed
		ourseed = origSeed
		cfg.Local.Settings.Seed = 0
		cfg.WriteLCfg()

		msg := fmt.Sprintf("Using custom map seed: %v", origSeed)
		LogGameCMS(false, cfg.Local.Channel.ChatChannel, msg)
	}

	createSettings := resolveMapCreateSettings(cfg.Local.Settings.MapGenerator, cfg.Local.Settings.MapPreset)
	mapTypeNum := 0
	if !createSettings.usesGenerator() && createSettings.preset != "" {
		mapTypeNum = getMapTypeNum(createSettings.preset)
		if mapTypeNum < 0 {
			mapTypeNum = 0
		}
	}

	/* Generate code to make filename */
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, uint64(ourseed))
	ourcode := fmt.Sprintf("%02d%v", mapTypeNum, base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	sName := "gen-" + ourcode + ".zip"

	filename := cfg.GetSavesFolder() +
		"/" + sName
	factargs := buildNewMapArgs(filename, haveSeed, ourseed, createSettings)

	if cfg.Local.Settings.Scenario != "" || strings.EqualFold(cfg.Local.Settings.Scenario, "none") {
		cfg.Local.Settings.NewMap = true
	}

	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", GetFactorioBinary(), strings.Join(factargs, " "))
	cwlog.DoLogCW(lbuf)

	cmd := exec.Command(GetFactorioBinary(), factargs...)
	_, aerr := cmd.CombinedOutput()

	if aerr != nil {
		err := fmt.Errorf("an error occurred attempting to generate the map: %w", aerr)
		cwlog.DoLogCW(err.Error())
		return "", err
	}

	return sName, nil
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
		LogCMS(cfg.Local.Channel.ChatChannel, buf)
		return
	}
	defer func() {
		if err := from.Close(); err != nil {
			cwlog.DoLogCW("mapReset: failed to close source map: %v", err)
		}
	}()

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
	defer func() {
		if err := to.Close(); err != nil {
			cwlog.DoLogCW("mapReset: failed to close archive file: %v", err)
		}
	}()

	_, errc := io.Copy(to, from)
	if errc != nil {
		buf := fmt.Sprintf("An error occurred when attempting to write the map archive file: %s", errc)
		LogCMS(cfg.Local.Channel.ChatChannel, buf)
		return
	}

	buf := fmt.Sprintf("Map archived as: %s", newmapurl)
	LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
}
