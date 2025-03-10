package user

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/util"
)

var (
	modPackLock sync.Mutex
	lastRun     time.Time
)

/* executes /online on the server, response handled in chat.go */
func ModPack(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	modPackLock.Lock()
	defer modPackLock.Unlock()

	if !lastRun.IsZero() && time.Since(lastRun) < constants.ModPackCooldownMin*time.Minute {
		disc.InteractionEphemeralResponse(i, "Error", "A modpack was already created recently, please wait a bit.")
		return
	}
	lastRun = time.Now()

	if len(cfg.Local.ModPackList) >= constants.MaxModPacks {
		disc.InteractionEphemeralResponse(i, "Error", "Too many existing modpack files already!\nTry again later.")
		return
	}

	mfile, err := modupdate.GetModFiles()
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read mod files.")
		return
	}
	jfile, err := modupdate.GetModList()
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read mod list.")
		return
	}

	modPath := util.GetModsFolder()
	var modsList []string = []string{}
	modFiles := 0
	for _, item := range jfile.Mods {
		if modupdate.IsBaseMod(item.Name) {
			continue
		}
		if !item.Enabled {
			continue
		}
		for _, mod := range mfile {
			if strings.EqualFold(mod.Name, item.Name) {
				modsList = append(modsList, modPath+mod.OldFilename)
				modFiles++
			}
		}
	}

	if modFiles > 0 {
		msg := fmt.Sprintf("%d enabled mods found.\nGenerating modpack zip, please wait.", modFiles)
		disc.InteractionEphemeralResponse(i, "Modpack", msg)

		makeModPack(i, modsList)
	} else {

		disc.InteractionEphemeralResponse(i, "Error:", "No mods are currently installed.")
	}
}

func makeModPack(i *discordgo.InteractionCreate, modsList []string) {
	packName := fmt.Sprintf("%v-%v-%v.zip",
		cfg.Local.Callsign,
		cfg.Local.Name,
		(time.Now().UTC().UnixNano()-constants.CWEpoch)/1000000000/60)

	err := makeZipFromFileList(modsList, cfg.Global.Paths.Folders.ModPack+packName)
	if err {
		buf := "Could not read/write the files, please inform moderators."
		disc.InteractionEphemeralResponse(i, "Error", buf)
		return
	} else {

		name := constants.Unknown
		if i.Member != nil {
			name = i.Member.User.Username
		}
		xTime := time.Now().UTC()
		xTime = xTime.Add(time.Duration(time.Minute * constants.ModPackLifeMins))
		buf := fmt.Sprintf("Modpack requested by %v, now available at https://%v%v%v%v\n\nFile will be deleted at: <t:%v:F>(LOCAL)\n",
			name,
			cfg.Global.Paths.URLs.Domain,
			cfg.Global.Paths.URLs.PathPrefix,
			cfg.Global.Paths.URLs.ModPackPath,
			packName,
			xTime.Unix(),
		)

		disc.InteractionEphemeralResponse(i, "Success", buf)
	}

}

func makeZipFromFileList(files []string, dest string) bool {

	dir := path.Dir(dest)
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		cwlog.DoLogCW("Unable to make modpack dir.")
	}

	archive, err := os.Create(dest)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return true
	}
	defer archive.Close()

	mitem := cfg.ModPackData{Path: dest, Created: time.Now()}
	cfg.Local.ModPackList = append(cfg.Local.ModPackList, mitem)
	cfg.WriteLCfg()

	info, err := archive.Stat()
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return true
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return true
	}
	header.Method = zip.Store
	zipWriter := zip.NewWriter(archive)

	for _, file := range files {

		f1, err := os.Open(file)
		if err != nil {
			cwlog.DoLogCW(err.Error())
			return true
		}

		w1, err := zipWriter.Create(filepath.Base(file))
		if err != nil {
			cwlog.DoLogCW(err.Error())
			f1.Close()
			return true
		}

		if _, err := io.Copy(w1, f1); err != nil {
			cwlog.DoLogCW(err.Error())
			f1.Close()
			return true
		}
		f1.Close()
	}
	zipWriter.Close()
	return false
}
