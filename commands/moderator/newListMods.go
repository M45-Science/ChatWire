package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"
	"archive/zip"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"
)

func ListGameMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	pathPrefix := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/"

	files, err := os.ReadDir(pathPrefix)

	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(i, "Error:", "Unable to read the FTP directory.")
		return
	}

	var tempf []fs.DirEntry
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		return strings.ToLower(tempf[i].Name()) < strings.ToLower(tempf[j].Name())
	})

	numFiles := len(tempf)
	//Limit results
	if numFiles > constants.MaxMapResults-1 {
		numFiles = constants.MaxMapResults - 1
	}

	units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
	if err != nil {
		panic(err)
	}

	buf := ""
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
			if err == nil && zip != nil {
				buf = buf + fmt.Sprintf("%-32v (%v)\n",
					sclean.TruncateStringEllipsis(saveName, 32),
					modStr)
			}
		}
	}

	if buf == "" {
		disc.EphemeralResponse(i, "Info", "No mods were found!")
	} else {
		disc.EphemeralResponse(i, "Mod list:", buf)
	}
}
