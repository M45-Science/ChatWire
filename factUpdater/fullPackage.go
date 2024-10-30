package factUpdater

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	baseDownloadURL = "https://www.factorio.com/get-download/"
	baseUpdateURL   = "https://updater.factorio.com/get-download-link?"
	releaseURL      = "https://factorio.com/api/latest-releases"
)

func fullPackage(info *InfoData) error {
	var filename string
	var err error

	branch := "stable"
	if info.Xreleases {
		branch = "latest"
	}
	url := fmt.Sprintf("%v%v/%v/%v", baseDownloadURL, branch, info.Build, info.Distro)
	var data []byte

	cwlog.DoLogCW("Downloading: %v", url)

	data, filename, err = httpGet(url)
	if err != nil {
		embed := &discordgo.MessageEmbed{Title: "ERROR:", Description: "Downloading Factorio failed."}
		disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, embed)
		return fmt.Errorf("failed to get response: %v", err.Error())
	}

	cwlog.DoLogCW("Download of %v complete, verifying...", filename)

	archive, err := checkFullPackage(data)
	if err != nil {
		return fmt.Errorf("checking full package failed: %v", err.Error())
	}

	pathParts := strings.Split(fact.GetFactorioBinary(), "/")
	numParts := len(pathParts)
	if numParts < 4 {
		return fmt.Errorf("factorio's binary path does not seem valid. Make sure to include factorio/bin/x64/factorio")
	}
	factPath := strings.Join(pathParts[:numParts-4], "/")

	fact.DoUpdateFactorio = true
	fact.WaitFactQuit(true)

	err = os.RemoveAll(factPath + "/factorio/bin")
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	err = os.RemoveAll(factPath + "/factorio/config")
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	err = os.RemoveAll(factPath + "/factorio/data")
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	err = os.RemoveAll(factPath + "/factorio/temp")
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
	err = os.RemoveAll(factPath + "/factorio/bin")
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}

	err = untar(factPath, archive)
	if err != nil {
		return fmt.Errorf("installing factorio to '%v' failed: %v", factPath, err.Error())
	}

	cwlog.DoLogCW("Factorio was installed to: %v", factPath)
	fact.DoUpdateFactorio = false
	return nil
}

func checkFullPackage(data []byte) ([]byte, error) {
	archive, err := unXZData(data)
	if err != nil {
		return nil, fmt.Errorf("checkFullPackage: xz failed to decompress: %v", err.Error())
	}

	cwlog.DoLogCW("unxz complete, checking tar file.")
	err = checkInstallTar(archive)
	if err != nil {
		return nil, fmt.Errorf("checkFullPackage: tar verfied failed: %v", err.Error())
	}
	cwlog.DoLogCW("Install package verified!")

	return archive, nil
}
