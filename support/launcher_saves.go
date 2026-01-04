package support

import (
	"io/fs"
	"os"
	"sort"
	"strings"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
)

func GetSaveGame(doInject bool) (foundGood bool, fileName string, fileDir string) {
	path := cfg.GetSavesFolder()

	files, err := os.ReadDir(path)

	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW("Unable to read saves folder, stopping.")
		return false, "", ""
	}

	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files and directories
		if !f.IsDir() {
			if strings.HasSuffix(f.Name(), ".zip") && !strings.HasSuffix(f.Name(), "tmp.zip") {
				tempf = append(tempf, f)
			}
		}
	}

	//Newest first
	sort.Slice(tempf, func(i, j int) bool {
		iInfo, _ := tempf[i].Info()
		jInfo, _ := tempf[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
		//return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	numSaves := len(tempf)
	if numSaves <= 0 {
		fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, "No saves found, stopping.")
		return false, "", ""
	}

	for pos := 0; pos < numSaves; pos++ {
		name := tempf[pos].Name()

		if name == "" {
			continue
		}

		showError := false
		if pos == 0 {
			showError = true
		}
		good, folder := fact.CheckSave(path, name, showError)
		if good {
			return true, path + "/" + name, folder
		}

	}

	return false, "", ""
}
