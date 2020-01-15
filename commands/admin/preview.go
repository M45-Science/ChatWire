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

	"../../glob"
	"../../support"
	"github.com/bwmarrin/discordgo"
)

func GetMapType(mapt string) int {
	i := 0

	for i = 0; i < glob.MaxMapTypes; i = i + 1 {
		if strings.ToLower(glob.MapTypes[i]) == strings.ToLower(mapt) {
			return i
		}
	}
	return -1
}

func RandomMap(s *discordgo.Session, m *discordgo.MessageCreate) {
	glob.MapPrevLock.Lock()
	defer glob.MapPrevLock.Unlock()

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Generating map preview...")
	if err != nil {
		support.ErrorLog(err)
	}

	var filename = ""
	t := time.Now()
	ourseed := t.UnixNano()
	//ourseed = 1
	buf := new(bytes.Buffer)
	errb := binary.Write(buf, binary.BigEndian, ourseed)
	ourcode := fmt.Sprintf("%v%v", GetMapType(support.Config.MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))

	if errb != nil {
		support.ErrorLog(err)
	}

	path := fmt.Sprintf("%s%s.png", support.Config.PreviewPath, ourcode)
	jpgpath := fmt.Sprintf("%s%s.jpg", support.Config.PreviewPath, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + support.Config.PreviewRes, "--map-preview-scale=" + support.Config.PreviewScale, "--preset", support.Config.MapPreset, "--map-gen-seed", fmt.Sprintf("%v", ourseed), support.Config.PreviewArgs}

	cmd := exec.Command(support.Config.MapGenExec, args...)
	support.Log(fmt.Sprintf("\nRan: %s %s", support.Config.MapGenExec, strings.Join(args, " ")))
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		support.ErrorLog(aerr)
	}

	lines := strings.Split(string(out), "\n")
	//support.Log(lines[0])

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			result := regexp.MustCompile(`(?m).*Wrote map preview image file: \/home\/fact\/(.*)`)
			filename = result.ReplaceAllString(l, "http://bhmm.net/${1}")
		}
	}

	//convert X.png -quality X -scale xXy test.jpg
	imgargs := []string{path, "-quality", support.Config.JpgQuality, "-scale", support.Config.JpgScale, jpgpath}
	cmdb := exec.Command(support.Config.ConvertExec, imgargs...)
	support.Log(fmt.Sprintf("\nRan: %s %s", support.Config.ConvertExec, strings.Join(imgargs, " ")))
	bout, berr := cmdb.CombinedOutput()
	if bout != nil {
		//support.Log(string(bout))
	}

	//Delete PNG, we don't need it now
	if err := os.Remove(path); err != nil {
		support.Log("png preview file not found...")
	}

	if berr != nil {
		support.ErrorLog(aerr)
	}

	buffer := "Preview failed."
	if filename != "" {
		//buffer = fmt.Sprintf("**Map code:** `%v`\nSeed: `%v`, Map preset: `%s`\nPreview: http://bhmm.net/map-prev/%s.jpg\n", ourcode, ourseed, support.Config.MapPreset, ourcode)
		buffer = fmt.Sprintf("**Map code:** `%v`\nPreview: http://bhmm.net/map-prev/%s.jpg\n", ourcode, ourcode)
	}

	_, err = s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}

	return
}
