package admin

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/support"
)

/* Generate map */
func NewMap(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	if fact.FactorioBooted || fact.FactIsRunning {
		buf := "Factorio is currently running. You must stop factorio first."
		disc.EphemeralResponse(i, "Error:", buf)
		return
	}

	disc.EphemeralResponse(i, "Status:", "Generating new map.")
	fileName := fact.GenNewMap()
	disc.FollowupResponse(i, &discordgo.WebhookParams{Content: "Map generator: " + fileName})
}

/* Archive map */
func ArchiveMap(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	version := strings.Split(fact.FactorioVersion, ".")
	vlen := len(version)

	if vlen < 3 {
		buf := "Unable to determine Factorio version."
		disc.EphemeralResponse(i, "Error:", buf)
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
			disc.EphemeralResponse(i, "Error:", buf)
			return
		}
		defer from.Close()

		/* Make directory if it does not exist */
		newdir := fmt.Sprintf("%s%s%s/", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix)
		err := os.MkdirAll(newdir, os.ModePerm)
		if err != nil {
			buf := fmt.Sprintf("Unable to create archive directory: %v", err.Error())
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(i, "Error:", buf)
			return
		}

		_ = os.Remove(newmappath)
		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			buf := fmt.Sprintf("Unable to write archive file: %v", errb)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(i, "Error:", buf)
			return
		}
		respData := &discordgo.InteractionResponseData{Content: newmapurl}

		resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
		err = disc.DS.InteractionRespond(i.Interaction, resp)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		defer to.Close()

		_, _ = from.Seek(0, io.SeekStart)

		_, errc := io.Copy(to, from)
		if errc != nil {
			buf := fmt.Sprintf("Unable to write map archive file: %s", errc)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(i, "Error:", buf)
			return
		}
		return

	} else {
		disc.EphemeralResponse(i, "Error:", "No map has been loaded yet.")
	}

}

/* Reboots Factorio only */
func StartFact(cmd *glob.CommandData, i *discordgo.InteractionCreate) {

	if !support.WithinHours() {
		buf := fmt.Sprintf("Will not start Factorio. Current time allowed is: %v - %v GMT.",
			cfg.Local.Options.PlayStartHour, cfg.Local.Options.PlayEndHour)
		disc.EphemeralResponse(i, "Status:", buf)
	} else if fact.FactorioBooted || fact.FactIsRunning {
		buf := "Restarting Factorio..."
		disc.EphemeralResponse(i, "Status:", buf)
		fact.QuitFactorio("Server rebooting...")
	} else {
		buf := "Starting Factorio..."
		disc.EphemeralResponse(i, "Status:", buf)
	}

	fact.FactAutoStart = true
	glob.RelaunchThrottle = 0
}

/*  StopServer saves the map and closes Factorio.  */
func StopFact(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	glob.RelaunchThrottle = 0
	fact.FactAutoStart = false

	if fact.FactorioBooted || fact.FactIsRunning {

		buf := "Stopping Factorio."
		disc.EphemeralResponse(i, "Status:", buf)
		fact.QuitFactorio("Server quitting...")
	} else {
		buf := "Factorio isn't running, disabling auto-reboot."
		disc.EphemeralResponse(i, "Warning:", buf)
	}

}

func UpdateMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(i, "Status:", "Checking for mod updates.")
	modupdate.CheckMods(true, true)
}

func UpdateFactorio(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(i, "Status:", "Checking for factorio updates.")

	_, msg, err, _ := factUpdater.DoQuickLatest(false)
	if err {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "ERROR:", Description: "Factorio update failed:  " + msg})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(i, &f)
	} else {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Info:", Description: msg})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(i, &f)
	}
}

func InstallFactorio(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(i, "Info", "Installing Factorio...")
	_, msg, _, _ := factUpdater.DoQuickLatest(true)

	var elist []*discordgo.MessageEmbed
	elist = append(elist, &discordgo.MessageEmbed{Title: "Info:", Description: msg})
	f := discordgo.WebhookParams{Embeds: elist}
	disc.FollowupResponse(i, &f)
}
