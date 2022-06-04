package user

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dustin/go-humanize"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
)

/* executes /online on the server, response handled in chat.go */
func ModPack(s *discordgo.Session, i *discordgo.InteractionCreate) {
	/* Mod path */
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	files, err := ioutil.ReadDir(modPath)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(s, i, "Error", "Error reading mods folder, please inform mods.")
		return
	}

	totalFiles := 0
	modFiles := 0
	var fbytes int64
	var modsList []string = []string{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			modsList = append(modsList, f.Name())
			modFiles++
			totalFiles++

			fbytes += f.Size()
		} else if strings.EqualFold(f.Name(), "mod-list.json") {
			modsList = append(modsList, f.Name())
			totalFiles++

			fbytes += f.Size() / 1024
		} else if strings.EqualFold(f.Name(), "mod-settings.dat") {
			modsList = append(modsList, f.Name())
			totalFiles++

			fbytes += f.Size() / 1024
		}
	}

	if modFiles > 0 {
		msg := fmt.Sprintf("%d mods found, %v total.\nGenerating modpack zip, please wait.", modFiles, humanize.Bytes(uint64(fbytes)))
		disc.EphemeralResponse(s, i, "Mods", msg)

		makeModPack(s, i, modsList)
	} else {

		disc.EphemeralResponse(s, i, "Error:", "No mods are installed.")
	}
}

func makeModPack(s *discordgo.Session, i *discordgo.InteractionCreate, modsList []string) {
	modPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"

	err := makeZipFromFileList(modsList, modPath)
	if err {
		buf := fmt.Sprintf("Could not read the files, please inform moderators.")
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
		return
	} else {
		buf := fmt.Sprintf("Modpack created, now available at https://%v/%v/%v/%v", cfg.Global.Paths.URLs.Domain)
		var elist []*discordgo.MessageEmbed
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Error:", Description: buf})
		f := discordgo.WebhookParams{Embeds: elist}
		disc.FollowupResponse(s, i, &f)
	}

}

func makeZipFromFileList(files []string, dest string) bool {
	fmt.Println("creating zip archive...")
	archive, err := os.Create(dest)
	if err != nil {
		panic(err)
	}
	defer archive.Close()
	zipWriter := zip.NewWriter(archive)

	for _, file := range files {
		fmt.Println("opening first file...")
		f1, err := os.Open(file)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		defer f1.Close()

		fmt.Println("writing first file to archive...")
		w1, err := zipWriter.Create("csv/test.csv")
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		if _, err := io.Copy(w1, f1); err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}
	fmt.Println("closing zip archive...")
	zipWriter.Close()
}
