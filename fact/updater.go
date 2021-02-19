package fact

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"../cfg"
	"../constants"
	"../glob"
	"../logs"
)

func CheckZip(filename string) bool {

	ctx, cancel := context.WithTimeout(context.Background(), constants.ZipIntegrityLimit)
	defer cancel()

	cmdargs := []string{"-t", filename}
	logs.Log("Update args: " + strings.Join(cmdargs, " "))
	cmd := exec.CommandContext(ctx, cfg.Global.PathData.ZipBinaryPath, cmdargs...)
	o, err := cmd.CombinedOutput()
	out := string(o)

	if ctx.Err() == context.DeadlineExceeded {
		logs.Log("Zip integrity check timed out.")
	}

	if err == nil {
		if strings.Contains(out, "No errors detected in compressed data of ") {
			logs.Log("Zipfile integrity good!")
			return true
		}
	}

	logs.Log("Zipfile integrity check failed!")
	return false
}

func CheckFactUpdate(logNoUpdate bool) {

	if cfg.Local.AutoUpdate {

		glob.UpdateFactorioLock.Lock()
		defer glob.UpdateFactorioLock.Unlock()

		//Give up on check/download after a while
		ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
		defer cancel()

		//Create cache directory
		os.MkdirAll(cfg.Global.PathData.FactUpdateCache, 0777)

		cmdargs := []string{cfg.Global.PathData.FactUpdaterPath, "-O", cfg.Global.PathData.FactUpdateCache, "-a", "../" + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + cfg.Global.PathData.FactorioBinary, "-d"}
		if cfg.Local.UpdateFactExp {
			cmdargs = append(cmdargs, "-x")
		}
		logs.Log("Update args: " + strings.Join(cmdargs, " "))

		cmd := exec.CommandContext(ctx, cfg.Global.PathData.FactUpdaterShell, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			logs.Log("fact update check: download/check timed out... purging cache.")
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
						if logNoUpdate == true {
							mess := "fact update check: Factorio is up-to-date."
							logs.Log(mess)
						}
						return
					} else if strings.HasPrefix(line, "Wrote ") {
						if linelen > 1 && strings.Contains(line, ".zip") {

							//Only trigger on a new patch file
							if line != glob.NewPatchName {
								glob.NewPatchName = line

								if numwords > 1 &&
									CheckZip(words[1]) {
									mess := "Factorio update downloaded and verified, will update when no players are online."
									CMS(cfg.Local.ChannelData.ChatID, mess)
									WriteFact("/cchat [SYSTEM] " + mess)
									logs.Log(mess)
								} else {
									os.RemoveAll(cfg.Global.PathData.FactUpdateCache)
									//Purge patch name so we attempt check again
									glob.NewPatchName = constants.Unknown
									logs.Log("fact update check: Factorio update zip invalid... purging cache.")
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
							if glob.NewVersion != newversion {
								glob.NewVersion = newversion

								CMS(cfg.Local.ChannelData.ChatID, messdisc)

								WriteFact("/cchat [SYSTEM] " + messfact)
								logs.Log(messfact)
							}
						}
					}
				}
			}
		}

		logs.Log(fmt.Sprintf("fact update dry: update_fact.py:\n%v", out))
	}

}

func FactUpdate() {

	glob.UpdateFactorioLock.Lock()
	defer glob.UpdateFactorioLock.Unlock()

	//Give up on patching eventually
	ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
	defer cancel()

	os.MkdirAll(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactUpdateCache, 0777)

	if IsFactRunning() == false {
		//Keep us from stepping on a factorio launch or update
		glob.FactorioLaunchLock.Lock()
		defer glob.FactorioLaunchLock.Unlock()

		cmdargs := []string{cfg.Global.PathData.FactUpdaterPath, "-O", cfg.Global.PathData.FactUpdateCache, "-a", "../" + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + cfg.Global.PathData.FactorioBinary}
		if cfg.Local.UpdateFactExp {
			cmdargs = append(cmdargs, "-x")
		}
		logs.Log("Update args: " + strings.Join(cmdargs, " "))

		cmd := exec.CommandContext(ctx, cfg.Global.PathData.ShellPath, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			logs.Log("fact update: (error) Factorio update patching timed out, deleting possible corrupted patch file.")

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
						logs.Log(mess)
						return
					}
				}
			}
		} else {
			os.RemoveAll(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactUpdateCache)
			logs.Log("fact update: (error) Non-zero exit code... purging update cache.")
			logs.Log(fmt.Sprintf("fact update: (error) update_fact.py:\n%v", out))
			return
		}

		logs.Log("fact update: (unknown error): " + out)
		return
	} else {

		logs.Log("fact update: (error) Factorio is currently running, unable to update.")
		return
	}
}
