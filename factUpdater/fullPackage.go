package factUpdater

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	baseDownloadURL = "https://www.factorio.com/get-download/"
	releaseURL      = "https://factorio.com/api/latest-releases"
	sha256URL       = "https://www.factorio.com/download/sha256sums/"
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

	glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Updating Factorio", "Downloading...", glob.COLOR_CYAN)
	cwlog.DoLogCW("Downloading: %v", url)

	data, filename, err = HttpGet(url, false)
	if err != nil {
		embed := &discordgo.MessageEmbed{Title: "ERROR:", Description: "Downloading Factorio failed."}
		disc.SmartWriteDiscordEmbed(cfg.Local.Channel.ChatChannel, embed)
		return errors.New("failed to get response: " + err.Error())
	}

	cwlog.DoLogCW("Download of %v complete, verifying...", filename)
	glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Updating Factorio", "Verifying update.", glob.COLOR_CYAN)

	hash, err := GetSHA256(filename)
	if err != nil {
		emsg := "unable to fetch SHA256 data." + err.Error()
		cwlog.DoLogCW(emsg)
		return errors.New(emsg)
	}
	if !CheckSHA256(data, hash) {
		emsg := "the download has an invalid checksum"
		cwlog.DoLogCW(emsg)
		return errors.New(emsg)
	} else {
		cwlog.DoLogCW("Download SHA256 verified!")
	}

	archive, err := checkFullPackage(data)
	if err != nil {
		return errors.New("checking full package failed: " + err.Error())
	}

	pathParts := strings.Split(fact.GetFactorioBinary(), "/")
	numParts := len(pathParts)
	if numParts < 4 {
		return errors.New("factorio's binary path does not seem valid. Make sure to include factorio/bin/x64/factorio")
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
		return fmt.Errorf("installing Factorio to '%v' failed: %v", factPath, err.Error())
	}

	fact.FactorioVersion = info.Version
	cwlog.DoLogCW("Factorio was installed to: %v", factPath)
	glob.UpdateMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.UpdateMessage, "Complete", "Factorio has been installed!", glob.COLOR_CYAN)
	fact.DoUpdateFactorio = false

	return nil
}

func GetSHA256(filename string) (string, error) {
	data, _, err := HttpGet(sha256URL, false)
	if err != nil {
		emsg := "Unable to fetch SHA256 sum data: " + err.Error()
		cwlog.DoLogCW(emsg)
		return "", errors.New(emsg)
	}
	sumsList := strings.Split(string(data), "\n")

	for _, sum := range sumsList {
		lineParts := strings.Split(sum, "  ")
		numparts := len(lineParts)

		if numparts == 2 {
			sum, file := lineParts[0], lineParts[1]
			if filename == file {
				return sum, nil
			}
		} else {
			emsg := "SHA256 data is invalid"
			cwlog.DoLogCW(emsg)
			return "", errors.New(emsg)
		}
	}

	emsg := "SHA256 not found for that file"
	cwlog.DoLogCW(emsg)
	return "", errors.New(emsg)
}

func CheckSHA256(data []byte, checkHash string) bool {

	hash := sha256.New()
	hash.Write([]byte(data))
	hashBytes := hash.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	return strings.EqualFold(hashString, checkHash)
}

func checkFullPackage(data []byte) ([]byte, error) {
	archive, err := unXZData(data)
	if err != nil {
		return nil, errors.New("checkFullPackage: xz failed to decompress:" + err.Error())
	}

	cwlog.DoLogCW("unxz complete, checking tar file.")
	err = checkInstallTar(archive)
	if err != nil {
		return nil, errors.New("checkFullPackage: tar verfied failed:" + err.Error())
	}
	cwlog.DoLogCW("Install package verified!")

	return archive, nil
}
