package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
)

/* RandomMap locks FactorioLaunchLock */
func NewMapPrev(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.IsFactRunning() {
		buf := "Factorio is currently, running. You must stop the game first. See /stop-factorio"
		embed := &discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return
	}

	fact.FactorioLaunchLock.Lock()
	defer fact.FactorioLaunchLock.Unlock()

	buff := "Generating map preview..."
	embed := &discordgo.MessageEmbed{Title: "Status:", Description: buff}
	disc.InteractionResponse(s, i, embed)

	var preview_made = false
	t := time.Now()
	ourseed := uint64(t.UnixNano())
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, ourseed)
	fact.LastMapSeed = ourseed
	MapPreset := cfg.Local.Settings.MapPreset
	ourcode := fmt.Sprintf("%02d%v", fact.GetMapTypeNum(cfg.Local.Settings.MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	fact.LastMapCode = ourcode

	path := fmt.Sprintf("%s%s.png", cfg.Global.Paths.Folders.MapPreviews, ourcode)
	jpgpath := fmt.Sprintf("%s%s.jpg", cfg.Global.Paths.Folders.MapPreviews, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + cfg.Global.Options.PreviewSettings.PNGRes, "--map-preview-scale=" + cfg.Global.Options.PreviewSettings.PNGScale, "--map-gen-seed", fmt.Sprintf("%v", ourseed), cfg.Global.Options.PreviewSettings.Arguments}

	/* Append map gen if set */
	if cfg.Local.Settings.MapGenerator != "" {
		args = append(args, "--map-gen-settings")
		args = append(args, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-gen.json")

		args = append(args, "--map-settings")
		args = append(args, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-set.json")
	} else {
		args = append(args, "--preset")
		args = append(args, MapPreset)
	}

	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", fact.GetFactorioBinary(), strings.Join(args, " "))
	cwlog.DoLogCW(lbuf)
	cmd := exec.Command(fact.GetFactorioBinary(), args...)

	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		buf := fmt.Sprintf("An error occurred when attempting to generate the map preview: %s", aerr)
		cwlog.DoLogCW(buf)
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			preview_made = true
		}
	}
	if !preview_made {
		buf := "The game did not generate a map preview. Try clearing map-gen."
		cwlog.DoLogCW(buf)
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
		return
	}

	imgargs := []string{path, "-quality", cfg.Global.Options.PreviewSettings.JPGQuality, "-scale", cfg.Global.Options.PreviewSettings.JPGScale, jpgpath}
	cmdb := exec.Command(cfg.Global.Paths.Binaries.ImgCmd, imgargs...)
	_, berr := cmdb.CombinedOutput()

	/* Delete PNG, we don't need it now */
	if err := os.Remove(path); err != nil {
		cwlog.DoLogCW("png preview file not found...")
	}

	if berr != nil {
		buf := fmt.Sprintf("An error occurred when attempting to convert the map preview: %s", berr)
		cwlog.DoLogCW(buf)
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
		return
	}

	url := fmt.Sprintf("%v%v.jpg", cfg.Global.Paths.URLs.MapPreviewURL, ourcode)
	img := &discordgo.MessageEmbedImage{URL: url}
	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "Map preview:", Image: img})
	f := discordgo.WebhookParams{Embeds: elist}
	disc.FollowupResponse(s, i, &f)
}
