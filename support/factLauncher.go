package support

import (
	"bytes"
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

func launchFactorio() {

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

	/* Insert soft mod */
	if cfg.Global.Paths.Binaries.SoftModInserter != "" {
		command := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Binaries.SoftModInserter
		out, errs := exec.Command(command, cfg.Local.Callsign).Output()
		if errs != nil {
			cwlog.DoLogCW(fmt.Sprintf("Unable to run soft-mod insert script. Details:\nout: %v\nerr: %v", string(out), errs))
		}
	}

	/* Generate config file for Factorio server, if it fails stop everything.*/
	if !fact.GenerateFactorioConfig() {
		fact.FactAutoStart = false
		fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to generate config file for Factorio server.")
		return
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			cwlog.DoLogCW(fmt.Sprintf("Automatically rebooting Factorio in %d seconds.", delay))
			for i := 0; i < delay*11 && throt > 0 && glob.ServerRunning; i++ {
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

	tempargs = append(tempargs, "--start-server-load-latest")
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
	if cfg.Local.Options.Whitelist {
		tempargs = append(tempargs, "--use-server-whitelist")
		tempargs = append(tempargs, "true")
	}

	/* Write or delete whitelist */
	count := fact.WriteWhitelist()
	if count > 0 && cfg.Local.Options.Whitelist {
		cwlog.DoLogCW(fmt.Sprintf("Whitelist of %v players written.", count))
	}

	//Clear mod load string
	fact.ModList = []string{}

	/* Run Factorio */
	var cmd *exec.Cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)

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
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		/* close lock  */
		fact.DoExit(true)
		return
	}

	/* Save pipe */
	if tpipe != nil && err == nil {
		fact.PipeLock.Lock()
		fact.Pipe = tpipe
		fact.PipeLock.Unlock()
	}

	/* Handle launch errors */
	err = cmd.Start()
	if err != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
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

	/* Config reset-interval */
	if fact.NextReset != "" {
		fact.WriteFact("/resetint " + fact.NextReset)
	}
	if fact.TillReset != "" {
		fact.WriteFact("/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")")
	}
	if cfg.Local.Options.SoftModOptions.CleanMap {
		//fact.LogCMS(cfg.Local.Channel.ChatChannel, "Cleaning map.")
		fact.WriteFact("/cleanmap")
	}
	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		fact.WriteFact("/blueprints off")
		//fact.LogCMS(cfg.Local.Channel.ChatChannel, "Blueprints disabled.")
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		fact.WriteFact("/enablecheats on")
		//fact.LogCMS(cfg.Local.Channel.ChatChannel, "Cheats enabled.")
	}

	/* Patreon list */
	if len(disc.RoleList.Patreons) > 0 {
		fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ","))
	}
	if len(disc.RoleList.NitroBooster) > 0 {
		fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ","))
	}
}
