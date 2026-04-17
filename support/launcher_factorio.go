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
func LaunchFactorio(generation uint64) error {
	fact.SetLastBan("")

	waitForDiscord()

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
		return fmt.Errorf("factorio is not installed at %s", checkFactPath)
	}

	/* Find, test and load newest save game available */
	found, fileName, folderName := GetSaveGame(true)
	if !found {
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Unable to access save-games.", glob.COLOR_RED))
		fact.SetAutolaunch(false, true)
		return fmt.Errorf("unable to access save-games")
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if !*glob.LocalTestMode && throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			buf := fmt.Sprintf("Rate limiting: Waiting for %v seconds before launching Factorio.", delay)
			glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "Warning", buf, glob.COLOR_ORANGE))

			for i := 0; i < delay*11 && throt > 0 && glob.ServerRunning() && glob.RelaunchThrottle > 0; i++ {
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

		return fmt.Errorf("unable to write factorio config")
	}

	/* Write or delete whitelist */
	if !cfg.Local.Options.CustomWhitelist {
		count := fact.WriteWhitelist()
		if count > 0 && (cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly) {
			cwlog.DoLogCW("Whitelist of %v players written.", count)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, fact.GetFactorioBinary(), tempargs...)
	cmd.WaitDelay = time.Minute

	/* Hide RCON password and port */
	for i, targ := range tempargs {
		if targ == rconpass {
			tempargs[i] = "***private***"
		} else if targ == rconportStr {
			/* funny, and impossible port number  */
			tempargs[i] = "69420"
		}
	}

	//Reset relaunch throttle
	if !fact.FactorioBootedAt.IsZero() && time.Since(fact.FactorioBootedAt) > time.Hour {
		glob.RelaunchThrottle = 0
	}

	fact.Gametime = (constants.Unknown)
	glob.ResetNoResponseCount()
	cwlog.DoLogCW("Factorio booting...")

	/* Launch Factorio */
	cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))
	linuxSetProcessGroup(cmd)
	/* Connect Factorio stdout to a blocking reader */
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdoutPipe() Details: %s", err))
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))
		cancel()
		return err
	}
	/* Stdin */
	tpipe, errp := cmd.StdinPipe()

	/* Factorio is not happy. */
	if errp != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))
		cancel()
		return errp
	}

	/* Handle launch errors */
	err = cmd.Start()
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
		glob.SetBootMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetBootMessage(), "ERROR", "Launching Factorio failed!", glob.COLOR_RED))
		cancel()
		return err
	}

	glob.FactorioLock.Lock()
	glob.FactorioContext = ctx
	glob.FactorioCancel = cancel
	glob.FactorioCmd = cmd
	glob.FactorioLock.Unlock()

	/* Save pipe */
	if tpipe != nil {
		fact.PipeLock.Lock()
		fact.Pipe = tpipe
		fact.PipeLock.Unlock()
	}

	lines := make(chan string, constants.FactorioStdoutChannelCapacity)
	fact.SetGameLineCh(lines)
	go func(r io.ReadCloser, lines chan<- string) {
		defer r.Close()
		defer close(lines)
		scanner := bufio.NewScanner(r)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		dropped := 0
		lastDropLog := time.Time{}
		for scanner.Scan() {
			line := scanner.Text()
			select {
			case lines <- line:
			default:
				// Prevent Factorio from blocking on stdout if the consumer is slow.
				dropped++
				if isCriticalFactorioLine(line) {
					input := preProcessFactorioOutput(line)
					runHandles(noChatHandles, input)
				}
				if lastDropLog.IsZero() || time.Since(lastDropLog) > time.Second*10 {
					lastDropLog = time.Now()
					cwlog.DoLogCW("Factorio stdout backlog: dropped %v lines (consumer slow).", dropped)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			cwlog.DoLogCW("Factorio stdout scan error: %v", err)
			fact.NotifyFactorioHealth("stdout-scan-error", err)
			return
		}
		cwlog.DoLogCW("Factorio stdout stream closed.")
		fact.NotifyFactorioHealth("stdout-closed", nil)
	}(stdout, lines)
	go func() {
		err := cmd.Wait()
		fact.NotifyFactorioProcessExit(generation, err)
	}()

	return nil
}

func isCriticalFactorioLine(line string) bool {
	// These are used to transition ChatWire state; dropping them can leave the server stuck in "booting".
	return strings.Contains(line, "Info RemoteCommandProcessor") && strings.Contains(line, "Starting RCON interface") ||
		strings.Contains(line, "Goodbye")
}
