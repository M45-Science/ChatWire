package support

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

var dlcNames = []string{"elevated-rails", "quality", "space-age"}

type modListData struct {
	Mods []modData
}
type modData struct {
	Name    string
	Enabled bool
}

func GetGameMods() (*modListData, error) {
	return configGameMods(nil, false)
}

func SyncMods() {
	_, fileName, _ := GetSaveGame(true)
	var tempargs []string = []string{"--sync-mods", fileName}

	var cmd *exec.Cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)
	data, err := cmd.Output()
	if err != nil {
		cwlog.DoLogCW(err.Error())
	} else {
		if strings.Contains(string(data), "Mods synced") {
			cwlog.DoLogCW("Mods synced.")
		}
	}
}

func configGameMods(controlList []string, setState bool) (*modListData, error) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/" + "mod-list.json"

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

					cwlog.DoLogCW(BoolToString(setState) + " " + serverMod.Name)
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
		cwlog.DoLogCW("Wrote mod-list.json")
	}
	return &serverMods, err
}

/* Find the newest save game */
func GetSaveGame(doInject bool) (foundGood bool, fileName string, fileDir string) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	files, err := os.ReadDir(path)

	/* We can't read saves dir */
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to read saves folder, stopping.")
		return false, "", ""
	}

	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files and directories
		if !f.IsDir() {
			if strings.HasSuffix(f.Name(), ".zip") && !strings.HasSuffix(f.Name(), "tmp.zip") {
				tempf = append(tempf, f)
			}
		}
	}

	//Newest first
	sort.Slice(tempf, func(i, j int) bool {
		iInfo, _ := tempf[i].Info()
		jInfo, _ := tempf[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
		//return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	numSaves := len(tempf)
	if numSaves <= 0 {
		fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "No saves found, stopping.")
		return false, "", ""
	}

	for pos := 0; pos < numSaves; pos++ {
		name := tempf[pos].Name()

		if name == "" {
			continue
		}

		showError := false
		if pos == 0 {
			showError = true
		}
		good, folder := fact.CheckSave(path, name, showError)
		if good {
			return true, path + "/" + name, folder
		}

	}

	return false, "", ""
}

type zipFilesData struct {
	Name string
	Data []byte
}

/* Used for reading softmod directory */
func readFolder(path string, sdir string) []zipFilesData {

	var zipFiles []zipFilesData

	/* Get all softmod files */
	sFiles, err := os.ReadDir(path)
	if err != nil {
		cwlog.DoLogCW("Unable to read softmod folder.")
		return nil
	}

	for _, file := range sFiles {
		if !file.IsDir() {
			dat, err := os.ReadFile(path + "/" + file.Name())
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to read softmod files.")
				continue
			}

			zipFiles = append(zipFiles, zipFilesData{Name: sdir + "/" + file.Name(), Data: dat})
		} else {
			tfiles := readFolder(path+"/"+file.Name(), sdir+"/"+file.Name())
			zipFiles = append(zipFiles, tfiles...)
		}
	}

	return zipFiles
}

