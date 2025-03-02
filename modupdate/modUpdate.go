package modupdate

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

/* Read entire mod folder */
func CheckMods(force bool, doReport bool) {

	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	glob.ModLock.Lock()
	defer glob.ModLock.Unlock()

	/* Update mods if needed */
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	files, err := os.ReadDir(modPath)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			UpdateMods(doReport)
			break
		}
	}
}

/* Using external mod updater, update mods */
func UpdateMods(doReport bool) {

	ctx, cancel := context.WithTimeout(context.Background(), constants.ModUpdateLimit)
	defer cancel()

	cmdargs := []string{
		cfg.Global.Paths.Folders.ServersRoot + constants.ModUpdaterPath,
		"-u",
		cfg.Global.Factorio.Username,
		"-t",
		cfg.Global.Factorio.Token,
		"-s",
		cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			constants.ServSettingsName,

		"-m",
		cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			constants.ModsFolder + "/",

		"--fact-path",
		fact.GetFactorioBinary(),
		"--update",
	}

	cmd := exec.CommandContext(ctx, cfg.Global.Paths.Binaries.UpdaterShell, cmdargs...)

	o, err := cmd.CombinedOutput()
	out := string(o)

	if err != nil {
		buf := fmt.Sprintf("Error while attempting to update game mods: %v: %v", err.Error(), out)
		cwlog.DoLogCW(buf)
		return
	}

	lines := strings.Split(out, "\n")
	updated := 0
	updateResult := ""
	for _, line := range lines {
		if strings.Contains(line, "Download") {
			cwlog.DoLogCW("mod updater: " + line)

			cLine := regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
			words := strings.Split(cLine, " ")

			if len(words) >= 7 {
				if updateResult != "" {
					updateResult = updateResult + ", "
				}
				if strings.EqualFold(words[2], "Download") && strings.EqualFold(words[3], "Success") {
					updateResult = updateResult + fmt.Sprintf("%v (%v)", words[0], words[1])
					updated++
				}
			}
		}
	}
	if updated > 0 {
		if fact.NumPlayers > 0 {
			buf := fmt.Sprintf("**Factorio mods updated:** %v\nThis will take effect on the next reboot (when server is empty)", updateResult)
			fact.LogGameCMS(true, cfg.Local.Channel.ChatChannel, buf)
		} else {
			buf := fmt.Sprintf("**Factorio mods updated:** %v\nRebooting.", updateResult)
			fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, buf)
		}
		fact.QueueFactReboot = true
	} else if doReport {
		fact.CMS(cfg.Local.Channel.ChatChannel, "All game mods are up-to-date.")
	}

}
