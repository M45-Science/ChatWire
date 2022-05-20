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
	"ChatWire/sclean"
)

/* Archive map */
func ArchiveMap(s *discordgo.Session, i *discordgo.InteractionCreate) {

	fact.GameMapLock.Lock()
	defer fact.GameMapLock.Unlock()

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
		newmapname := fmt.Sprintf("%v-%v.zip", sclean.AlphaNumOnly(constants.MembersPrefix+cfg.Local.Callsign)+"-"+cfg.Local.Name, date)
		newmappath := fmt.Sprintf("%v%v%v/%v", cfg.Global.Paths.Folders.MapArchives, shortversion, constants.ArchiveFolderSuffix, newmapname)
		newmapurl := fmt.Sprintf("%v%v/%v", cfg.Global.Paths.URLs.ArchiveURL, url.PathEscape(shortversion+constants.ArchiveFolderSuffix), url.PathEscape(newmapname))

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

		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			buf := fmt.Sprintf("Unable to write archive file: %v", errb)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}
		defer to.Close()

		_, errc := io.Copy(to, from)
		if errc != nil {
			buf := fmt.Sprintf("Unable to write map archive file: %s", errc)
			cwlog.DoLogCW(buf)
			disc.EphemeralResponse(s, i, "Error:", buf)
			return
		}

		buf := fmt.Sprintf("Map archived as: %v", newmapurl)
		embed := &discordgo.MessageEmbed{Title: "Complete:", Description: buf}
		disc.InteractionResponse(s, i, embed)
		return

	} else {
		disc.EphemeralResponse(s, i, "Error:", "No map has been loaded yet.")
	}

}
