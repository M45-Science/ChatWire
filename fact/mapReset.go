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
	return submitLifecycleRequestAndWait(Request{
		Kind:      ActionMapReset,
		Reason:    "Server rebooting for map reset!",
		RequestID: fmt.Sprintf("map-reset-%d", time.Now().UnixNano()),
	})
}

func mapResetAfterStop(doReport bool) (string, error) {
	/* Only proceed if we were running a map, and we know our Factorio version. */
	if GameMapPath != "" && FactorioVersion != constants.Unknown {
		quickArchive()
	}

	saveName, err := GenNewMap()
	if err != nil {
		msg := fmt.Sprintf("Map reset failed: %v", err)
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		return "", err
	}

	/* If available, use per-server ping setting... otherwise use global */
	pingstr := ""
	if cfg.Local.Options.ResetPingRole != "" {
		pingstr = fmt.Sprintf("<@&%v>", cfg.Local.Options.ResetPingRole)
	} else if cfg.Global.Options.ResetPingRole != "" {
		pingstr = fmt.Sprintf("<@&%v>", cfg.Global.Options.ResetPingRole)
	}
	LogGameCMS(false, cfg.Global.Discord.AnnounceChannel, pingstr+" Map "+cfg.Local.Callsign+"-"+cfg.Local.Name+" auto-reset.")

	glob.VoteBox.LastMapChange = time.Now()
	VoidAllVotes()
	WriteVotes()
	return saveName, nil
}

type mapCreateSettings struct {
	preset               string
	generator            string
	mapGenSettingsPath   string
	mapSettingsPath      string
	usingCachedGenerator bool
	fallbackNotice       string
}

func (s mapCreateSettings) usesGenerator() bool {
	return s.generator != "" && s.mapGenSettingsPath != "" && s.mapSettingsPath != ""
}

func (s mapCreateSettings) withoutGenerator() mapCreateSettings {
	s.generator = ""
	s.mapGenSettingsPath = ""
	s.mapSettingsPath = ""
	s.usingCachedGenerator = false
	s.fallbackNotice = ""
	return s
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
	if pathExists(genSettingsPath) && pathExists(mapSettingsPath) {
		settings.generator = generator
		settings.mapGenSettingsPath = genSettingsPath
		settings.mapSettingsPath = mapSettingsPath
		cacheMapGeneratorBestEffort(generator)
		return settings
	}

	cacheGenSettingsPath, cacheMapSettingsPath := cfg.GetCachedMapGeneratorFiles(generator)
	if pathExists(cacheGenSettingsPath) && pathExists(cacheMapSettingsPath) {
		cwlog.DoLogCW("GenNewMap: map generator %q is unavailable; using local cache. map-gen-settings=%q map-settings=%q", generator, cacheGenSettingsPath, cacheMapSettingsPath)
		settings.generator = generator
		settings.mapGenSettingsPath = cacheGenSettingsPath
		settings.mapSettingsPath = cacheMapSettingsPath
		settings.usingCachedGenerator = true
		settings.fallbackNotice = fmt.Sprintf("**MAP GENERATOR FALLBACK:** Configured generator %q is unavailable. Using the local cached copy.", generator)
		return settings
	}

	cwlog.DoLogCW("GenNewMap: map generator %q is unavailable; generating without map generator. map-gen-settings=%q map-settings=%q cached-map-gen-settings=%q cached-map-settings=%q", generator, genSettingsPath, mapSettingsPath, cacheGenSettingsPath, cacheMapSettingsPath)
	settings.fallbackNotice = fmt.Sprintf("**MAP GENERATOR FALLBACK:** Configured generator %q is unavailable and no local cached copy is available. %s", generator, describeMapGeneratorFallbackTarget(settings.preset))
	return settings
}

func cacheMapGeneratorBestEffort(generator string) {
	if _, _, err := cfg.CacheMapGenerator(generator); err != nil {
		cwlog.DoLogCW("GenNewMap: unable to update local map generator cache for %q: %v", generator, err)
	}
}

