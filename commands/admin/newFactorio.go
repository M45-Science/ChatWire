package admin

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modupdate"
)

func Factorio(s *discordgo.Session, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()
			if strings.EqualFold(arg, "start") {
				startFact(s, i)
				return
			} else if strings.EqualFold(arg, "stop") {
				stopFact(s, i)
				return
			} else if strings.EqualFold(arg, "new-map-preview") {
				newMapPrev(s, i)
				return
			} else if strings.EqualFold(arg, "new-map") {
				makeNewMap(s, i)
				return
			} else if strings.EqualFold(arg, "update-factorio") {
				updateFact(s, i)
				return
			} else if strings.EqualFold(arg, "update-mods") {
				updateMods(s, i)
				return
			} else if strings.EqualFold(arg, "archive-map") {
				archiveMap(s, i)
				return
			} else if strings.EqualFold(arg, "install-factorio") {
				installFactorio(s, i)
			} else if strings.EqualFold(arg, "map-schedule") {
				resetSchedule(s, i)
			}
		}
	}
}

func resetSchedule(s *discordgo.Session, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(s, i, "Status:", "Schedule reset.")
}

/* RandomMap locks FactorioLaunchLock */
func newMapPrev(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if fact.FactorioBooted || fact.FactIsRunning {
		buf := "Factorio is currently, running. You must stop the game first. See /stop-factorio"
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}

	/* Make directory if it does not exist */
	newdir := fmt.Sprintf("%s/", cfg.Global.Paths.Folders.MapPreviews)
	err := os.MkdirAll(newdir, os.ModePerm)
	if err != nil {
		buf := fmt.Sprintf("Unable to create map preview directory: %v", err.Error())
		cwlog.DoLogCW(buf)
		elist := discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, &elist)
		return
	}

	var preview_made = false
	t := time.Now()
	ourseed := int(t.UnixNano() - constants.CWEpoch)

	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.BigEndian, uint64(ourseed))
	fact.LastMapSeed = ourseed
	MapPreset := cfg.Local.Settings.MapPreset
	ourcode := fmt.Sprintf("%02d%v", fact.GetMapTypeNum(cfg.Local.Settings.MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	fact.LastMapCode = ourcode

	path := fmt.Sprintf("%s%s.png", cfg.Global.Paths.Folders.MapPreviews, ourcode)
	args := []string{"--generate-map-preview", path, "--map-preview-size=" + cfg.Global.Options.PreviewSettings.PNGRes, "--map-preview-scale=" + cfg.Global.Options.PreviewSettings.PNGScale, "--map-gen-seed", fmt.Sprintf("%v", ourseed), cfg.Global.Options.PreviewSettings.Arguments}

	/* Append map gen if set */
	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
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
		elist := discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, &elist)
	}

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Wrote map preview image file:") {
			preview_made = true
		}
	}
	if !preview_made {
		buf := "The game did not generate a map preview."
		cwlog.DoLogCW(buf)
		elist := discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, &elist)
		return
	}

	//Attempt to attach a map preview
	to, errb := os.OpenFile(path, os.O_RDONLY, 0666)
	if errb != nil {
		buf := fmt.Sprintf("Unable to read png file: %v", errb)
		cwlog.DoLogCW(buf)

		elist := discordgo.MessageEmbed{Title: "Error:", Description: buf}
		disc.InteractionResponse(s, i, &elist)
		return
	}
	defer to.Close()

	/* Delete PNG, we don't need it now */
	if err := os.Remove(path); err != nil {
		cwlog.DoLogCW("png preview file not found...")
	}

	respData := &discordgo.InteractionResponseData{Files: []*discordgo.File{{Name: path, Reader: to}}}
	resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
	err = s.InteractionRespond(i.Interaction, resp)
	if err != nil {
		cwlog.DoLogCW(err.Error())
	}
}

/* Generate map */
func makeNewMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if fact.FactorioBooted || fact.FactIsRunning {
		buf := "Factorio is currently, running. You must stop the game first. See /stop-factorio"
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}

	t := time.Now()
	ourseed := int(t.UnixNano() - constants.CWEpoch)
	MapPreset := cfg.Local.Settings.MapPreset

	if fact.LastMapSeed > 0 {
		ourseed = fact.LastMapSeed
	}

	//Use seed if specified, then clear it
	if cfg.Local.Settings.Seed > 0 {
		ourseed = cfg.Local.Settings.Seed
		cfg.WriteLCfg()
	}

	if ourseed <= 0 {
		buf := "Invalid seed data. (internal error)"
		cwlog.DoLogCW(buf)
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}

	if strings.EqualFold(MapPreset, "error") {
		buf := "Invalid map preset."
		cwlog.DoLogCW(buf)
		disc.EphemeralResponse(s, i, "Error:", buf)
		return
	}

	disc.EphemeralResponse(s, i, "Status:", "Generating map...")

	/* Generate code to make filename */
	buf := new(bytes.Buffer)

	_ = binary.Write(buf, binary.BigEndian, uint64(ourseed))
	ourcode := fmt.Sprintf("%02d%v", fact.GetMapTypeNum(MapPreset), base64.RawURLEncoding.EncodeToString(buf.Bytes()))
	filename := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves +
		"/gen-" + ourcode + ".zip"

	factargs := []string{"--map-gen-seed", fmt.Sprintf("%v", ourseed), "--create", filename}

	/* Append map gen if set */
	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
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
		buf := fmt.Sprintf("An error occurred attempting to generate the map: %s", aerr)
		cwlog.DoLogCW(buf)
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
		return
	}

	glob.VoteBox.LastMapChange = time.Now()
	fact.VoidAllVotes() /* Void all votes */
	fact.WriteVotes()

	lines := strings.Split(string(out), "\n")

	for _, l := range lines {
		if strings.Contains(l, "Creating new map") {
			buf := fmt.Sprintf("New map saved as: %v", ourcode+".zip")
			var elist []*discordgo.MessageEmbed
			elist = append(elist, &discordgo.MessageEmbed{Title: "Complete:", Description: buf})
			f := discordgo.WebhookParams{Embeds: elist}
			disc.FollowupResponse(s, i, &f)
			return
		}
	}

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: "Unknown error."})
	f := discordgo.WebhookParams{Embeds: elist}
	disc.FollowupResponse(s, i, &f)
}

