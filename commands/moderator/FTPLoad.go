package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"
)

const (
	MapFolder     = "maps"
	ModFolder     = "mods"
	ModPackFolder = "modPacks"
)

func FTPLoad(s *discordgo.Session, i *discordgo.InteractionCreate) {
	path := cfg.Global.Paths.Folders.FTP + MapFolder

	ShowFTPList(s, i, path)
}

/* Load a different save-game */
func ShowFTPList(s *discordgo.Session, i *discordgo.InteractionCreate, path string) {

	files, err := os.ReadDir(path)

	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(s, i, "Error:", "Unable to read FTP map directory.")
		return
	}

	var tempf []fs.DirEntry
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		iInfo, _ := tempf[i].Info()
		jInfo, _ := tempf[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	var availableMaps []discordgo.SelectMenuOption

	numFiles := len(tempf)
	//Limit results
	if numFiles > constants.MaxMapResults-1 {
		numFiles = constants.MaxMapResults - 1
	}

	for i := 0; i < numFiles; i++ {

		f := tempf[i]
		fName := f.Name()

		if strings.HasSuffix(fName, ".zip") {
			saveName := strings.TrimSuffix(fName, ".zip")

			units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
			if err != nil {
				panic(err)
			}

			/* Get mod date */
			info, _ := f.Info()
			modDate := time.Since(info.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(2).Format(units) + " ago"

			availableMaps = append(availableMaps,
				discordgo.SelectMenuOption{

					Label:       saveName,
					Description: modStr,
					Value:       path + "/" + saveName,
					Emoji: &discordgo.ComponentEmoji{
						Name: "🗜️",
					},
				},
			)
		}
	}

	if numFiles <= 0 {
		disc.EphemeralResponse(s, i, "Error:", "No maps were found.")
	} else {

		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Choose a map to load from the FTP:",
				Flags:   1 << 6,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								// Select menu, as other components, must have a customID, so we set it to this value.
								CustomID:    "FTPFile",
								Placeholder: "Select one",
								Options:     availableMaps,
							},
						},
					},
				},
			},
		}
		err := s.InteractionRespond(i.Interaction, response)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}
}