func describeMapGeneratorFallbackTarget(preset string) string {
	if preset != "" {
		return fmt.Sprintf("Using map preset %q.", preset)
	}
	return "Using Factorio default map creation."
}

func announceMapGeneratorFallback(notice string) {
	if notice != "" {
		LogCMS(cfg.Local.Channel.ChatChannel, notice)
	}
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

func runMapCreateCommand(factargs []string) error {
	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", GetFactorioBinary(), strings.Join(factargs, " "))
	cwlog.DoLogCW(lbuf)

	cmd := exec.Command(GetFactorioBinary(), factargs...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		mapErr := fmt.Errorf("an error occurred attempting to generate the map: %w", err)
		cwlog.DoLogCW(mapErr.Error())
		return mapErr
	}
	return nil
}

func GenNewMap() (string, error) {
	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	genpath := cfg.GetSavesFolder()
	staging, err := os.MkdirTemp(genpath, ".cw-map-create-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(staging)

	t := time.Now()
	ourseed := int(t.UnixNano() - constants.CWEpoch)
	haveSeed := false

	// Consume a requested seed only after the new save is ready.
	if cfg.Local.Settings.Seed > 0 {
		haveSeed = true
		origSeed := cfg.Local.Settings.Seed
		ourseed = origSeed

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

	filename := filepath.Join(staging, sName)
	factargs := buildNewMapArgs(filename, haveSeed, ourseed, createSettings)
	announceMapGeneratorFallback(createSettings.fallbackNotice)

	if err := runMapCreateCommand(factargs); err != nil {
		if !createSettings.usingCachedGenerator {
			return "", err
		}

		cwlog.DoLogCW("GenNewMap: cached map generator %q failed: %v", createSettings.generator, err)
		fallbackSettings := createSettings.withoutGenerator()
		msg := fmt.Sprintf("**MAP GENERATOR FALLBACK:** Cached generator %q failed while generating the map. Retrying with %s", createSettings.generator, describeMapGeneratorFallbackTarget(fallbackSettings.preset))
		LogCMS(cfg.Local.Channel.ChatChannel, msg)

		if removeErr := os.Remove(filename); removeErr != nil && !os.IsNotExist(removeErr) {
			cwlog.DoLogCW("GenNewMap: failed to remove partial generated map %q before fallback retry: %v", filename, removeErr)
		}

		fallbackArgs := buildNewMapArgs(filename, haveSeed, ourseed, fallbackSettings)
		if fallbackErr := runMapCreateCommand(fallbackArgs); fallbackErr != nil {
			return "", fmt.Errorf("%w; fallback after cached map generator also failed: %v", err, fallbackErr)
		}
	}

	if good, _ := CheckSave(staging, sName, false); !good {
		return "", fmt.Errorf("map generator did not produce a valid save")
	}
	if err := os.Rename(filename, filepath.Join(genpath, sName)); err != nil {
		return "", fmt.Errorf("install generated map: %w", err)
	}
	// Commit reset settings and retire older generated maps only after success.
	cfg.Local.Options.Speed = 1
	cfg.Local.Settings.AutoPause = true
	cfg.Local.Options.SkipReset = false
	if haveSeed {
		cfg.Local.Settings.Seed = 0
	}
	cfg.Local.Settings.NewMap = cfg.Local.Settings.Scenario != "" && !strings.EqualFold(cfg.Local.Settings.Scenario, "none")
	cfg.Local.PendingSave = sName
	SetResetDate()
	if !HasResetInterval() && HasResetTime() && !cfg.Local.Options.NextReset.After(time.Now()) {
		// Generation completes the reset even if launching the new map has to
		// wait or fails. A retry should start this map, not generate another one.
		cfg.Local.Options.NextReset = time.Time{}
	}
	cfg.WriteLCfg()
	flist, err := filepath.Glob(filepath.Join(genpath, "gen-*.zip"))
	if err != nil {
		cwlog.DoLogCW("Unable to list old generated maps: %v", err)
	}
	for _, old := range flist {
		if filepath.Base(old) != sName {
			if err := os.Remove(old); err != nil {
				cwlog.DoLogCW("Failed to delete old generated map %s: %v", old, err)
			}
		}
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
