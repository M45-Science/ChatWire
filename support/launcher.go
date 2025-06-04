package support

import (
	"archive/zip"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"sort"
	"strings"
	"syscall"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

var (
	BotIsReady bool
)

const (
	dotInterval      = time.Second * 5
	progressInterval = time.Second * 10
	syncModsTimeout  = time.Minute * 15
)

func SyncMods(i *discordgo.InteractionCreate, optionalFileName string) bool {

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	fileName := ""
	if optionalFileName == "" {
		_, fileName, _ = GetSaveGame(true)
	} else {
		fileName = optionalFileName
	}

	if !fact.GenerateFactorioConfig() {
		cwlog.DoLogCW("Unable to write factorio config file.")
		return false
	}

	var tempargs []string = []string{"--sync-mods", fileName}

	// Create a context with the timeout
	ctx, cancel := context.WithTimeout(context.Background(), syncModsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, fact.GetFactorioBinary(), tempargs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		cwlog.DoLogCW("Error starting command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)

	modsLoading := false
	var lastDot time.Time

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		//cwlog.DoLogCW("Received line: %s\n", line)
		parts := strings.Split(line, " ")
		numParts := len(parts)
		if numParts > 1 {
			if parts[1] == "Goodbye" {
				break
			}
		}
		if numParts > 2 {
			if parts[1] == "Loading" && parts[2] == "mod" {
				if !modsLoading {
					modsLoading = true
					msg := "Loading mods"
					glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Sync",
						msg, glob.COLOR_CYAN)
					cwlog.DoLogCW(msg)
				} else if time.Since(lastDot) > dotInterval {
					lastDot = time.Now()
					if glob.UpdateMessage != nil && len(glob.UpdateMessage.Embeds) > 0 {
						embed := glob.UpdateMessage.Embeds[0]
						embed.Description = embed.Description + " ."
						disc.DS.ChannelMessageEditEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage.ID, embed)
					}
				}
			}
		}
		if numParts > 3 {
			if parts[3] == "Downloading" {
				urlParts := strings.Split(parts[4], "/")
				if len(urlParts) > 1 {
					if urlParts[1] == "download" {
						msg := "Downloading: " + urlParts[2]
						glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Sync",
							msg, glob.COLOR_CYAN)
						cwlog.DoLogCW(msg)
					}
				}
			}
		}

		if line == "Mods synced" {
			msg := "All mods synced."
			glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Mod Sync",
				msg, glob.COLOR_CYAN)
			cwlog.DoLogCW(msg)
		}
	}

	if err := scanner.Err(); err != nil {
		cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
		return false
	}

	if err := cmd.Wait(); err != nil {
		cwlog.DoLogCW("SyncMods: Command finished with error: %v\n", err)
		return false
	}

	return true
}

