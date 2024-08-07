package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"archive/zip"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path"
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
	ModListsFolder    = "modLists"
	ModSettingsFolder = "modSettings"
)

const (
	TYPE_MAP = iota
	TYPE_MOD
	TYPE_MODPACK
	TYPE_MODLIST
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

type modInfoData struct {
	Name, Version, Author, Factorio_version string
}

var FTPTypes [TYPE_MAX]ftpTypeData = [TYPE_MAX]ftpTypeData{
	{fType: TYPE_MAP, Name: "map", ID: "ftp-map", Command: "load-map", Path: MapFolder},
	{fType: TYPE_MOD, Name: "mod", ID: "ftp-mod", Command: "load-mod", Path: ModFolder},
	{fType: TYPE_MODPACK, Name: "modpack", ID: "ftp-modpack", Command: "load-modpack", Path: ModPackFolder},
	{fType: TYPE_MODLIST, Name: "modlist", ID: "ftp-modlist", Command: "load-modlist", Path: ModListsFolder},
	{fType: TYPE_MODSETTINGS, Name: "settings", ID: "ftp-settings", Command: "load-settings", Path: ModSettingsFolder},
}

func MakeFTPFolders() {
	pathPrefix := cfg.Global.Paths.Folders.FTP + "/"

	var err error
	for _, item := range FTPTypes {
		dirPath := pathPrefix + item.Path
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			cwlog.DoLogCW("Unable to create FTP dir: " + dirPath)
		}
	}
}

func FTPLoad(s *discordgo.Session, i *discordgo.InteractionCreate) {

	a := i.ApplicationCommandData()

	for _, arg := range a.Options {
		if arg.Type == discordgo.ApplicationCommandOptionString {
			for _, item := range FTPTypes {
				if item.Command == arg.Value {
					if item.fType == TYPE_MODSETTINGS {
						ShowDatList(s, i, item)
					} else {
						ShowZipList(s, i, item)
					}
					return
				}
			}
		}
	}

	disc.EphemeralResponse(s, i, "Error:", "No valid options were selected.")
}

func modCheckError(s *discordgo.Session, i *discordgo.InteractionCreate) {
	disc.EphemeralResponse(s, i, "Error:", "The mod appears to be invalid or corrupt.")
}

func LoadFTPFile(s *discordgo.Session, i *discordgo.InteractionCreate, file string, fType int) {

	pathPrefix := cfg.Global.Paths.Folders.FTP + FTPTypes[fType].Path + "/"
	zipPath := pathPrefix + file + ".zip"

	if fType == TYPE_MAP {
		if fact.HasZipBomb(zipPath) {
			fact.ReportZipBomb(s, i, zipPath)
			return
		}
		pass, _ := fact.CheckSave(pathPrefix, file+".zip", false)

		if pass {
			disc.EphemeralResponse(s, i, "Debug:", "Map appears to be valid.")
		} else {
			disc.EphemeralResponse(s, i, "Error:", "Map appears to be invalid!")
		}
	} else if fType == TYPE_MODSETTINGS {

	} else {
		if fact.HasZipBomb(zipPath) {
			fact.ReportZipBomb(s, i, zipPath)
			return
		}

		zip, err := zip.OpenReader(zipPath)
		if err != nil || zip == nil {
			disc.EphemeralResponse(s, i, "Error:", "Unable to read the zip file!")
			return
		}
		defer zip.Close()

		if fType == TYPE_MODPACK {
			for _, file := range zip.File {
				if !strings.HasSuffix(file.Name, ".zip") {
					if strings.EqualFold(file.Name, "mod-list.json") {
						//check mod-list
					} else {
						disc.EphemeralResponse(s, i, "Error:", "The modpack contains unknown files, aborting.")
						return
					}
				} else {
					if file.UncompressedSize64 > fact.MaxZipSize {
						fact.ReportZipBomb(s, i, zipPath)
						return
					}
				}
			}
			disc.EphemeralResponse(s, i, "Error:", "Doing the stuff.")
		} else if fType == TYPE_MOD {
			for _, file := range zip.File {
				if path.Base(file.Name) == "info.json" {
					fc, err := file.Open()
					if err != nil {
						disc.EphemeralResponse(s, i, "Error:", "The mod info file could not be opened.")
						return
					}
					defer fc.Close()

					content, err := io.ReadAll(fc)
					if err != nil {
						disc.EphemeralResponse(s, i, "Error:", "The mod info file could not be read.")
						return
					}

					jsonData := modInfoData{}
					err = json.Unmarshal(content, &jsonData)
					if err != nil {
						disc.EphemeralResponse(s, i, "Error:", "The mod info could not be decoded.")
						return
					}

					if len(jsonData.Author) < 2 || len(jsonData.Factorio_version) < 3 ||
						len(jsonData.Name) < 3 || len(jsonData.Version) < 3 {
						disc.EphemeralResponse(s, i, "Error:", "The mod info contains invalid values.")
						return
					}

					disc.EphemeralResponse(s, i, "Status:", "Doing the thing.")
					return
				}
			}
			modCheckError(s, i)
		}
	}

}

func ShowDatList(s *discordgo.Session, i *discordgo.InteractionCreate, fType ftpTypeData) {
	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path + "/"
	dir, err := os.ReadDir(pathPrefix)
	if err != nil {
		disc.EphemeralResponse(s, i, "Error:", "Unable to read the mod settings directory.")
		return
	}

	var availableFiles []discordgo.SelectMenuOption

	found := false
	units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
	if err != nil {
		panic(err)
	}

	sort.Slice(dir, func(i, j int) bool {
		iInfo, _ := dir[i].Info()
		jInfo, _ := dir[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
	})

	for _, file := range dir {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".dat") {
			found = true

			/* Get mod date */
			info, err := file.Info()
			if err != nil {
				continue
			}
			modDate := time.Since(info.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(2).Format(units) + " ago"

			if info.Size() < 15 {
				availableFiles = append(availableFiles,
					discordgo.SelectMenuOption{

						Label:       file.Name(),
						Description: "INVALID SETTINGS FILE!",
						Value:       "INVALID",
						Emoji: &discordgo.ComponentEmoji{
							Name: "🚫",
						},
					},
				)
			} else {
				availableFiles = append(availableFiles,
					discordgo.SelectMenuOption{

						Label:       file.Name(),
						Description: modStr,
						Value:       file.Name(),
						Emoji: &discordgo.ComponentEmoji{
							Name: "📄",
						},
					},
				)
			}
		}
	}
	if found {
		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Choose a settings file to load from the FTP:",
				Flags:   1 << 6,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								// Select menu, as other components, must have a customID, so we set it to this value.
								CustomID:    fType.ID,
								Placeholder: "Select a " + fType.Name + " file.",
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
	} else {
		disc.EphemeralResponse(s, i, "Error:", "No settings files were found.")
	}
}

/* Load a different save-game */
func ShowZipList(s *discordgo.Session, i *discordgo.InteractionCreate, fType ftpTypeData) {

	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path + "/"
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

	units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
	if err != nil {
		panic(err)
	}

	for i := 0; i < numFiles; i++ {

		f := tempf[i]
		fName := f.Name()

		if strings.HasSuffix(fName, ".zip") {
			saveName := strings.TrimSuffix(fName, ".zip")

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
		disc.EphemeralResponse(s, i, "Error:", "No files of that type were found.")
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
