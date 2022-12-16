package moderator

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"archive/zip"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type infoJSONData struct {
	Name             string
	Version          string
	Title            string
	Dependencies     []string
	Description      string
	Factorio_version string
	Homepage         string
}

type modData struct {
	Name    string
	Enabled bool
}

type modListData struct {
	Mods []modData
}

func readInfoJson(content []byte) (*infoJSONData, bool) {

	info := &infoJSONData{}
	err := json.Unmarshal(content, &info)
	if err != nil {
		cwlog.DoLogCW("readInfoJson: Unmarshal failure")
		return nil, false
	}
	return info, true
}

func readMod(filename string) (*infoJSONData, bool) {
	read, err := zip.OpenReader(filename)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return nil, false
	}

	for _, file := range read.File {
		fc, err := file.Open()
		if err != nil {
			cwlog.DoLogCW(err.Error())
			return nil, false
		}
		fileName := filepath.Base(file.Name)
		if fileName == "info.json" {
			content, err := ioutil.ReadAll(fc)
			if err != nil {
				cwlog.DoLogCW(err.Error())
				return nil, false
			}
			return readInfoJson(content)
		}
	}

	return nil, false
}

func readModList(path string) (*modListData, bool) {

	content, err := os.ReadFile(path + "mod-list.json")
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return nil, false
	}

	modList := &modListData{}
	err = json.Unmarshal(content, &modList)
	if err != nil {
		cwlog.DoLogCW("readModList: Unmarshal failure")
		return nil, false
	}

	return modList, true
}

func searchModlist(modList *modListData, modName string) (bool, bool) {
	if modList != nil && modList.Mods != nil && modName != "" {
		for _, mod := range modList.Mods {
			if mod.Name == modName {
				return true, true
			}
		}
		return false, true
	}

	return false, false
}

func ModManager(s *discordgo.Session, i *discordgo.InteractionCreate) {
	a := i.ApplicationCommandData()

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/"

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(s, i, "Error:", "Unable to read mods directory.")
	}

	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files, and our map-change temp file.
		if strings.HasSuffix(f.Name(), ".zip") {
			tempf = append(tempf, f)
		}
	}

	modList, _ := readModList(path)

	numFiles := len(tempf)

	found := 0
	options := 0
	for _, o := range a.Options {
		options++
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()
			if strings.EqualFold(arg, "clear-all") {
				for x := 0; x < numFiles; x++ {
					f := tempf[x]
					fName := f.Name()

					if strings.HasSuffix(fName, ".zip") {
						err := os.Remove(path + fName)
						if err != nil {
							disc.EphemeralResponse(s, i, "Error:", "Unable to delete mod files.")
							return
						}
						found++
					}
				}
				err := os.Remove(path + "mod-list.json")
				if err != nil {
					disc.EphemeralResponse(s, i, "Error:", "Unable to delete mod files.")
					return
				}
				if found == 0 {
					disc.EphemeralResponse(s, i, "Info:", "No mods to delete.")
					return
				}
				disc.EphemeralResponse(s, i, "Info:", "All mods deleted.")
				return
			} else if strings.EqualFold(arg, "list-all") {
				options = -1
			}
		}
	}
	if options == 0 || options == -1 {
		buf := ""
		count := 1
		for x := 0; x < numFiles; x++ {
			file := tempf[x]

			number := fmt.Sprintf("#%3v ", count)
			info, found := readMod(path + file.Name())
			if found {
				buf = buf + number
				foundMod, foundList := searchModlist(modList, info.Name)
				if foundList {
					if foundMod {
						buf = buf + "( ON) "
					} else {
						buf = buf + "(OFF) "
					}
				} else {
					buf = buf + "( ? ) "
				}
				buf = buf + info.Name + " v" + info.Version
			} else {
				buf = buf + number + "Filename: " + strings.TrimSuffix(file.Name(), ".zip") + " (corrupt mod file?)"
			}
			buf = buf + "\n"
			count++
		}
		if count == 0 {
			buf = buf + "None."
		}
		disc.EphemeralResponse(s, i, "Mods available:", "```\n"+buf+"\n```")
	} else {
		disc.EphemeralResponse(s, i, "ERROR:", "Not implemented.")
	}
}