/* Find the newest save game */
func GetSaveGame(doInject bool) (foundGood bool, fileName string, fileDir string) {
	path := cfg.GetSavesFolder()

	files, err := os.ReadDir(path)

	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW("Unable to read saves folder, stopping.")
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
		path := cfg.GetSavesFolder()

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

// Wait for a moment, so we don't lose factorio booting message on first connect.

func waitForDiscord() {
	if BotIsReady {
		return
	}
	for x := 0; x <= 10; x++ {
		if BotIsReady {
			return
		}
		cwlog.DoLogCW("Waiting for Discord...")
		time.Sleep(time.Second)
	}
}

func isProcessRunning(cmd *exec.Cmd) bool {
	// Check if the process state is nil (still running)
	if cmd.ProcessState == nil {
		return true
	}

	// Check if the process has exited
	return !cmd.ProcessState.Exited()
}

func isProcessAlive(pid int) bool {
	// Send signal 0 to the process
	err := syscall.Kill(pid, 0)
	return err == nil
}

/* Create config files, launch factorio */
func launchFactorio() {

	if fact.FactIsRunning {
		cwlog.DoLogCW("launchFactorio: Factorio is already running.")
		return
	}

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	fact.SetLastBan("")

	waitForDiscord()
	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Notice", "Launching Factorio...", glob.COLOR_GREEN)
	fact.QueueFactReboot = false

	/* Allow crash reports right away */
	glob.LastCrashReport = time.Time{}

	/* Clear this so we know if the loaded map has our soft mod or not */
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
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "ERROR", "Factorio is not installed. Use `/factorio install-factorio` to install it.", glob.COLOR_RED)

		cwlog.DoLogCW("Factorio does not appear to be installed at the configured path: " + checkFactPath)
		fact.SetAutolaunch(false, true)
		return
	}

	/* Find, test and load newest save game available */
	found, fileName, folderName := GetSaveGame(true)
	if !found {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "ERROR", "Unable to access save-games.", glob.COLOR_RED)
		fact.SetAutolaunch(false, true)
		return
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if !*glob.LocalTestMode && throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			buf := fmt.Sprintf("Rate limiting: Waiting for %v seconds before launching Factorio.", delay)
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Warning", buf, glob.COLOR_ORANGE)

			for i := 0; i < delay*11 && throt > 0 && glob.ServerRunning && glob.RelaunchThrottle > 0; i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	/* Timer gets longer each reboot */
	glob.RelaunchThrottle = (glob.RelaunchThrottle + 1)

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

	if cfg.Local.Settings.NewMap && cfg.Local.Settings.Scenario != "none" && cfg.Local.Settings.Scenario != "" {
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

	modFiles := GetModFiles()
	if len(modFiles) > 0 {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", "Loading mods...", glob.COLOR_GREEN)
	}

	if glob.FactorioCancel != nil {
		cwlog.DoLogCW("Canceling existing context.")
		glob.FactorioCancel()
	}
	if glob.FactorioCmd != nil {
		if isProcessRunning(glob.FactorioCmd) || isProcessAlive(glob.FactorioCmd.Process.Pid) {
			cwlog.DoLogCW("Killing Previous Factorio process.")

			glob.FactorioCmd.Process.Kill()
			glob.FactorioCmd.Wait()
		}
	}
	glob.FactorioContext, glob.FactorioCancel = context.WithCancel(context.Background())

	/* Inject softmod */
	if cfg.Local.Options.SoftModOptions.InjectSoftMod {
		if cfg.Local.Settings.Scenario == "" || cfg.Local.Settings.Scenario == "none" {
			injectSoftMod(fileName, folderName)
		} else {
			cwlog.DoLogCW("Softmod disabled for scenario.")
		}
	}

	/* Generate config file for Factorio server, if it fails stop everything.*/
	if !fact.GenerateFactorioConfig() {
		fact.SetAutolaunch(false, true)

		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "ERROR", "Unable to write a config file for Fatorio.", glob.COLOR_RED)

		return
	}

	/* Write or delete whitelist */
	if !cfg.Local.Options.CustomWhitelist {
		count := fact.WriteWhitelist()
		if count > 0 && (cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly) {
			cwlog.DoLogCW("Whitelist of %v players written.", count)
		}
	}

	/* Run Factorio */
	glob.FactorioCmd = exec.Command(fact.GetFactorioBinary(), tempargs...)
	glob.FactorioCmd.WaitDelay = time.Minute

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
	fact.SetFactRunning(true, false)
	fact.FactorioBooted = false

	//Reset relaunch throttle
	if time.Since(fact.FactorioBootedAt) > time.Hour {
		glob.RelaunchThrottle = 0
	}
	fact.FactorioBootedAt = time.Time{}

	fact.Gametime = (constants.Unknown)
	glob.NoResponseCount = 0
	cwlog.DoLogCW("Factorio booting...")

	/* Launch Factorio */
	cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))
	linuxSetProcessGroup(glob.FactorioCmd)
	/* Connect Factorio stdout to a buffer for processing */
	fact.GameBuffer = new(bytes.Buffer)
	logwriter := io.MultiWriter(fact.GameBuffer)
	glob.FactorioCmd.Stdout = logwriter
	/* Stdin */
	tpipe, errp := glob.FactorioCmd.StdinPipe()

	/* Factorio is not happy. */
	if errp != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "ERROR", "Launching Factorio failed!", glob.COLOR_RED)

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
	err = glob.FactorioCmd.Start()
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "ERROR", "Launching Factorio failed!", glob.COLOR_RED)
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

	UpdateDuration()
	UpdateInterval()

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

	fact.WriteFact("/gspeed %0.2f", cfg.Local.Options.Speed)

}

func GetModFiles() []string {
	modPath := cfg.GetModsFolder()

	modList, errm := os.ReadDir(modPath)
	modStrings := []string{}

	if errm == nil {
		for _, mod := range modList {
			if strings.HasSuffix(mod.Name(), ".zip") {
				modStrings = append(modStrings, strings.TrimSuffix(mod.Name(), ".zip"))
			}
		}
	}

	return modStrings
}
