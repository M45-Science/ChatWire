package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"context"
	"io/ioutil"
	"os/exec"
	"strings"
)

func CheckMods(force bool) {
	if !cfg.Local.AutoModUpdate && !force {
		return
	}

	/* Update mods if needed */
	modPath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" +
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
		UpdateMods()
	}
}

func UpdateMods() {

	ctx, cancel := context.WithTimeout(context.Background(), constants.ModUpdateLimit)
	defer cancel()

	cmdargs := []string{
		cfg.Global.PathData.FactorioServersRoot + constants.ModUpdaterPath,
		"-u",
		cfg.Global.FactorioData.Username,
		"-t",
		cfg.Global.FactorioData.Token,
		"-s",
		cfg.Global.PathData.FactorioServersRoot +
			cfg.Global.PathData.FactorioHomePrefix +
			cfg.Local.ServerCallsign + "/" +
			constants.ServSettingsName,

		"-m",
		cfg.Global.PathData.FactorioServersRoot +
			cfg.Global.PathData.FactorioHomePrefix +
			cfg.Local.ServerCallsign + "/mods/",

		"--fact-path",
		fact.GetFactorioBinary(),
		"--update",
	}

	temp := strings.ReplaceAll(strings.Join(cmdargs, " "), cfg.Global.FactorioData.Token, "**private**")
	cwlog.DoLogCW(temp)
	cmd := exec.CommandContext(ctx, cfg.Global.PathData.FactUpdaterShell, cmdargs...)

	o, err := cmd.CombinedOutput()
	out := string(o)

	if err != nil {
		cwlog.DoLogCW("Error running mod updater: " + err.Error())
	}

	cwlog.DoLogCW(out)

	lines := strings.Split(out, "\n")
	buf := ""
	for _, line := range lines {
		if strings.Contains(line, "Download") {
			buf = buf + line + "\n"
		}
	}
	if buf != "" {
		fact.CMS(cfg.Local.ChannelData.ChatID, "Some Factorio mods were updated, and will take effect on the next reboot (when server is empty)")
		fact.SetQueued(true)
	}

}
