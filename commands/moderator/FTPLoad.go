package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"archive/zip"
	"encoding/json"
	"fmt"
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
	ModListFolder     = "modLists"
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
	fType    int
	Name     string
	Value    string
	Path     string
	Function func(ftype ftpTypeData, i *discordgo.InteractionCreate)
}

type modInfoData struct {
	Name, Version, Author, Factorio_version string
}

var FTPTypes [TYPE_MAX]ftpTypeData = [TYPE_MAX]ftpTypeData{
	{fType: TYPE_MAP, Name: "map", Value: "load-map", Path: MapFolder, Function: ListZips},
	{fType: TYPE_MOD, Name: "mod", Value: "load-mod", Path: ModFolder, Function: ListZips},
	{fType: TYPE_MODPACK, Name: "modpack", Value: "load-modpack", Path: ModPackFolder, Function: ListZips},
	{fType: TYPE_MODLIST, Name: "modlist", Value: "load-modlist", Path: ModListFolder, Function: ListModlists},
	{fType: TYPE_MODSETTINGS, Name: "settings", Value: "load-settings", Path: ModSettingsFolder, Function: ListModlists},
}

func HandleFTP(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	for _, arg := range a.Options {
		if arg.Type != discordgo.ApplicationCommandOptionString {
			continue
		}
		for _, ftype := range FTPTypes {
			if strings.EqualFold(ftype.Value, arg.StringValue()) {
				ftype.Function(ftype, i)
				return
			}
		}
	}

	disc.InteractionEphemeralResponse(i, "Error", "Sorry, you didn't supply any valid options.")

}

func MakeFTPFolders() {
	pathPrefix := cfg.Global.Paths.Folders.FTP

	var err error
	for _, item := range FTPTypes {
		dirPath := pathPrefix + item.Path
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			cwlog.DoLogCW("Unable to create FTP dir: " + dirPath)
		}
	}
}

func checkModSettings(path string) error {

	if !strings.HasSuffix(path, ".dat") {
		return fmt.Errorf("the mod settings file does not end with .dat")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	dlen := len(data)
	if dlen < 10 {
		return fmt.Errorf("the mod settings file is too small")
	} else if dlen > (1024 * 1024) { //1MB
		return fmt.Errorf("the mod settings file is too large")
	}

	return nil
}

func LoadFTPFile(i *discordgo.InteractionCreate, file string, fType ftpTypeData) {

	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path + "/"
	zipPath := pathPrefix + file + ".zip"

	if fType.fType == TYPE_MAP {
		if fact.HasZipBomb(zipPath) {
			fact.ReportZipBomb(i, zipPath)
			return
		}
		pass, _ := fact.CheckSave(pathPrefix, file+".zip", false)

		if pass {
			disc.InteractionEphemeralResponse(i, "Status", "Map appears to be valid!")
		} else {
			disc.InteractionEphemeralResponse(i, "Status", "Map appears to be invalid!")
		}
	} else if fType.fType == TYPE_MODSETTINGS { //Mod settings here
		err := checkModSettings(pathPrefix + file)
		if err != nil {
			cwlog.DoLogCW("checkModSettings: Error: " + err.Error())
			disc.InteractionEphemeralResponse(i, "Error", "The mod settings file appears to be invalid: "+err.Error())
			return
		}
		disc.InteractionEphemeralResponse(i, "Status", "Would load mod-settings here.")

	} else { //mod or modpack
		if fact.HasZipBomb(zipPath) {
			fact.ReportZipBomb(i, zipPath)
			return
		}

		zip, err := zip.OpenReader(zipPath)
		if err != nil || zip == nil {
			disc.InteractionEphemeralResponse(i, "Error", "The zip file is invalid!")
			return
		}
		defer zip.Close()

		if fType.fType == TYPE_MODPACK {
			for _, file := range zip.File {
				if !strings.HasSuffix(file.Name, ".zip") {
					if strings.EqualFold(file.Name, "mod-list.json") {
						//check mod-list
					} else {
						return
					}
				}
			}
			disc.InteractionEphemeralResponse(i, "Error", "Would load modpack here.")

		} else if fType.fType == TYPE_MOD {
			for _, file := range zip.File {
				if path.Base(file.Name) == "info.json" {
					fc, err := file.Open()
					if err != nil {
						disc.InteractionEphemeralResponse(i, "Error", "The mod data could not be opened.")
						return
					}
					defer fc.Close()

					content, err := io.ReadAll(fc)
					if err != nil {
						disc.InteractionEphemeralResponse(i, "Error", "The mod data could not be read.")
						return
					}

					jsonData := modInfoData{}
					err = json.Unmarshal(content, &jsonData)
					if err != nil {
						disc.InteractionEphemeralResponse(i, "Error", "The mod info could not be parsed.")
						return
					}

					if len(jsonData.Author) < 2 || len(jsonData.Factorio_version) < 3 ||
						len(jsonData.Name) < 3 || len(jsonData.Version) < 3 {
						disc.InteractionEphemeralResponse(i, "Error", "The mod data contains invalid data.")
						return
					}

					disc.InteractionEphemeralResponse(i, "Error", "Would load the mod here.")
					return
				}
			}
			disc.InteractionEphemeralResponse(i, "Error", "The mod appears to be invalid or corrupted.")
		}
	}

}

func ListModlists(fType ftpTypeData, i *discordgo.InteractionCreate) {
	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path + "/"
	dir, err := os.ReadDir(pathPrefix)
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read the mod settings directory.")
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
								CustomID:    fType.Value,
								Placeholder: "Select a " + fType.Name + " file.",
								Options:     availableFiles,
							},
						},
					},
				},
			},
		}
		err := disc.DS.InteractionRespond(i.Interaction, response)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	} else {
		disc.InteractionEphemeralResponse(i, "Error:", "No settings files were found.")
	}
}

func ListZips(fType ftpTypeData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}
	pathPrefix := cfg.Global.Paths.Folders.FTP + fType.Path + "/"
	files, err := os.ReadDir(pathPrefix)

	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read the FTP directory.")
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

		return iInfo.ModTime().UnixNano() >= jInfo.ModTime().UnixNano()
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
		disc.InteractionEphemeralResponse(i, "Error:", "No files of that type were found.")
	} else {

		response := &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Choose a " + fType.Name + " to load from the FTP:",
				Flags:   1 << 6,
				Components: []discordgo.MessageComponent{
					discordgo.ActionsRow{
						Components: []discordgo.MessageComponent{
							discordgo.SelectMenu{
								// Select menu, as other components, must have a customID, so we set it to this value.
								CustomID:    fType.Value,
								Placeholder: "Select a " + fType.Name,
								Options:     availableFiles,
							},
						},
					},
				},
			},
		}
		err := disc.DS.InteractionRespond(i.Interaction, response)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}
}
