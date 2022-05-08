package fact

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

/* Write record of max number of players online */
func WriteRecord() {
	/* Write to file */
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	fo, err := os.Create(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.FactorioPrefix + cfg.Local.Callsign + "/" + cfg.Global.Paths.DataFiles.RecordPlayers)
	if err != nil {
		cwlog.DoLogCW("Couldn't open max file, skipping...")
		return
	}
	/*  close fo on exit and check for its returned error */
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	buffer := fmt.Sprintf("%d", glob.RecordPlayers)

	err = ioutil.WriteFile(cfg.Global.Paths.Folders.ServersRoot+cfg.Global.Paths.FactorioPrefix+cfg.Local.Callsign+"/"+cfg.Global.Paths.DataFiles.RecordPlayers, []byte(buffer), 0644)

	if err != nil {
		cwlog.DoLogCW("Couldn't write max file.")
	}
}

/* Read record of max number of players online */
func LoadRecord() {
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	filedata, err := ioutil.ReadFile(cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.FactorioPrefix + cfg.Local.Callsign + "/" + cfg.Global.Paths.DataFiles.RecordPlayers)
	if err != nil {
		cwlog.DoLogCW("Couldn't read max file, skipping...")
		return
	}

	if filedata != nil {
		readstrnum := string(filedata)
		readnum, _ := strconv.Atoi(readstrnum)

		if readnum > glob.RecordPlayers {
			glob.RecordPlayers = readnum
		}
	}
}
