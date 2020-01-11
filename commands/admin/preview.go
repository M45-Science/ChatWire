package admin

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"../../support"
	"github.com/bwmarrin/discordgo"
)

func Preview(s *discordgo.Session, m *discordgo.MessageCreate) {

	var filename = ""
	cmd := exec.Command("/home/fact/fact-prev/bin/x64/factorio", "--generate-map-preview /home/fact/map-prev/"...)
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		support.ErrorLog(aerr)
	}

	support.Log("Output:")
	support.Log(string(out))

	lines := strings.Split(string(out), "\n")
	support.Log("Looking for preview line...")
	buf := fmt.Sprintf("Found %d lines...", len(lines))
	support.Log(buf)

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			result := regexp.MustCompile(`(?m)Wrote map preview image file: \/home\/fact\/(.*)`)
			filename = result.ReplaceAllString(l, "http://bhmm.net/${1}")
			support.Log("Found preview line.")
		}
	}

	buffer := "Preview failed."
	if filename != "" {
		buffer = fmt.Sprintf("Preview: %s", filename)
	}

	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}
	return
}
