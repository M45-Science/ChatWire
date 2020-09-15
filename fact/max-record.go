package fact

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"

	"../config"
	"../glob"
	"../logs"
)

func WriteRecord() {
	//Write to file
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	fo, err := os.Create(config.Config.MaxFile)
	if err != nil {
		logs.Log("Couldn't open max file, skipping...")
		return
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	buffer := fmt.Sprintf("%d", glob.RecordPlayers)

	err = ioutil.WriteFile(config.Config.MaxFile, []byte(buffer), 0644)

	if err != nil {
		logs.Log("Couldn't write max file.")
	}
}

func LoadRecord() {
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	filedata, err := ioutil.ReadFile(config.Config.MaxFile)
	if err != nil {
		logs.Log("Couldn't read max file, skipping...")
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
