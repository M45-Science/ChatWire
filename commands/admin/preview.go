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

	"../../cfg"
	"../../fact"
	"../../glob"
	"../../logs"
	"github.com/bwmarrin/discordgo"
)

//RandomMap locks FactorioLaunchLock
func RandomMap(s *discordgo.Session, m *discordgo.MessageCreate, arguments []string) {

	if fact.IsFactRunning() {
		fact.CMS(m.ChannelID, "Stop server first! ($stop)")
		return
	}

	glob.FactorioLaunchLock.Lock()
	defer glob.FactorioLaunchLock.Unlock()

	fact.CMS(m.ChannelID, "Generating map preview...")

	var preview_made = false
	t := time.Now()
	ourseed := uint64(t.UnixNano())
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, ourseed)
	glob.LastMapSeed = ourseed
	ourcode := fmt.Sprintf("%02d%v", fact.GetMapTypeNum(cfg.Local.MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	glob.LastMapCode = ourcode

	path := fmt.Sprintf("%s%s.png", cfg.Global.PathData.MapPreviewPath, ourcode)
	jpgpath := fmt.Sprintf("%s%s.jpg", cfg.Global.PathData.MapPreviewPath, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + cfg.Global.MapPreviewData.Res, "--map-preview-scale=" + cfg.Global.MapPreviewData.Scale, "--preset", cfg.Local.MapPreset, "--map-gen-seed", fmt.Sprintf("%v", ourseed), cfg.Global.MapPreviewData.Args}

	//Append map gen if set
	if cfg.Local.MapGenPreset != "" {
		args = append(args, "--map-gen-settings")
		args = append(args, cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.MapGenPath+"/"+cfg.Local.MapGenPreset+"-gen.json")
	}

	//Append map settings if set
	if cfg.Local.MapGenPreset != "" {
		args = append(args, "--map-settings")
		args = append(args, cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.MapGenPath+"/"+cfg.Local.MapGenPreset+"-set.json")
	}

	cmd := exec.Command(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactorioHomePrefix+cfg.Local.ServerCallsign+"/"+cfg.Global.PathData.FactorioBinary, args...)
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to generate the preview. Details: %s", aerr))
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			preview_made = true
		}
	}

	imgargs := []string{path, "-quality", cfg.Global.MapPreviewData.JPGQuality, "-scale", cfg.Global.MapPreviewData.JPGScale, jpgpath}
	cmdb := exec.Command(cfg.Global.PathData.ImageMagickPath, imgargs...)
	_, berr := cmdb.CombinedOutput()

	//Delete PNG, we don't need it now
	if err := os.Remove(path); err != nil {
		logs.Log("png preview file not found...")
	}

	if berr != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to convert the map preview. Details: %s", berr))
	}

	buffer := "Preview failed."
	if preview_made {
		buffer = fmt.Sprintf("**Map code:** `%v`\nPreview: %s%s.jpg\n", ourcode, cfg.Global.PathData.MapPreviewURL, ourcode)
	}

	fact.CMS(m.ChannelID, buffer)
}
