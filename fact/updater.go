package fact

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/constants"
)

/* Check if Factorio update zip is valid */
func CheckZip(filename string) bool {

	ctx, cancel := context.WithTimeout(context.Background(), constants.ZipIntegrityLimit)
	defer cancel()

	cmdargs := []string{"-t", filename}
	//: " + strings.Join(cmdargs, " "))
	cmd := exec.CommandContext(ctx, cfg.Global.PathData.ZipBinaryPath, cmdargs...)
	o, err := cmd.CombinedOutput()
	out := string(o)

	if ctx.Err() == context.DeadlineExceeded {
		botlog.DoLog("Zip integrity check timed out.")
	}

	if err == nil {
		if strings.Contains(out, "No errors detected in compressed data of ") {
			botlog.DoLog("Zipfile integrity good!")
			return true
		}
	}

	botlog.DoLog("Zipfile integrity check failed!")
	return false
}

/* Check if there is a new Factorio update */
func CheckFactUpdate(logNoUpdate bool) {

	if cfg.Global.PathData.FactUpdaterPath != "" {

		UpdateFactorioLock.Lock()
		defer UpdateFactorioLock.Unlock()

		//Give up on check/download after a while
		ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
		defer cancel()

		//Create cache directory
		err := os.MkdirAll(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactUpdateCache, 0777)
		if err != nil {
			botlog.DoLog(err.Error())
		}

		cmdargs := []string{cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdaterPath, "-O", cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache, "-a", GetFactorioBinary(), "-d"}
		if cfg.Local.UpdateFactExp {
			cmdargs = append(cmdargs, "-x")
		}
		//botlog.DoLog("Update args: " + strings.Join(cmdargs, " "))

		cmd := exec.CommandContext(ctx, cfg.Global.PathData.FactUpdaterShell, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			botlog.DoLog("fact update check: download/check timed out... purging cache.")
			os.RemoveAll(cfg.Global.PathData.FactUpdateCache)
			return
		}

		if err == nil {
			clines := strings.Split(out, "\n")
			for _, line := range clines {
				linelen := len(line)
				var newversion string
				var oldversion string

				if linelen > 0 {

					words := strings.Split(line, " ")
					numwords := len(words)

					if strings.HasPrefix(line, "No updates available") {
						if logNoUpdate {
							mess := "fact update check: Factorio is up-to-date."
							botlog.DoLog(mess)
						}
						return
					} else if strings.HasPrefix(line, "Wrote ") {
						if linelen > 1 && strings.Contains(line, ".zip") {

							//Only trigger on a new patch file
							if line != NewPatchName {
								NewPatchName = line

								if numwords > 1 &&
									CheckZip(words[1]) {
									mess := "Factorio update downloaded and verified, will update when no players are online."
									CMS(cfg.Local.ChannelData.ChatID, mess)
									WriteFact("/cchat [SYSTEM] " + mess)
									botlog.DoLog(mess)
								} else {
									os.RemoveAll(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache)
									//Purge patch name so we attempt check again
									NewPatchName = constants.Unknown
									botlog.DoLog("fact update check: Factorio update zip invalid... purging cache.")
								}
							}
							return
						}
					} else if strings.HasPrefix(line, "Dry run: would have fetched update from") {
						if numwords >= 9 {
							oldversion = words[7]
							newversion = words[9]

							messdisc := fmt.Sprintf("**Factorio update available:** '%v' to '%v'", oldversion, newversion)
							messfact := fmt.Sprintf("Factorio update available: '%v' to '%v'", oldversion, newversion)
							SetDoUpdateFactorio(true)

							//Don't message, unless this is actually a unique new version
							if NewVersion != newversion {
								NewVersion = newversion

								CMS(cfg.Local.ChannelData.ChatID, messdisc)

								WriteFact("/cchat [SYSTEM] " + messfact)
								botlog.DoLog(messfact)
							}
						}
					}
				}
			}
		}

		botlog.DoLog(fmt.Sprintf("fact update dry: update_fact.py:\n%v", out))
	}

}

/* Update Factorio */
func FactUpdate() {

	UpdateFactorioLock.Lock()
	defer UpdateFactorioLock.Unlock()

	//Give up on patching eventually
	ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
	defer cancel()

	err := os.MkdirAll(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactUpdateCache, 0777)
	if err != nil {
		botlog.DoLog(err.Error())
	}

	if !IsFactRunning() {
		//Keep us from stepping on a factorio launch or update
		FactorioLaunchLock.Lock()
		defer FactorioLaunchLock.Unlock()

		cmdargs := []string{cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdaterPath, "-O", cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache, "-a", GetFactorioBinary()}
		if cfg.Local.UpdateFactExp {
			cmdargs = append(cmdargs, "-x")
		}
		//botlog.DoLog("Update args: " + strings.Join(cmdargs, " "))

		cmd := exec.CommandContext(ctx, cfg.Global.PathData.FactUpdaterShell, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			botlog.DoLog("fact update: (error) Factorio update patching timed out, deleting possible corrupted patch file.")

			os.RemoveAll(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache)
			return
		}

		if err == nil {
			clines := strings.Split(out, "\n")
			for _, line := range clines {
				linelen := len(line)

				if linelen > 0 {

					//words := strings.Split(line, " ")
					if strings.HasPrefix(line, "Update applied successfully!") {
						mess := "fact update: Factorio updated successfully!"
						botlog.DoLog(mess)
						return
					}
				}
			}
		} else {
			os.RemoveAll(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache)
			botlog.DoLog("fact update: (error) Non-zero exit code... purging update cache.")
			botlog.DoLog(fmt.Sprintf("fact update: (error) update_fact.py:\n%v", out))
			return
		}

		botlog.DoLog("fact update: (unknown error): " + out)
		return
	} else {

		botlog.DoLog("fact update: (error) Factorio is currently running, unable to update.")
		return
	}
}
