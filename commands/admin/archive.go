package admin

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"../../config"
	"../../constants"
	"../../fact"
	"../../glob"
	"../../logs"
	"github.com/bwmarrin/discordgo"
)

//Archive map
func ArchiveMap(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	glob.GameMapLock.Lock()
	defer glob.GameMapLock.Unlock()

	version := strings.Split(glob.FactorioVersion, ".")
	vlen := len(version)

	if vlen < 3 {
		logs.Log("Unable to determine factorio version.")
		return
	}

	if glob.GameMapPath != "" && glob.FactorioVersion != constants.Unknown {
		shortversion := strings.Join(version[0:2], ".")

		t := time.Now()
		date := fmt.Sprintf("%02d-%02d-%04d_%02d-%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute())
		newmapname := fmt.Sprintf("%s-%s.zip", config.Config.ChannelName, date)
		newmappath := fmt.Sprintf("%s/%s maps/%s", config.Config.MapArchivePath, shortversion, newmapname)

		from, erra := os.Open(glob.GameMapPath)
		if erra != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to the map to archive. Details: %s", erra))
		}
		defer from.Close()

		to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
		if errb != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to create the archive map file. Details: %s", errb))
		}
		defer to.Close()

		_, errc := io.Copy(to, from)
		if errc != nil {
			logs.Log(fmt.Sprintf("An error occurred when attempting to write the archived map. Details: %s", errc))
		}

		var buf string
		if erra == nil && errb == nil && errc == nil {
			buf = fmt.Sprintf("Map archived as: %s", newmappath)
		} else {
			buf = "Map archive failed."
		}

		fact.CMS(m.ChannelID, buf)
	} else {
		fact.CMS(m.ChannelID, "No map has been loaded yet.")
	}

}
