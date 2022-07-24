package support

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"

	"github.com/dustin/go-humanize"
)

func GetSaveGame(doInject bool) (foundGood bool, fileName string, fileDir string) {
	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	files, err := ioutil.ReadDir(path)

	/* We can't read saves dir */
	if err != nil {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "Unable to read saves folder, stopping.")
		return false, "", ""
	}

	/* Loop all files */
	var tempf []fs.FileInfo
	for _, f := range files {
		//Hide non-zip files and temp files
		if strings.HasSuffix(f.Name(), ".zip") && !strings.HasSuffix(f.Name(), "tmp.zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	numSaves := len(tempf)
	if numSaves <= 0 {
		fact.LogCMS(cfg.Local.Channel.ChatChannel, "No saves found, stopping.")
		return false, "", ""
	}

	for pos := 0; pos < numSaves; pos++ {
		name := tempf[pos].Name()

		if name == "" {
			continue
		}

		zip, err := zip.OpenReader(path + "/" + name)
		if err != nil || zip == nil {
			buf := fmt.Sprintf("Save '%v' is not a valid zip file: '%v', trying next save.", name, err.Error())
			if pos == 0 {
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
			}
			cwlog.DoLogCW(buf)
		} else {

			for _, file := range zip.File {
				_, err := file.Open()

				if err != nil {
					buf := fmt.Sprintf("Save '%v' contains corrupt data: '%v', trying next save.", name, err.Error())
					if pos == 0 {
						fact.CMS(cfg.Local.Channel.ChatChannel, buf)
					}
					cwlog.DoLogCW(buf)
					break
				} else {
					if strings.HasSuffix(file.Name, "level.dat0") {
						//Save appears valid
						cwlog.DoLogCW("Found " + file.Name + ", loading.")
						return true, path + "/" + name, filepath.Dir(file.Name)
					}
				}
			}
			buf := fmt.Sprintf("Save '%v' did not contain a level.dat0 file.", name)
			if pos == 0 {
				fact.CMS(cfg.Local.Channel.ChatChannel, buf)
			}
			cwlog.DoLogCW(buf)
		}
	}

	return false, "", ""
}

type zipFilesData struct {
	Name string
	Data []byte
}

var zipFiles []zipFilesData

func launchFactorio() {

	/* Clear this so we know if the the loaded map has our soft mod or not */
	glob.SoftModVersion = constants.Unknown
	glob.OnlineCommand = constants.OnlineCommand
	fact.OnlinePlayersLock.Lock()
	glob.OnlinePlayers = []glob.OnlinePlayerData{}
	fact.OnlinePlayersLock.Unlock()

	/* Check for factorio install */
	checkFactPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir

	if _, err := os.Stat(checkFactPath); os.IsNotExist(err) {
		fact.CMS(cfg.Local.Channel.ChatChannel, "Factorio does not appear to be installed. Use /factorio install-factorio to install it.")
		cwlog.DoLogCW("Factorio does not appear to be installed at the configured path: " + checkFactPath)
		fact.FactAutoStart = false
		return
	}

	/* Insert soft mod */
	/* OLD SCRIPT VERSION
	if cfg.Global.Paths.Binaries.SoftModInserter != "" {
		command := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.Binaries.SoftModInserter
		out, errs := exec.Command(command, cfg.Local.Callsign).Output()
		if errs != nil {
			cwlog.DoLogCW(fmt.Sprintf("Unable to run soft-mod insert script. Details:\nout: %v\nerr: %v", string(out), errs))
		}
	} */

	/* Find, test and load newest save game available */
	found, fileName, folderName := GetSaveGame(true)
	if !found {
		cwlog.DoLogCW("Unable to load any saves.")
		fact.FactAutoStart = false
		return
	}

	/* Unzip save game */
	archive, errz := zip.OpenReader(fileName)
	if errz != nil {
		cwlog.DoLogCW("sm-inject: unable to open save game.")
	} else {
		for _, f := range archive.File {
			if strings.HasPrefix(f.Name, folderName+"/") &&
				(strings.HasPrefix(f.Name, folderName+"/level.dat") ||
					strings.HasSuffix(f.Name, ".json") ||
					strings.HasSuffix(f.Name, ".dat") ||
					strings.EqualFold(f.Name, folderName+"/level-init.dat") ||
					strings.EqualFold(f.Name, folderName+"/level.datmetadata")) {
				//cwlog.DoLogCW("sm-inject: found " + f.Name)
				file, err := f.Open()
				if err != nil {
					cwlog.DoLogCW("sm-inject: unable to open " + f.Name + ": " + err.Error())
				} else {
					defer file.Close()

					read, derr := f.Open()

					if derr != nil {
						cwlog.DoLogCW("sm-inject: unable to read: " + f.Name + ", " + derr.Error())
					} else {
						data, rerr := ioutil.ReadAll(read)
						dlen := uint64(len(data))
						if rerr != nil && rerr != io.EOF {
							cwlog.DoLogCW("Unable to read file: " + f.Name + ", " + rerr.Error())
						} else if dlen != f.UncompressedSize64 {
							sbuf := fmt.Sprintf("%v vs %v", dlen, f.UncompressedSize64)
							cwlog.DoLogCW("Sizes did not match: " + f.Name + ", " + sbuf)
						} else {
							defer read.Close()
							//Put in new zip file here!
							tmp := zipFilesData{Name: f.Name, Data: data}
							zipFiles = append(zipFiles, tmp)
						}
					}

				}
			}
		}

		/* Add softmod files */

		for _, z := range zipFiles {
			buf := fmt.Sprintf("Name: %v, Size: %v", filepath.Base(z.Name), humanize.Bytes(uint64(len(z.Data))))
			fmt.Println(buf)
		}
	}

	/* Generate config file for Factorio server, if it fails stop everything.*/
	if !fact.GenerateFactorioConfig() {
		fact.FactAutoStart = false
		fact.CMS(cfg.Local.Channel.ChatChannel, "Unable to generate config file for Factorio server.")
		return
	}

	/* Relaunch Throttling */
	throt := glob.RelaunchThrottle
	if throt > 0 {

		delay := throt * throt * 10

		if delay > 0 {
			cwlog.DoLogCW(fmt.Sprintf("Automatically rebooting Factorio in %d seconds.", delay))
			for i := 0; i < delay*11 && throt > 0 && glob.ServerRunning; i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	/* Timer gets longer each reboot */
	glob.RelaunchThrottle = (throt + 1)

	var err error
	var tempargs []string

	/* Factorio launch parameters */
	rconport := cfg.Local.Port + cfg.Global.Options.RconOffset
	rconportStr := fmt.Sprintf("%v", rconport)
	rconpass := glob.RandomBase64String(256)
	glob.RCONPass = rconpass
	cfg.Local.RCONPass = rconpass
	cfg.WriteLCfg()

	port := cfg.Local.Port
	postStr := fmt.Sprintf("%v", port)
	serversettings := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ServSettingsName

	tempargs = append(tempargs, "--start-server")
	tempargs = append(tempargs, fileName)
	tempargs = append(tempargs, "--rcon-port")
	tempargs = append(tempargs, rconportStr)

	tempargs = append(tempargs, "--rcon-password")
	tempargs = append(tempargs, rconpass)

	tempargs = append(tempargs, "--port")
	tempargs = append(tempargs, postStr)

	tempargs = append(tempargs, "--server-settings")
	tempargs = append(tempargs, serversettings)

	/* Auth Server Bans ( global bans ) */
	if cfg.Global.Options.UseAuthserver {
		tempargs = append(tempargs, "--use-authserver-bans")
	}

	/* Whitelist */
	if cfg.Local.Options.Whitelist {
		tempargs = append(tempargs, "--use-server-whitelist")
		tempargs = append(tempargs, "true")
	}

	/* Write or delete whitelist */
	count := fact.WriteWhitelist()
	if count > 0 && cfg.Local.Options.Whitelist {
		cwlog.DoLogCW(fmt.Sprintf("Whitelist of %v players written.", count))
	}

	//Clear mod load string
	fact.ModList = []string{}

	/* Run Factorio */
	var cmd *exec.Cmd = exec.Command(fact.GetFactorioBinary(), tempargs...)

	/* Hide RCON password and port */
	for i, targ := range tempargs {
		if targ == rconpass {
			tempargs[i] = "***private***"
		} else if targ == rconportStr {
			/* funny, and impossible port number  */
			tempargs[i] = "69420"
		}
	}

	/* Okay, prep for factorio launch */
	fact.SetFactRunning(true)
	fact.FactorioBooted = false

	fact.Gametime = (constants.Unknown)
	glob.NoResponseCount = 0
	cwlog.DoLogCW("Factorio booting...")

	/* Launch Factorio */
	cwlog.DoLogCW("Executing: " + fact.GetFactorioBinary() + " " + strings.Join(tempargs, " "))

	LinuxSetProcessGroup(cmd)
	/* Connect Factorio stdout to a buffer for processing */
	fact.GameBuffer = new(bytes.Buffer)
	logwriter := io.MultiWriter(fact.GameBuffer)
	cmd.Stdout = logwriter
	/* Stdin */
	tpipe, errp := cmd.StdinPipe()

	/* Factorio is not happy. */
	if errp != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to execute cmd.StdinPipe() Details: %s", errp))
		/* close lock  */
		fact.DoExit(true)
		return
	}

	/* Save pipe */
	if tpipe != nil && err == nil {
		fact.PipeLock.Lock()
		fact.Pipe = tpipe
		fact.PipeLock.Unlock()
	}

	/* Handle launch errors */
	err = cmd.Start()
	if err != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to start the game. Details: %s", err))
		fact.DoExit(true)
		return
	}
}

func ConfigSoftMod() {
	fact.WriteFact("/cname " + strings.ToUpper(cfg.Local.Callsign+"-"+cfg.Local.Name))

	/* Config new-player restrictions */
	if cfg.Local.Options.SoftModOptions.Restrict {
		fact.WriteFact("/restrict on")
	} else {
		fact.WriteFact("/restrict off")
	}

	/* Config friendly fire */
	if cfg.Local.Options.SoftModOptions.FriendlyFire {
		fact.WriteFact("/friendlyfire on")
	} else {
		fact.WriteFact("/friendlyfire off")
	}

	/* Config reset-interval */
	if fact.NextReset != "" {
		fact.WriteFact("/resetint " + fact.NextReset)
	}
	if fact.TillReset != "" {
		fact.WriteFact("/resetdur " + fact.TillReset + " (" + strings.ToUpper(cfg.Local.Options.Schedule) + ")")
	}
	if cfg.Local.Options.SoftModOptions.CleanMap {
		fact.WriteFact("/cleanmap")
	}
	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		fact.WriteFact("/blueprints off")
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		fact.WriteFact("/enablecheats on")
	}

	/* Patreon list */
	if len(disc.RoleList.Patreons) > 0 {
		fact.WriteFact("/patreonlist " + strings.Join(disc.RoleList.Patreons, ","))
	}
	if len(disc.RoleList.NitroBooster) > 0 {
		fact.WriteFact("/nitrolist " + strings.Join(disc.RoleList.NitroBooster, ","))
	}
}
