package admin

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

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
	ourseed := fmt.Sprintf("%v", t.UnixNano())
	path := fmt.Sprintf("%s%s.png", support.Config.PreviewPath, ourseed)
	jpgpath := fmt.Sprintf("%s%s.jpg", support.Config.PreviewPath, ourseed)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + support.Config.PreviewRes, "--map-preview-scale=" + support.Config.PreviewScale, "--preset", support.Config.MapPreset, "--map-gen-seed", ourseed, support.Config.PreviewArgs}

	cmd := exec.Command(support.Config.MapGenExec, args...)
	support.Log(fmt.Sprintf("Ran: %s %s", support.Config.MapGenExec, strings.Join(args, " ")))
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
	imgargs := []string{path, "-quality " + support.Config.JpgQuality, "-scale " + support.Config.JpgScale, jpgpath}
	_ = exec.Command(support.Config.ConvertExec, imgargs...)
	support.Log(fmt.Sprintf("Ran: %s %s", support.Config.ConvertExec, strings.Join(args, " ")))

	buffer := "Preview failed."
	if filename != "" {
		buffer = fmt.Sprintf("MapName: %s-%s.zip\nSeed: %s\nPreview: http://bhmm.net/map-prev/%s.jpg", support.Config.MapPreset, ourseed, ourseed, ourseed)
	}

	_, err = s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}

	return
}
