package fact

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

func CheckIfNewer(ca, cb, cc int) bool {

	if FactorioVersionA > ca &&
		FactorioVersionB > cb &&
		FactorioVersionC > cc {
		return true
	}
	return false
}

/* Check if Factorio update zip is valid */
func CheckZip(filename string) bool {
	read, err := zip.OpenReader(filename)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return false
	}

	for _, file := range read.File {
		_, err := file.Open()
		if err != nil {
			cwlog.DoLogCW(err.Error())
			return false
		}
		if file.UncompressedSize64 > 1024 {
			return true
		}
	}

	return false
}

/* Check if there is a new Factorio update */
func CheckFactUpdate(logNoUpdate bool) {

	if cfg.Global.Paths.Binaries.FactUpdater != "" {

		/* Give up on check/download after a while */
		ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
		defer cancel()

		/* Create cache directory */
		err := os.MkdirAll(GetUpdateCachePath(), 0777)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}

		cmdargs := []string{cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Binaries.FactUpdater, "-O", GetUpdateCachePath(), "-d", "-a", GetFactorioBinary()}
		if cfg.Local.Options.ExpUpdates {
			cmdargs = append(cmdargs, "-x")
		}
		/* cwlog.DoLogCW("Update args: " + strings.Join(cmdargs, " ")) */

		cmd := exec.CommandContext(ctx, cfg.Global.Paths.Binaries.UpdaterShell, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			cwlog.DoLogCW("fact update check: download/check timed out... purging cache.")
			os.RemoveAll(GetUpdateCachePath())
			return
		}

		if err == nil {
			clines := strings.Split(out, "\n")
			for _, line := range clines {
				linelen := len(line)
				var newversion, oldversion string

				if linelen > 0 {

					words := strings.Split(line, " ")
					numwords := len(words)

					if strings.HasPrefix(line, "No updates available") {
						if logNoUpdate {
							mess := "fact update check: Factorio is up-to-date."
							cwlog.DoLogCW(mess)
						}
						return
					} else if strings.HasPrefix(line, "Wrote ") {
						if linelen > 1 && strings.Contains(line, ".zip") {

							/* Only trigger on a new patch file */
							if line != NewPatchName {
								NewPatchName = line

								if numwords > 1 &&
									CheckZip(words[1]) {
									mess := "Factorio update downloaded and verified, will update when no players are online."
									CMS(cfg.Local.Channel.ChatChannel, mess)
									FactChat("[SYSTEM] " + mess)

									cwlog.DoLogCW(mess)
								} else {
									if glob.UpdateZipAttempts < constants.MaxUpdateZipAttempts {
										glob.UpdateZipAttempts++
										buf := fmt.Sprintf("Waiting for zipfile to finish...(%v/%v)", glob.UpdateZipAttempts, constants.MaxUpdateZipAttempts)
										cwlog.DoLogCW(buf)
										time.Sleep(constants.UpdateZipInterval) //Wait a bit
										return
									}
									//Reset counter
									glob.UpdateZipAttempts = 0

									os.RemoveAll(GetUpdateCachePath())
									/* Purge patch name so we attempt check again */
									NewPatchName = constants.Unknown
									cwlog.DoLogCW("fact update check: Factorio update zip invalid... purging cache.")
								}
							}
							return
						}
					} else if strings.HasPrefix(line, "Dry run: would have fetched update from") {
						if numwords >= 9 {
							oldversion = words[7]
							newversion = words[9]

							messdisc := "**Factorio update available!**"
							messfact := "Factorio update available!"
							DoUpdateFactorio = true

							newDigits := strings.Split(newversion, ".")
							oldDigits := strings.Split(oldversion, ".")

							if len(newDigits) < 2 || len(oldDigits) < 2 {
								return
							}

							if !cfg.Local.Options.ExpUpdates {
								if newDigits[0] != oldDigits[0] {
									//Full revision, only manually update.
									buf := fmt.Sprintf("Factorio update '%v' to '%v' is a full revision, disabling auto-update.", oldversion, newversion)
									CMS(cfg.Local.Channel.ChatChannel, buf)
									cfg.Local.Options.AutoUpdate = false
									return
								}
								if newDigits[1] != oldDigits[1] {
									//Major revision, only manually update.
									buf := fmt.Sprintf("Factorio update '%v' to '%v' is a major revision, disabling auto-update.", oldversion, newversion)
									CMS(cfg.Local.Channel.ChatChannel, buf)
									cfg.Local.Options.AutoUpdate = false
									return
								}
							}

							/* Don't message, unless this is actually a unique new version */
							if NewVersion != newversion {
								NewVersion = newversion

								CMS(cfg.Local.Channel.ChatChannel, messdisc)

								FactChat("[SYSTEM] " + messfact)
								cwlog.DoLogCW(messfact)
							}
						}
					}
				}
			}
		}

		cwlog.DoLogCW(fmt.Sprintf("fact update dry: update_fact.py:\n%v", out))
	}

}

/* Update Factorio */
func FactUpdate() {

	/* Give up on patching eventually */
	ctx, cancel := context.WithTimeout(context.Background(), constants.FactorioUpdateCheckLimit)
	defer cancel()

	err := os.MkdirAll(GetUpdateCachePath(), 0777)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	if !FactIsRunning && !FactorioBooted {
		/* Keep us from stepping on a factorio launch or update */

		cmdargs := []string{cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Binaries.FactUpdater, "-O", GetUpdateCachePath(), "-a", GetFactorioBinary()}
		if cfg.Local.Options.ExpUpdates {
			cmdargs = append(cmdargs, "-x")
		}

		cmd := exec.CommandContext(ctx, cfg.Global.Paths.Binaries.UpdaterShell, cmdargs...)
		o, err := cmd.CombinedOutput()
		out := string(o)

		if ctx.Err() == context.DeadlineExceeded {
			cwlog.DoLogCW("fact update: (error) Factorio update patching timed out, deleting possible corrupted patch file.")

			os.RemoveAll(GetUpdateCachePath())
			return
		}

		if err == nil {
			clines := strings.Split(out, "\n")
			for _, line := range clines {
				linelen := len(line)

				if linelen > 0 {

					if strings.HasPrefix(line, "Update applied successfully!") {
						mess := "fact update: Factorio updated successfully!"
						cwlog.DoLogCW(mess)
						return
					}
				}
			}
		} else {
			os.RemoveAll(GetUpdateCachePath())
			cwlog.DoLogCW("fact update: (error) Non-zero exit code... purging update cache.")
			cwlog.DoLogCW(fmt.Sprintf("fact update: (error) update_fact.py:\n%v", out))
			return
		}

		cwlog.DoLogCW("fact update: (unknown error): " + out)
		return
	} else {

		cwlog.DoLogCW("fact update: (error) Factorio is currently running, unable to update.")
		return
	}
}
