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
	ourseed := fmt.Sprintf("%s", t.Unix())

	path := fmt.Sprintf("/home/fact/map-prev/%d.png", ourseed)
	strseed := fmt.Sprintf("%d", ourseed)

	cmd := exec.Command("/home/fact/fact-prev/bin/x64/factorio", "--generate-map-preview", path, "--preset", "rail-world", "--map-gen-seed", strseed)
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		support.ErrorLog(aerr)
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			result := regexp.MustCompile(`(?m).*Wrote map preview image file: \/home\/fact\/(.*)`)
			filename = result.ReplaceAllString(l, "http://bhmm.net/${1}")
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
