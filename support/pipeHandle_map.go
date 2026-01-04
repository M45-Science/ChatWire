package support

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/fact"
)

func handleMapLoad(input *handleData) bool {
	/******************
	 * MAP LOAD
	 ******************/
	if strings.HasPrefix(input.noTimecode, "Loading map") {
		cwlog.DoLogCW(input.noTimecode)

		/* Strip file path */
		if input.noTimecodeListLen > 3 {
			fullpath := input.noTimecodeList[2]
			size := input.noTimecodeList[3]
			sizei, _ := strconv.Atoi(size)
			fullpath = strings.ReplaceAll(fullpath, ":", "")

			regaa := regexp.MustCompile(`\/.*?\/saves\/`)
			filename := regaa.ReplaceAllString(fullpath, "")

			fact.GameMapName = filename
			fact.GameMapPath = fullpath
			fact.LastSaveName = filename

			cwlog.DoLogCW("Loading map %s (%v)...", filename, humanize.Bytes(uint64(sizei)))
		} else { /* Just in case */
			cwlog.DoLogCW("Loading map...")
		}
		return true
	}
	return false
}

func handleSaveMsg(input *handleData) bool {
	/*************************
	 * CAPTURE SAVE MESSAGES
	 *************************/
	if strings.HasPrefix(input.noTimecode, "Info AppManager") && strings.Contains(input.noTimecode, "Saving to") {
		if !cfg.Local.Options.HideAutosaves {
			savreg := regexp.MustCompile(`Info AppManager.cpp:\d+: Saving to _(autosave\d+)`)
			savmatch := savreg.FindStringSubmatch(input.noTimecode)
			if len(savmatch) > 1 {
				if !cfg.Local.Options.HideAutosaves {
					buf := fmt.Sprintf("`%v` 💾 %s", fact.Gametime, savmatch[1])
					if fact.NumPlayers > 0 {
						fact.CMS(cfg.Local.Channel.ChatChannel, buf)
					}
					cwlog.DoLogGame(savmatch[1])
				}
				fact.LastSaveName = savmatch[1]
			}
		}
		return true
	}
	return false
}

func handleExitSave(input *handleData) bool {
	/*****************************
	 * CAPTURE MAP NAME, ON EXIT
	 *****************************/
	if strings.HasPrefix(input.noTimecode, "Info MainLoop") && strings.Contains(input.noTimecode, "Saving map as") {
		cwlog.DoLogCW(input.noTimecode)

		/* Strip file path */
		if input.noTimecodeListLen > 5 {
			//Fix odd filenames with spaces???
			fullpath := strings.Join(input.noTimecodeList[5:], " ")
			regaa := regexp.MustCompile(`\/.*?\/saves\/`)
			filename := regaa.ReplaceAllString(fullpath, "")
			filename = strings.ReplaceAll(filename, ":", "")

			/* Increment backup number */
			if cfg.Local.LastSaveBackup < constants.MaxSaveBackups {
				cfg.Local.LastSaveBackup++
			} else {
				cfg.Local.LastSaveBackup = 1
			}

			/* Path for backup save */
			newPath := cfg.GetSavesFolder() + "/"
			/* Name for backup save */
			newName := fmt.Sprintf("bak-%v.zip", cfg.Local.LastSaveBackup)

			/* Document save name for archive command */
			fact.GameMapName = filename
			fact.GameMapPath = fullpath

			/* Log actions */
			cwlog.DoLogCW("Map saved as: %v, backup: %v", filename, newName)
			fact.LastSaveName = filename

			/* Open the quit-save */
			from, erra := os.Open(fullpath)
			if erra != nil {

				buf := fmt.Sprintf("An error occurred when attempting to read the save to backup: %s", erra)
				cwlog.DoLogCW(buf)
				//fact.CMS(cfg.Local.Channel.ChatChannel, buf)
				return true
			}
			defer from.Close()

			/* Create the backup file */
			to, errb := os.OpenFile(newPath+newName, os.O_RDWR|os.O_CREATE, 0666)
			if errb != nil {
				buf := fmt.Sprintf("An error occurred when attempting to create the backup save: %s", errb)
				cwlog.DoLogCW(buf)
				return true
			}
			defer to.Close()

			/* Copy data */
			_, errc := io.Copy(to, from)
			if errc != nil {
				cwlog.DoLogCW("An error occurred when attempting to write the backup save: %s", errc)
				return true
			}

			/* Touch old save, so we won't load the backup file next time */
			currentTime := time.Now().UTC().Local()
			_ = os.Chtimes(fullpath, currentTime, currentTime)
			cfg.WriteLCfg()
		}
		return true
	}
	return false
}
