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

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

/* RandomMap locks FactorioLaunchLock */
func RandomMap(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string) {

	if fact.IsFactRunning() {
		fact.CMS(m.ChannelID, "Stop server first! ($stop)")
		return
	}

	fact.FactorioLaunchLock.Lock()
	defer fact.FactorioLaunchLock.Unlock()

	fact.CMS(m.ChannelID, "Generating map preview...")

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
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to generate the preview. Details: %s", aerr))
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			preview_made = true
		}
	}

	imgargs := []string{path, "-quality", cfg.Global.Options.PreviewSettings.JPGQuality, "-scale", cfg.Global.Options.PreviewSettings.JPGScale, jpgpath}
	cmdb := exec.Command(cfg.Global.Paths.Binaries.ImgCmd, imgargs...)
	_, berr := cmdb.CombinedOutput()

	/* Delete PNG, we don't need it now */
	if err := os.Remove(path); err != nil {
		cwlog.DoLogCW("png preview file not found...")
	}

	if berr != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to convert the map preview. Details: %s", berr))
	}

	buffer := "Preview failed."
	if preview_made {
		buffer = fmt.Sprintf("**Map code:** `%v`\nPreview: %s%s.jpg\n", ourcode, cfg.Global.Paths.URLs.MapPreviewURL, ourcode)
	}

	fact.CMS(m.ChannelID, buffer)
}