/* Insert our softmod files into the save zip */
func injectSoftMod(fileName, folderName string) {
	var zipFiles []zipFilesData

	/* Read needed files from existing save */
	archive, errz := zip.OpenReader(fileName)
	if errz != nil {
		cwlog.DoLogCW("sm-inject: unable to open save game.")
		return
	} else {
		defer archive.Close()
		for _, f := range archive.File {
			fileName := path.Base(f.Name)
			/* Make sure these files are in the correct directory in the zip */
			if strings.Compare(path.Dir(f.Name), folderName) == 0 &&
				/* Only copy relevant files */
				strings.HasPrefix(fileName, "level.dat") ||
				strings.HasSuffix(fileName, ".json") ||
				strings.HasSuffix(fileName, ".dat") ||
				strings.EqualFold(fileName, "level-init.dat") ||
				strings.EqualFold(fileName, "level.datmetadata") {
				file, err := f.Open()
				if err != nil {
					cwlog.DoLogCW("sm-inject: unable to open " + f.Name)
				} else {

					defer file.Close()
					data, rerr := io.ReadAll(file)

					dlen := uint64(len(data))
					if rerr != nil && rerr != io.EOF {
						cwlog.DoLogCW("Unable to read file: " + f.Name)
						continue
					} else if dlen != f.UncompressedSize64 {
						sbuf := fmt.Sprintf("%v vs %v", dlen, f.UncompressedSize64)
						cwlog.DoLogCW("Sizes did not match: " + f.Name + ", " + sbuf)
					} else {
						tmp := zipFilesData{Name: f.Name, Data: data}
						zipFiles = append(zipFiles, tmp)
					}
				}
			}
		}

		/* Read files in from softmod */
		blackList := []string{"img-source", "out"}                   /* Wildcard exclude */
		allowList := []string{"README.md", "preview.jpg", "LICENSE"} /* Always include */
		allowExt := []string{".lua", ".png"}

		tfiles := readFolder(cfg.Local.Options.SoftModOptions.SoftModPath, folderName)
		var addFiles []zipFilesData
		for _, tf := range tfiles {
			skip := false
			for _, al := range allowList {
				if strings.HasSuffix(tf.Name, al) {
					addFiles = append(addFiles, tf)
				}
			}
			for _, bl := range blackList {
				if strings.Contains(tf.Name, bl) {
					skip = true
				}
			}
			if skip {
				continue
			}
			for _, ext := range allowExt {
				if strings.HasSuffix(tf.Name, ext) {
					addFiles = append(addFiles, tf)
				}
			}
		}
		zipFiles = append(zipFiles, addFiles...)

		numFiles := len(zipFiles)
		if numFiles <= 0 {
			cwlog.DoLogCW("No softmod files found, stopping.")
			return
		}

		/* Add old save files into zip */
		path := cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			cfg.Global.Paths.Folders.Saves

		newZipFile, err := os.Create(path + constants.TempSaveName)
		if err != nil {
			cwlog.DoLogCW("injectSoftMod: Unable to create temp save.")
			return
		}
		defer newZipFile.Close()

		zipWriter := zip.NewWriter(newZipFile)
		defer zipWriter.Close()

		for _, file := range zipFiles {
			fh := new(zip.FileHeader)
			fh.Name = file.Name
			fh.UncompressedSize64 = uint64(len(file.Data))

			writer, err := zipWriter.CreateHeader(fh)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to create blank file in zip.")
				continue
			}

			_, err = writer.Write(file.Data)
			if err != nil {
				cwlog.DoLogCW("injectSoftMod: Unable to copy file data into zip.")
				continue
			}
		}

		err = os.Rename(path+constants.TempSaveName, fileName)
		if err != nil {
			cwlog.DoLogCW("Couldn't rename softmod temp save.")
			return
		}
		cwlog.DoLogCW("SoftMod injected.")

	}
}

/* Create config files, launch factorio */
func launchFactorio() {

	if fact.FactIsRunning {
		cwlog.DoLogCW("launchFactorio: Factorio is already running.")
		return
	}

	/* Allow crash reports right away */
	glob.LastCrashReport = time.Time{}

	/* Clear this so we know if the the loaded map has our soft mod or not */
	glob.SoftModVersion = constants.Unknown
	glob.OnlineCommand = constants.OnlineCommand
	fact.OnlinePlayersLock.Lock()
	glob.OnlinePlayers = []glob.OnlinePlayerData{}
	fact.OnlinePlayersLock.Unlock()

	/* Check for factorio install */
	checkFactPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir

	if _, err := os.Stat(checkFactPath); os.IsNotExist(err) {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio does not appear to be installed. Use /factorio install-factorio to install it.")
		cwlog.DoLogCW("Factorio does not appear to be installed at the configured path: " + checkFactPath)
		fact.FactAutoStart = false
		return
	}

	/* Find, test and load newest save game available */
	found, fileName, folderName := GetSaveGame(true)
	if !found {
		cwlog.DoLogCW("Unable to load any saves.")
		fact.FactAutoStart = false
		return
	}

	/* Inject softmod */
	if cfg.Local.Options.SoftModOptions.InjectSoftMod {
		if cfg.Local.Settings.Scenario == "" {
			injectSoftMod(fileName, folderName)
		} else {
			cwlog.DoLogCW("Softmod disabled for scenario.")
		}
	}

	/* Generate config file for Factorio server, if it fails stop everything.*/
	if !fact.GenerateFactorioConfig() {
		fact.FactAutoStart = false
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to generate config file for Factorio server.")
		return
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if !*glob.LocalTestMode && throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("Rate limiting: Automatically rebooting Factorio in %d seconds.", delay))
			for i := 0; i < delay*11 && throt > 0 && glob.ServerRunning && glob.RelaunchThrottle > 0; i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	/* Timer gets longer each reboot */
	glob.RelaunchThrottle = (throt + 1)

	var err error
	var tempargs []string

	/* Factorio launch parameters */
	rconport := cfg.Local.Port + cfg.Global.Options.RconOffset
	rconportStr := fmt.Sprintf("%v", rconport)
	rconpass := glob.RandomBase64String(256)
	glob.RCONPass = rconpass
	cfg.Local.RCONPass = rconpass
	cfg.WriteLCfg()

	port := cfg.Local.Port
	postStr := fmt.Sprintf("%v", port)
	serversettings := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ServSettingsName

	if cfg.Local.Settings.NewMap {
		cfg.Local.Settings.NewMap = false
		tempargs = append(tempargs, "--start-server-load-scenario")
		tempargs = append(tempargs, cfg.Local.Settings.Scenario)
	} else {
		tempargs = append(tempargs, "--start-server")
		tempargs = append(tempargs, fileName)
	}

	tempargs = append(tempargs, "--rcon-port")
	tempargs = append(tempargs, rconportStr)

	tempargs = append(tempargs, "--rcon-password")
	tempargs = append(tempargs, rconpass)

	tempargs = append(tempargs, "--port")
	tempargs = append(tempargs, postStr)

	tempargs = append(tempargs, "--server-settings")
	tempargs = append(tempargs, serversettings)

	/* Auth Server Bans ( global bans ) */
	if cfg.Global.Options.UseAuthserver {
		tempargs = append(tempargs, "--use-authserver-bans")
	}

	/* Whitelist */
	if cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly || cfg.Local.Options.CustomWhitelist {
		tempargs = append(tempargs, "--use-server-whitelist")
		tempargs = append(tempargs, "true")
	}

	/* Write or delete whitelist */
	if !cfg.Local.Options.CustomWhitelist {
		count := fact.WriteWhitelist()
		if count > 0 && (cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly) {
			cwlog.DoLogCW("Whitelist of %v players written.", count)
		}
	}
	fact.WriteAdminlist()

	modFiles := GetModFiles()
	if len(modFiles) > 0 {
		fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "Loading mods...")
	}

	if glob.FactorioCancel != nil {
		cwlog.DoLogCW("Killing previous factorio context!!!")
		glob.FactorioCancel()
	}
	glob.FactorioContext, glob.FactorioCancel = context.WithCancel(context.Background())

	/* Run Factorio */
	cmd := exec.Command(fact.GetFactorioBinary(), tempargs...)

	/* Hide RCON password and port */
	for i, targ := range tempargs {
		if targ == rconpass {
			tempargs[i] = "***private***"
		} else if targ == rconportStr {
			/* funny, and impossible port number  */
			tempargs[i] = "69420"
		}
	}

	/* Okay, prep for factorio launch */
	fact.SetFactRunning(true)
	fact.FactorioBooted = false
	fact.FactorioBootedAt = time.Time{}

	fact.Gametime = (constants.Unknown)
	glob.NoResponseCount = 0
	cwlog.DoLogCW("Factorio booting...")

	/* Launch Factorio */
	cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))
	LinuxSetProcessGroup(cmd)
	/* Connect Factorio stdout to a buffer for processing */
	fact.GameBuffer = new(bytes.Buffer)
	logwriter := io.MultiWriter(fact.GameBuffer)
	cmd.Stdout = logwriter
	/* Stdin */
	tpipe, errp := cmd.StdinPipe()

	/* Factorio is not happy. */
	if errp != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		/* close lock  */
		fact.DoExit(true)
		return
	}

	/* Save pipe */
	if tpipe != nil {
		fact.PipeLock.Lock()
		fact.Pipe = tpipe
		fact.PipeLock.Unlock()
	}

	/* Handle launch errors */
	err = cmd.Start()
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
		fact.DoExit(true)
		return
	}
}

