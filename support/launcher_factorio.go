package support

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

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
	glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Notice", "Launching Factorio...", glob.COLOR_GREEN))
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
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Factorio is not installed. Use `/factorio install-factorio` to install it.", glob.COLOR_RED))

		cwlog.DoLogCW("Factorio does not appear to be installed at the configured path: " + checkFactPath)
		fact.SetAutolaunch(false, true)
		return
	}

	/* Find, test and load newest save game available */
	found, fileName, folderName := GetSaveGame(true)
	if !found {
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Unable to access save-games.", glob.COLOR_RED))
		fact.SetAutolaunch(false, true)
		return
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if !*glob.LocalTestMode && throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			buf := fmt.Sprintf("Rate limiting: Waiting for %v seconds before launching Factorio.", delay)
			glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Warning", buf, glob.COLOR_ORANGE))

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
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Status", "Loading mods...", glob.COLOR_GREEN))
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

		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Unable to write a config file for Fatorio.", glob.COLOR_RED))

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
	if !fact.FactorioBootedAt.IsZero() && time.Since(fact.FactorioBootedAt) > time.Hour {
		glob.RelaunchThrottle = 0
	}
	fact.FactorioBootedAt = time.Now()

	fact.Gametime = (constants.Unknown)
	glob.NoResponseCount = 0
	cwlog.DoLogCW("Factorio booting...")

	/* Launch Factorio */
	cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))
	linuxSetProcessGroup(glob.FactorioCmd)
	/* Connect Factorio stdout to a blocking reader */
	stdout, err := glob.FactorioCmd.StdoutPipe()
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdoutPipe() Details: %s", err))
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))
		fact.DoExit(true)
		return
	}
	/* Stdin */
	tpipe, errp := glob.FactorioCmd.StdinPipe()

	/* Factorio is not happy. */
	if errp != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))

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
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))
		fact.DoExit(true)
		return
	}

	fact.GameLineCh = make(chan string, 256)
	go func(r io.ReadCloser, lines chan<- string) {
		defer r.Close()
		defer close(lines)
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			cwlog.DoLogCW("Factorio stdout scan error: %v", err)
		}
	}(stdout, fact.GameLineCh)
}