/* Archive map */
func archiveMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	version := strings.Split(fact.FactorioVersion, ".")
	vlen := len(version)

	if vlen < 3 {
		buf := "Unable to determine Factorio version."
		disc.EphemeralResponse(s, i, "Error:", buf)
	}

	if fact.GameMapPath != "" && fact.FactorioVersion != constants.Unknown {
		shortversion := strings.Join(version[0:2], ".")

		t := time.Now()
		date := t.Format("2006-01-02")

		newmapname := fmt.Sprintf("%v-%v-%v.zip", cfg.Local.Callsign, cfg.Local.Name, date)
		newmappath := fmt.Sprintf("%v%v%v%v%v", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix, "/", newmapname)
		newmapurl := fmt.Sprintf("https://%v%v%v%v%v%v",
			cfg.Global.Paths.URLs.Domain,
			cfg.Global.Paths.URLs.PathPrefix,
			cfg.Global.Paths.URLs.ArchivePath,
			url.PathEscape(shortversion+constants.ArchiveFolderSuffix),
			"/",
			url.PathEscape(newmapname))

		from, erra := os.Open(fact.GameMapPath)
		if erra != nil {
			buf := fmt.Sprintf("An error occurred reading the map to archive: %s", erra)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
		defer from.Close()

		/* Make directory if it does not exist */
		newdir := fmt.Sprintf("%s%s%s/", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix)
		err := os.MkdirAll(newdir, os.ModePerm)
		if err != nil {
			buf := fmt.Sprintf("Unable to create archive directory: %v", err.Error())
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}

		_ = os.Remove(newmappath)
		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			buf := fmt.Sprintf("Unable to write archive file: %v", errb)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
		respData := &discordgo.InteractionResponseData{Content: newmapurl}

		resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
		err = s.InteractionRespond(i.Interaction, resp)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		defer to.Close()

		_, _ = from.Seek(0, io.SeekStart)

		_, errc := io.Copy(to, from)
		if errc != nil {
			buf := fmt.Sprintf("Unable to write map archive file: %s", errc)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
		return

	} else {
		disc.EphemeralResponse(s, i, "Error:", "No map has been loaded yet.")
	}

}

/* Reboots Factorio only */
func startFact(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if fact.FactorioBooted || fact.FactIsRunning {

		buf := "Restarting Factorio..."
		disc.EphemeralResponse(s, i, "Status:", buf)
		fact.QuitFactorio("Server rebooting...")
	} else {
		buf := "Starting Factorio..."
		disc.EphemeralResponse(s, i, "Status:", buf)
	}

	fact.FactAutoStart = true
	glob.RelaunchThrottle = 0
}

/*  StopServer saves the map and closes Factorio.  */
func stopFact(s *discordgo.Session, i *discordgo.InteractionCreate) {
	glob.RelaunchThrottle = 0
	fact.FactAutoStart = false

	if fact.FactorioBooted || fact.FactIsRunning {

		buf := "Stopping Factorio."
		disc.EphemeralResponse(s, i, "Status:", buf)
		fact.QuitFactorio("Server quitting...")
	} else {
		buf := "Factorio isn't running, disabling auto-reboot."
		disc.EphemeralResponse(s, i, "Warning:", buf)
	}

}

/* Update Factorio  */
func updateFact(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split("", " ")
	argnum := len(args)

	if cfg.Global.Paths.Binaries.FactUpdater != "" {
		if argnum > 0 && strings.EqualFold(args[0], "cancel") {
			fact.DoUpdateFactorio = false
			cfg.Local.Options.AutoUpdate = false

			buf := "Update canceled, and auto-update disabled."
			disc.EphemeralResponse(s, i, "Status:", buf)
			return
		}
		fact.CheckFactUpdate(true)
		disc.EphemeralResponse(s, i, "Status:", "Checking for Factorio updates.")
	} else {
		buf := "The Factorio updater isn't configured."
		disc.EphemeralResponse(s, i, "Error:", buf)
	}
}

func updateMods(s *discordgo.Session, i *discordgo.InteractionCreate) {

	disc.EphemeralResponse(s, i, "Status:", "Checking for mod updates.")
	modupdate.CheckMods(true, true)
}
