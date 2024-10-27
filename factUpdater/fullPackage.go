package factUpdater

import (
	"ChatWire/cwlog"
	"ChatWire/fact"
	"fmt"
	"strings"
)

const (
	baseDownloadURL = "https://www.factorio.com/get-download/"
	baseUpdateURL   = "https://updater.factorio.com/get-download-link?"
)

func fullPackage(info *infoData) error {
	var filename string
	var err error

	branch := "latest"
	if info.xreleases {
		branch = "experimental"
	}
	url := fmt.Sprintf("%v%v/%v/%v", baseDownloadURL, branch, info.gBuild, info.gDistro)
	genName := fmt.Sprintf("factorio_%v_%v_%v.cache", info.gBuild, info.gDistro, info.vInt.intToString())

	skipDownload := false
	var data []byte

	if !skipDownload {
		cwlog.DoLogCW("Downloading: %v", url)
		data, filename, err = httpGet(info, url)
		if err != nil {
			return fmt.Errorf("failed to get response: %v", err.Error())
		}
		cwlog.DoLogCW("Download of %v complete, verifying...", filename)
	} else {
		cwlog.DoLogCW("Verifying cached download: %v...", genName)
	}

	archive, err := checkFullPackage(info, data)
	if err != nil {
		return fmt.Errorf("checking full package failed: %v", err.Error())
	}

	pathParts := strings.Split(fact.GetFactorioBinary(), "/")
	numParts := len(pathParts)
	if numParts < 4 {
		return fmt.Errorf("factorio's binary path does not seem valid. Make sure to include factorio/bin/x64/factorio")
	}
	factPath := strings.Join(pathParts[:numParts-4], "/")

	err = untar(factPath, archive)
	if err != nil {
		return fmt.Errorf("installing factorio to '%v' failed: %v", factPath, err.Error())
	}

	cwlog.DoLogCW("Factorio was installed to: %v", factPath)
	return nil
}

func checkFullPackage(info *infoData, data []byte) ([]byte, error) {
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