func ConfigSoftMod() {
	fact.WriteFact("/cname " + strings.ToUpper(cfg.Local.Callsign+"-"+cfg.Local.Name))

	/* Config new-player restrictions */
	if cfg.Local.Options.SoftModOptions.Restrict {
		fact.WriteFact("/restrict on")
	} else {
		fact.WriteFact("/restrict off")
	}

	/* Config friendly fire */
	if cfg.Local.Options.SoftModOptions.FriendlyFire {
		fact.WriteFact("/friendlyfire on")
	} else {
		fact.WriteFact("/friendlyfire off")
	}

	/* Config friendly fire */
	if cfg.Local.Options.SoftModOptions.OneLife {
		fact.WriteFact("/onelife on")
	} else {
		fact.WriteFact("/onelife off")
	}

	/* Config reset-interval */
	if fact.NextReset != "" {
		fact.WriteFact("/resetint " + fact.NextReset)
	} else {
		fact.WriteFact("/resetint")
	}
	if fact.TillReset != "" && cfg.Local.Options.Schedule != "" {
		fact.WriteFact("/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")")
	} else {
		fact.WriteFact("/resetdur")
	}
	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		fact.WriteFact("/blueprints off")
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		fact.WriteFact("/enablecheats on")
	}

	/* Patreon list */
	if len(disc.RoleList.Patreons) > 0 {
		fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ", ") + ", " +
			strings.Join(disc.RoleList.Supporters, ", "))
	}
	if len(disc.RoleList.NitroBooster) > 0 {
		fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ", "))
	}

	fact.WriteFact(fmt.Sprintf("/gspeed %0.2f", cfg.Local.Options.Speed))
}

func GetModFiles() []string {
	// Check for zip bombs in mod files
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	modList, errm := os.ReadDir(modPath)
	modStrings := []string{}

	if errm == nil {
		for _, mod := range modList {
			if strings.HasSuffix(mod.Name(), ".zip") {
				modStrings = append(modStrings, mod.Name())
			}
		}
	}

	return modStrings
}
