package fact

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/Distortions81/M45-ChatWire/cfg"
	"github.com/Distortions81/M45-ChatWire/glob"
)

func WriteRecord() {
	//Write to file
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	fo, err := os.Create(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.RecordPlayersFilename)
	if err != nil {
		log.Println("Couldn't open max file, skipping...")
		return
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	buffer := fmt.Sprintf("%d", glob.RecordPlayers)

	err = ioutil.WriteFile(cfg.Global.PathData.FactorioServersRoot+cfg.Global.PathData.FactorioHomePrefix+cfg.Local.ServerCallsign+"/"+cfg.Global.PathData.RecordPlayersFilename, []byte(buffer), 0644)

	if err != nil {
		log.Println("Couldn't write max file.")
	}
}

func LoadRecord() {
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	filedata, err := ioutil.ReadFile(cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.RecordPlayersFilename)
	if err != nil {
		log.Println("Couldn't read max file, skipping...")
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
