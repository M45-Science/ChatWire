package admin

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

func Rewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	argnum := len(args)
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath

	if argnum == 1 {
		num, err := strconv.Atoi(args[0])
		if err == nil {
			if num < 0 || num > 9999 {
				fact.CMS(m.ChannelID, "That isn't an acceptable number.")
				return
			} else {
				autoSaveStr := fmt.Sprintf("_autosave%v.zip", num)
				_, err := os.Stat(path + "/" + autoSaveStr)
				notfound := os.IsNotExist(err)

				if notfound {
					fact.CMS(m.ChannelID, "I don't see that autosave.")
				} else {
					fact.SetAutoStart(false)
					fact.QuitFactorio()
					for x := 0; x < 60 && fact.IsFactRunning(); x++ {
						time.Sleep(time.Millisecond * 100)
						if x == 59 {
							fact.CMS(m.ChannelID, "Factorio may be frozen, canceling rewind.")
							return
						}
					}
					currentTime := time.Now().Local()
					err = os.Chtimes(path+"/"+autoSaveStr, currentTime, currentTime)
					if err != nil {
						fact.CMS(m.ChannelID, "Was unable to load the autosave.")
					}
					fact.CMS(m.ChannelID, fmt.Sprintf("Loading autosave%v", num))
					fact.SetAutoStart(true)

				}
			}
		} else {
			if strings.EqualFold(args[0], "list") {
				files, err := ioutil.ReadDir(path)
				if err != nil {
					log.Fatal(err)
					fact.CMS(m.ChannelID, "Something went wrong, sorry.")
				}

				buf := ""
				lastNum := -1
				for _, f := range files {
					fName := f.Name()
					if strings.HasPrefix(fName, "_autosave") && strings.HasSuffix(fName, ".zip") {
						fTmp := strings.TrimPrefix(fName, "_autosave")
						fNumStr := strings.TrimSuffix(fTmp, ".zip")
						fNum, err := strconv.Atoi(fNumStr)
						if err != nil {
							continue
						}
						if fNum != lastNum+1 {
							if buf == "" {
								buf = "Autosaves: "
							} else {
								buf = buf + "-" + strconv.Itoa(lastNum) + ", "
							}
							buf = buf + fmt.Sprintf("%v", fNum)
						} else {
							//Silent
						}
						lastNum = fNum
					}
				}
				buf = buf + "-" + strconv.Itoa(lastNum)

				fact.CMS(m.ChannelID, buf)
				return
			}
			fact.CMS(m.ChannelID, "I didn't find a valid number, or `list`.")
			return
		}
	} else {
		fact.CMS(m.ChannelID, "Please supply a autosave number.")
	}

}
