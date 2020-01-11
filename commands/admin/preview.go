package admin

import (
	"os/exec"
	"strings"
	"regexp"
	"fmt"

	"../../support"
	"github.com/bwmarrin/discordgo"
)

func Preview(s *discordgo.Session, m *discordgo.MessageCreate) {

	var filename = ""
	out, aerr := exec.Command(support.Config.Executable, " --generate-map-preview /home/fact/map-prev/").Output()

	if aerr != nil {
		support.ErrorLog(aerr)
	}

	lines := strings.Split(string(out), "\n")
	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			result := regexp.MustCompile(`(?m)Wrote map preview image file: \/var\/www\/html\/(.*)`)
			filename = result.ReplaceAllString(l, "http://bhmm.net/${1}")
		}
	}

	buffer := fmt.Sprintf("Preview: %s", filename)
	_, err := s.ChannelMessageSend(support.Config.FactorioChannelID, buffer)
	if err != nil {
		support.ErrorLog(err)
	}
	return
}
