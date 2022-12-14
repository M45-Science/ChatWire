package moderator

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"io/fs"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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

	numFiles := len(tempf)

	found := 0
	for _, o := range a.Options {
		if o.Type == discordgo.ApplicationCommandOptionString {
			arg := o.StringValue()
			if strings.EqualFold(arg, "clear") {
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
				err := os.Remove(path + "mods-list.json")
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
			}
		}
	}
}
