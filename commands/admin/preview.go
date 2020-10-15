package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"../../config"
	"../../constants"
	"../../fact"
	"../../glob"
	"../../logs"
	"github.com/bwmarrin/discordgo"
)

func GetMapTypeNum(mapt string) int {
	i := 0

	if config.Config.MapGenJson != "" {
		return 0
	}
	for i = 0; i < glob.MaxMapTypes; i = i + 1 {
		if strings.EqualFold(constants.MapTypes[i], mapt) {
			return i
		}
	}
	return -1
}

//RandomMap locks FactorioLaunchLock
func RandomMap(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string) {

	if fact.IsFactRunning() {
		fact.CMS(m.ChannelID, "Stop server first! ($stop)")
		return
	}

	glob.FactorioLaunchLock.Lock()
	defer glob.FactorioLaunchLock.Unlock()

	fact.CMS(m.ChannelID, "Generating map preview...")

	var filename = ""
	t := time.Now()
	ourseed := uint64(t.UnixNano())
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, ourseed)
	glob.LastMapSeed = ourseed
	ourcode := fmt.Sprintf("%02d%v", GetMapTypeNum(config.Config.MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	glob.LastMapCode = ourcode

	path := fmt.Sprintf("%s%s.png", config.Config.PreviewPath, ourcode)
	jpgpath := fmt.Sprintf("%s%s.jpg", config.Config.PreviewPath, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + config.Config.PreviewRes, "--map-preview-scale=" + config.Config.PreviewScale, "--preset", config.Config.MapPreset, "--map-gen-seed", fmt.Sprintf("%v", ourseed), config.Config.PreviewArgs}

	//Append map gen if set
	if config.Config.MapGenJson != "" {
		args = append(args, "--map-gen-settings")
		args = append(args, config.Config.MapGenJson)
	}

	cmd := exec.Command(config.Config.Executable, args...)
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to generate the preview. Details: %s", aerr))
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			//plug-in PreviewPath in the future?
			result := regexp.MustCompile(`(?m).*Wrote map preview image file: \/home\/fact\/public_html\/(.*)`)
			filename = result.ReplaceAllString(l, config.Config.SiteURL+"${1}")
		}
	}

	imgargs := []string{path, "-quality", config.Config.JpgQuality, "-scale", config.Config.JpgScale, jpgpath}
	cmdb := exec.Command(config.Config.ConvertExec, imgargs...)
	_, berr := cmdb.CombinedOutput()

	//Delete PNG, we don't need it now
	if err := os.Remove(path); err != nil {
		logs.Log("png preview file not found...")
	}

	if berr != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to convert the map preview. Details: %s", berr))
	}

	buffer := "Preview failed."
	if filename != "" {
		buffer = fmt.Sprintf("**Map code:** `%v`\nPreview: %s%s.jpg\n", ourcode, config.Config.SiteURL, ourcode)
	}

	fact.CMS(m.ChannelID, buffer)
}
