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

func Preview(s *discordgo.Session, m *discordgo.MessageCreate) {

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, "Generating map preview...")
	if err != nil {
		support.ErrorLog(err)
	}

	var filename = ""
	t := time.Now()
	ourseed := t.UnixNano()
	buf := new(bytes.Buffer)
	errb := binary.Write(buf, binary.LittleEndian, ourseed)
	ourcode := fmt.Sprint("%s%v", support.Config.MapPreset, base64.StdEncoding.EncodeToString(buf.Bytes()))

	if errb != nil {
		support.ErrorLog(err)
	}

	glob.NewMaps[glob.NewMapLast] = ourcode

	//Handle max maps
	if glob.NewMapLast > glob.MaxMaps {
		glob.NewMapLast = 0
	} else {
		glob.NewMapLast = glob.NewMapLast + 1
	}

	path := fmt.Sprintf("%s%s.png", support.Config.PreviewPath, ourcode)
	jpgpath := fmt.Sprintf("%s%s.jpg", support.Config.PreviewPath, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + support.Config.PreviewRes, "--map-preview-scale=" + support.Config.PreviewScale, "--preset", support.Config.MapPreset, "--map-gen-seed", ourcode, support.Config.PreviewArgs}

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

	//convert 1578776871716251163.png -quality 70 -scale 768x768 test.jpg
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
		buffer = fmt.Sprintf("Map code: %v, Seed: %s\nMap preset: %s\nPreview: http://bhmm.net/map-prev/%s.jpg\n", glob.NewMapLast, ourcode, support.Config.MapPreset, ourcode)
	}

	_, err = s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}

	return
}
