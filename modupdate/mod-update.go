package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

func CheckMods(force bool, doReport bool) {
	if !cfg.Local.Options.AutoUpdate && !force {
		return
	}

	/* Update mods if needed */
	modPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.FactorioPrefix + cfg.Local.Callsign + "/" +
		constants.ModsFolder + "/"
	mCount := 0

	files, err := ioutil.ReadDir(modPath)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			mCount++
		}
	}

	/* There are mods, so check for updates */
	if mCount > 0 {
		UpdateMods(doReport)
	}
}

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
			cfg.Global.Paths.FactorioPrefix +
			cfg.Local.Callsign + "/" +
			constants.ServSettingsName,

		"-m",
		cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.FactorioPrefix +
			cfg.Local.Callsign + "/mods/",

		"--fact-path",
		fact.GetFactorioBinary(),
		"--update",
	}

	//temp := strings.ReplaceAll(strings.Join(cmdargs, " "), cfg.Global.FactorioData.Token, "**private**")
	//cwlog.DoLogCW(temp)
	cmd := exec.CommandContext(ctx, cfg.Global.Paths.Binaries.UpdaterShell, cmdargs...)

	o, err := cmd.CombinedOutput()
	out := string(o)

	if err != nil {
		buf := fmt.Sprintf("Error while attempting to update game mods: %v", err.Error())
		cwlog.DoLogCW(buf)
		fact.CMS(cfg.Local.Channel.ChatChannel, buf)
	} else {
		//cwlog.DoLogCW(out)
	}

	lines := strings.Split(out, "\n")
	buf := ""
	for _, line := range lines {
		if strings.Contains(line, "Download") {
			buf = buf + line + "\n"
		}
	}
	if buf != "" {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Some Factorio mods were updated, and will take effect on the next reboot (when server is empty)")
		fact.SetQueued(true)
	} else if doReport {
		fact.CMS(cfg.Local.Channel.ChatChannel, "All game mods are up-to-date.")
	}

}
