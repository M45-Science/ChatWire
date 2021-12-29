package admin

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"

	"github.com/bwmarrin/discordgo"
)

//Load a different save-game
func Rewind(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	layoutUS := "01/02 03:04 PM MST"
	argnum := len(args)
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + cfg.Global.PathData.SaveFilePath

	//Correct number of arguments (1)
	if argnum == 1 {
		num, err := strconv.Atoi(args[0])
		//Seems to be a number
		if err == nil {
			if num > 0 || num < 9999 {
				//Check if file is valid and found
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
					//Touch file
					currentTime := time.Now().Local()
					err = os.Chtimes(path+"/"+autoSaveStr, currentTime, currentTime)
					if err != nil {
						fact.CMS(m.ChannelID, "Unable to load the autosave.")
					}
					fact.CMS(m.ChannelID, fmt.Sprintf("Loading autosave%v", num))
					fact.SetAutoStart(true)
					return
				}
			}
		} else {
			//List all autosaves
			if strings.EqualFold(args[0], "list") {
				files, err := ioutil.ReadDir(path)
				//We can't read saves dir
				if err != nil {
					log.Fatal(err)
					fact.CMS(m.ChannelID, "Something went wrong, sorry.")
				}

				rangeBuf := ""
				fileNames := ""
				lastNum := -1
				step := 1
				//Loop all files
				var tempf []fs.FileInfo
				for _, f := range files {
					if strings.HasPrefix(f.Name(), "_autosave") && strings.HasSuffix(f.Name(), ".zip") {
						tempf = append(tempf, f)
					}
				}

				sort.Slice(tempf, func(i, j int) bool {
					return tempf[i].ModTime().After(tempf[j].ModTime())
				})

				maxList := constants.MaxRewindResults
				for _, f := range tempf {
					maxList--
					if maxList <= 0 {
						break
					}
					fName := f.Name()

					//Check if its a properly name autosave
					if strings.HasPrefix(fName, "_autosave") && strings.HasSuffix(fName, ".zip") {
						fTmp := strings.TrimPrefix(fName, "_autosave")
						fNumStr := strings.TrimSuffix(fTmp, ".zip")
						fNum, err := strconv.Atoi(fNumStr) //autosave number
						//Nope, no valid numer
						if err != nil {
							continue
						}
						step++

						//Get mod date
						modDate := f.ModTime().Local().Format(layoutUS)
						//Not first file add commas/newlines
						if fileNames != "" {
							if step%2 == 0 {
								fileNames = fileNames + "\n"
							} else {
								fileNames = fileNames + ",   "
							}
						}
						//Add to list with mod date
						fileNames = fileNames + fmt.Sprintf("(%15v ): #%3v", modDate, fNum)

						//autosave number range list
						//If number is not sequential, save end of range and print it
						if fNum != lastNum-1 {
							//If we just started, add prefix, otherwise add dash and the end of the range, with comma for next item.
							if rangeBuf == "" {
								rangeBuf = "Autosaves: "
							} else {
								rangeBuf = rangeBuf + "-" + strconv.Itoa(lastNum) + ", "
							}
							rangeBuf = rangeBuf + fmt.Sprintf("%v", fNum)
						}
						lastNum = fNum //Save for compairsion next loop
					}
				}
				//Add last item to range
				rangeBuf = rangeBuf + "-" + strconv.Itoa(lastNum)

				if lastNum == -1 {
					fact.CMS(m.ChannelID, "No autosaves found.")
				} else {
					fact.CMS(m.ChannelID, "```\n"+rangeBuf+"\n\n"+fileNames+"\n```")
				}
				return
			}
		}
	}

	fact.CMS(m.ChannelID, "Not a valid autosave number, try `list`.")
	return
}
