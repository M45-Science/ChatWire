package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"archive/zip"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"
)

const (
	MapFolder         = "maps"
	ModFolder         = "mods"
	ModPackFolder     = "modPacks"
	ModSettingsFolder = "modSettings"
)

const (
	TYPE_MAP = iota
	TYPE_MOD
	TYPE_MODPACK
	TYPE_MODSETTINGS

	TYPE_MAX
)

type ftpTypeData struct {
	fType   int
	Name    string
	ID      string
	Command string
	Path    string
}

var FTPTypes [TYPE_MAX]ftpTypeData = [TYPE_MAX]ftpTypeData{
	{fType: TYPE_MAP, Name: "map", ID: "ftp-map", Command: "load-map", Path: MapFolder},
	{fType: TYPE_MOD, Name: "mod", ID: "ftp-mod", Command: "load-mod", Path: ModFolder},
	{fType: TYPE_MODPACK, Name: "modpack", ID: "ftp-modpack", Command: "load-modpack", Path: ModPackFolder},
	{fType: TYPE_MODSETTINGS, Name: "settings", ID: "ftp-settings", Command: "load-settings", Path: ModSettingsFolder},
}

func FTPLoad(s *discordgo.Session, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()

	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			for _, item := range FTPTypes {
				if item.Command == arg.Value {
					ShowFTPList(s, i, item)
					return
				}
			}
		}
	}

	disc.EphemeralResponse(s, i, "Error:", "No valid options were selected.")
}

func LoadFTPFile(file string, fType int) {
	fact.CMS(cfg.Local.Channel.ChatChannel, "Would have loaded: "+file)
}

/* Load a different save-game */
func ShowFTPList(s *discordgo.Session, i *discordgo.InteractionCreate, fType ftpTypeData) {

	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path
	files, err := os.ReadDir(pathPrefix)

	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(s, i, "Error:", "Unable to read the FTP directory.")
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

	var availableFiles []discordgo.SelectMenuOption

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
			info, err := f.Info()
			if err != nil {
				continue
			}
			modDate := time.Since(info.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(2).Format(units) + " ago"

			zip, err := zip.OpenReader(pathPrefix + "/" + fName)
			if err != nil || zip == nil {
				availableFiles = append(availableFiles,
					discordgo.SelectMenuOption{

						Label:       saveName,
						Description: "INVALID ZIP FILE!",
						Value:       "INVALID",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🚫",
						},
					},
				)
			} else {
				availableFiles = append(availableFiles,
					discordgo.SelectMenuOption{

						Label:       saveName,
						Description: modStr,
						Value:       saveName,
						Emoji: &discordgo.ComponentEmoji{
							Name: "🗜️",
						},
					},
				)
			}
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
								CustomID:    fType.ID,
								Placeholder: "Select a " + fType.Name,
								Options:     availableFiles,
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
