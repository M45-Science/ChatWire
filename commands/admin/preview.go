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

	var filename = ""
	t := time.Now()
	ourseed := fmt.Sprintf("%v", t.UnixNano())
	path := fmt.Sprintf("%s%s.png", support.Config.PreviewPath, ourseed)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + support.Config.PreviewRes, "--preset", support.Config.MapPreset, "--map-gen-seed", ourseed, support.Config.PreviewArgs}

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

	buffer := "Preview failed."
	if filename != "" {
		buffer = fmt.Sprintf("MapName: %s-%s.zip, Seed: %s\nPreview: %s", support.Config.MapPreset, ourseed, ourseed, filename)
	}

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}

	return
}
