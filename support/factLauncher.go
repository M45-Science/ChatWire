package support

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

func launchFactortio() {

	/* Insert soft mod */
	if cfg.Global.Paths.Binaries.SoftModInserter != "" {
		command := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Binaries.SoftModInserter
		out, errs := exec.Command(command, cfg.Local.Callsign).Output()
		if errs != nil {
			cwlog.DoLogCW(fmt.Sprintf("Unable to run soft-mod insert script. Details:\nout: %v\nerr: %v", string(out), errs))
		} else {
			//cwlog.DoLogCW("Soft-mod inserted into save file.")
		}
	}

	/* Generate config file for Factorio server, if it fails stop everything.*/
	if !fact.GenerateFactorioConfig() {
		fact.SetAutoStart(false)
		fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to generate config file for Factorio server.")
		return
	}

	/* Relaunch Throttling */
	throt := fact.GetRelaunchThrottle()
	if throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			cwlog.DoLogCW(fmt.Sprintf("Automatically rebooting Factroio in %d seconds.", delay))
			for i := 0; i < delay*11 && throt > 0; i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	/* Timer gets longer each reboot */
	fact.SetRelaunchThrottle(throt + 1)

	/* Lock so we don't interrupt updates or launch twice */
	fact.FactorioLaunchLock.Lock()

	var err error
	var tempargs []string

	/* Factorio launch parameters */
	rconport := cfg.Local.Port + cfg.Global.Options.RconOffset
	rconportStr := fmt.Sprintf("%v", rconport)
	rconpass := cfg.Global.Factorio.RCONPass
	port := cfg.Local.Port
	postStr := fmt.Sprintf("%v", port)
	serversettings := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.FactorioPrefix +
		cfg.Local.Callsign + "/" +
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

	/*Auth Server Bans ( global bans ) */
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
	fact.SetModLoadString(constants.Unknown)

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
		fact.FactorioLaunchLock.Unlock()
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
		/* close lock */
		fact.FactorioLaunchLock.Unlock()
		fact.DoExit(true)
		return
	}

	/* Okay, Factorio is running now, prep */
	fact.SetFactRunning(true, false)
	fact.SetFactorioBooted(false)

	fact.SetGameTime(constants.Unknown)
	fact.SetNoResponseCount(0)
	cwlog.DoLogCW("Factorio booting...")

	/* Unlock launch lock */
	fact.FactorioLaunchLock.Unlock()
}
