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
)

var (
	modPackLock sync.Mutex
	lastRun     time.Time
)

func init() {
	lastRun = time.Now().Add(-(constants.ModPackCooldownMin * time.Minute))
}

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

	modPath := cfg.GetModsFolder()
	var modsList []string
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
				modsList = append(modsList, modPath+mod.Filename)
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
	defer func() {
		if err := archive.Close(); err != nil {
			cwlog.DoLogCW("modPack: failed to close archive: %v", err)
		}
	}()

	mitem := cfg.ModPackData{Path: dest, Created: time.Now()}
	cfg.Local.ModPackList = append(cfg.Local.ModPackList, mitem)
	cfg.WriteLCfg()
	zipWriter := zip.NewWriter(archive)

	for _, file := range files {
		fileLower := strings.ToLower(file)
		if strings.HasSuffix(fileLower, ".zip") || strings.HasSuffix(fileLower, ".json") || strings.HasSuffix(fileLower, ".dat") {
			if err := addFileToZip(zipWriter, file); err != nil {
				cwlog.DoLogCW("modPack: failed to add file %v: %v", file, err)
			}
		}
	}
	if err := zipWriter.Close(); err != nil {
		cwlog.DoLogCW("modPack: failed to close zip writer: %v", err)
	}
	return false
}

// addFileToZip adds an individual file to the zip using Store (no compression)
func addFileToZip(zipWriter *zip.Writer, filename string) error {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file info (needed for header)
	info, err := file.Stat()
	if err != nil {
		return err
	}

	// Create the zip header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	// Use relative path in archive
	header.Name = filepath.Base(filename)

	// Set compression method to Store (no compression)
	header.Method = zip.Store

	// Create writer for the file in zip
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	// Copy file contents to zip
	_, err = io.Copy(writer, file)
	return err
}
