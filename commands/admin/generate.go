package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/bwmarrin/discordgo"
)

/* Generate map */
func Generate(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if fact.IsFactRunning() {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Stop server first! ($stop)")
		return
	}

	fact.FactorioLaunchLock.Lock()
	defer fact.FactorioLaunchLock.Unlock()

	t := time.Now()
	ourseed := uint64(t.UnixNano())

	MapPreset := cfg.Local.Settings.MapPreset

	if fact.LastMapSeed > 0 {
		ourseed = fact.LastMapSeed
	}

	//Use seed if specified, then clear it
	if cfg.Local.Settings.Seed > 0 {
		ourseed = cfg.Local.Settings.Seed
		cfg.Local.Settings.Seed = 0
		cfg.WriteLCfg()
	}

	if ourseed <= 0 {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Error, no seed.")
		return
	}

	if MapPreset == "Error" {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Invalid map preset.")
		return
	}

	fact.CMS(cfg.Local.Channel.ChatChannel, "Generating map...")

	/* Delete old sav-* map to save space */
	fact.DeleteOldSav()

	/* Generate code to make filename */
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, ourseed)
	ourcode := fmt.Sprintf("%02d%v", fact.GetMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	filename := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.FactorioPrefix + cfg.Local.Callsign + "/" + cfg.Global.Paths.Folders.Saves + "/gen-" + ourcode + ".zip"

	factargs := []string{"--map-gen-seed", fmt.Sprintf("%v", ourseed), "--create", filename}

	/* Append map gen if set */
	if cfg.Local.Settings.MapGenerator != "" {
		factargs = append(factargs, "--map-gen-settings")
		factargs = append(factargs, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-gen.json")

		factargs = append(factargs, "--map-settings")
		factargs = append(factargs, cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.Folders.MapGenerators+"/"+cfg.Local.Settings.MapGenerator+"-set.json")
	} else {
		factargs = append(factargs, "--preset")
		factargs = append(factargs, MapPreset)
	}

	lbuf := fmt.Sprintf("EXEC: %v ARGS: %v", fact.GetFactorioBinary(), strings.Join(factargs, " "))
	cwlog.DoLogCW(lbuf)

	cmd := exec.Command(fact.GetFactorioBinary(), factargs...)
	out, aerr := cmd.CombinedOutput()

	if aerr != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred attempting to generate the map. Details: %s", aerr))
	}

	glob.VoteBoxLock.Lock()
	glob.VoteBox.LastRewindTime = time.Now()
	fact.VoidAllVotes()    /* Void all votes */
	fact.ResetTotalVotes() /* New map, reset player's vote limits */
	fact.WriteRewindVotes()
	glob.VoteBoxLock.Unlock()

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Creating new map") {
			fact.CMS(cfg.Local.Channel.ChatChannel, "New map saved as: "+filename)
			return
		}
	}

	fact.CMS(cfg.Local.Channel.ChatChannel, "Unknown error.")
}
