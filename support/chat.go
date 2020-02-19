package support

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"../glob/"
	"github.com/bwmarrin/discordgo"
	"github.com/hpcloud/tail"
)

func PlayerFound(pname string) bool {
	LoadPlayers()

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	for i := 0; i <= glob.PlayerListMax; i++ {
		if glob.PlayerList[i] == pname {

			//Found in list
			//Log("Player found!")
			return true
		}
	}

	//Not in list, add them
	if glob.PlayerListMax < glob.MaxPlayers { //Don't go over max
		glob.PlayerList[glob.PlayerListMax] = pname
		glob.PlayerListMax++
	}

	WritePlayers()
	//Log("Player not found!")
	return false
}

func LoadPlayers() {
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()

	filedata, err := ioutil.ReadFile(Config.DBFile)
	if err != nil {
		Log("Couldn't read dbfile, skipping...")
		return
	}

	if filedata != nil {
		//Log("File data not nil.")
		glob.PlayerListMax = 0

		dblines := strings.Split(string(filedata), ":")
		numlines := len(dblines)

		for pos := 0; pos < numlines; pos++ {
			items := strings.Split(string(dblines[pos]), ",")
			numitems := len(items) - 1

			for x := 0; x < numitems; x++ {
				//buf := fmt.Sprintf("Line: %d, Item: %d, Contents: %s", pos, x, items[x])
				//Log(buf)

				if items[x] != "" {
					glob.PlayerList[glob.PlayerListMax] = items[x]
					glob.PlayerListMax++

					//buf = fmt.Sprintf("Adding player: %s\n", items[x])
					//Log(buf)
				}
			}

		}
		//Log("Player list loaded...")
	}
}

func WritePlayers() {
	//Write to file
	glob.PlayerListWriteLock.Lock()
	defer glob.PlayerListWriteLock.Unlock()
	buffer := ""

	fo, err := os.Create(Config.DBFile)
	if err != nil {
		Log("Couldn't open db file, skipping...")
		return
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	for i := 0; i < glob.PlayerListMax; i++ {
		//Filter comma from names, just in case
		name := strings.ReplaceAll(glob.PlayerList[i], ",", "")
		nameb := strings.ReplaceAll(name, ":", "")
		buffer = buffer + fmt.Sprintf("%s,", nameb)
	}

	err = ioutil.WriteFile(Config.DBFile, []byte(buffer), 0644)

	if err != nil {
		Log("Couldn't write db file.")
	}
}

func writerecord() {
	//Write to file
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	fo, err := os.Create(Config.MaxFile)
	if err != nil {
		Log("Couldn't open max file, skipping...")
		return
	}
	// close fo on exit and check for its returned error
	defer func() {
		if err := fo.Close(); err != nil {
			panic(err)
		}
	}()

	buffer := fmt.Sprintf("%d", glob.RecordPlayers)

	err = ioutil.WriteFile(Config.MaxFile, []byte(buffer), 0644)

	if err != nil {
		Log("Couldn't write max file.")
	}
}

func LoadRecord() {
	glob.RecordPlayersWriteLock.Lock()
	defer glob.RecordPlayersWriteLock.Unlock()

	glob.RecordPlayersLock.Lock()
	defer glob.RecordPlayersLock.Unlock()

	filedata, err := ioutil.ReadFile(Config.MaxFile)
	if err != nil {
		Log("Couldn't read max file, skipping...")
		return
	}

	if filedata != nil {
		readstrnum := string(filedata)
		readnum, _ := strconv.Atoi(readstrnum)

		if readnum > glob.RecordPlayers {
			glob.RecordPlayers = readnum
		}
		Log("MaxFile read.")
	}
}

// Chat pipes in-game chat to Discord.
func Chat() {
	time.Sleep(5 * time.Second)
	go func() {
		for {
			time.Sleep(100 * time.Millisecond)

			t, err := tail.TailFile(glob.OurLogname, tail.Config{Follow: true})
			if err != nil {
				ErrorLog(fmt.Errorf("%s: An error occurred when attempting to tail factorio.log\nDetails: %s", time.Now(), err))
				os.Exit(1)
			}
			for line := range t.Lines {
				//Ignore blanks
				if line.Text == "" {
					break
				}
				if len(line.Text) > 1900 {
					//Message too long
					break
				}

				if len(line.Text) > 0 && !strings.Contains(line.Text, "<server>") {
					glob.NoResponseCount = 0 //Server is alive
					glob.Running = true

					if !strings.Contains(line.Text, "[CHAT]") {
						TmpList := strings.Split(line.Text, " ")

						//Send join/leave to Discord
						if strings.Contains(line.Text, "Online players") {

							poc := strings.Join(TmpList[2:], " ")
							poc = strings.ReplaceAll(poc, "(", "")
							poc = strings.ReplaceAll(poc, ")", "")
							poc = strings.ReplaceAll(poc, ":", "")

							glob.NumPlayers, _ = strconv.Atoi(poc)

							if glob.NumPlayers > glob.RecordPlayers {
								glob.RecordPlayersLock.Lock()
								glob.RecordPlayers = glob.NumPlayers
								glob.RecordPlayersLock.Unlock()
								writerecord()

								buf := fmt.Sprintf("**New record!** Players online: %v\n", glob.RecordPlayers)
								_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, buf)
								if err != nil {
									ErrorLog(err)
								}
								//write to factorio as well
								buf = strings.ReplaceAll(buf, "*", "") //Remove bold
								_, err = io.WriteString(glob.Pipe, buf)
								if err != nil {
									ErrorLog(err)
								}

							}

							oldch, errch := glob.DS.Channel(Config.FactorioChannelID)

							if errch == nil {
								newchname := ""
								oldchname := oldch.Name

								if poc == "0" {
									newchname = fmt.Sprintf("⚫%s", Config.ChannelName)
									if glob.QueueReload == true {
										_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, "No players online, performing scheduled reload.")
										if err != nil {
											ErrorLog(err)
										}
										if glob.Running {
											_, err = io.WriteString(glob.Pipe, "/quit\n")
											if err != nil {
												ErrorLog(err)
											}
										}
										glob.Reboot = true

									}
								} else {
									newchname = fmt.Sprintf("🔴%s_%v╱%v", Config.ChannelName, glob.NumPlayers, glob.RecordPlayers)
								}

								//Dont clutter audit log if name didn't change
								if newchname != oldchname || glob.Refresh {
									glob.Refresh = false
									//_, _ = glob.DS.ChannelEdit(Config.FactorioChannelID, newchname)

									chpos, _ := strconv.Atoi(Config.ChannelPos)
									chpos = chpos + 500
									_, aerr := glob.DS.ChannelEditComplex(Config.FactorioChannelID, &discordgo.ChannelEdit{Name: newchname, Position: chpos})
									if aerr != nil {
										ErrorLog(aerr)
									}
								}

								//_, _ = glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("%s players online", poc))
							}
						}
						//Join message, with delay
						if strings.Contains(line.Text, "[JOIN]") {
							_, err = io.WriteString(glob.Pipe, "/p o c\n")

							if err != nil {
								ErrorLog(fmt.Errorf("%s: error when getting player count\nDetails: %s", time.Now(), err))
							}
							pname := TmpList[3]
							glob.NumLogins = glob.NumLogins + 1
							newusername := ""
							if PlayerFound(pname) == false {
								newusername = " *(New Player)* "

								//Don't block, make new thread
								go func() {
									time.Sleep(20 * time.Second)
									_, err := io.WriteString(glob.Pipe, fmt.Sprintf("/w %s [color=0,1,1]Welcome! use tilde/tick ( ` or ~ key ) to chat! Chat /online to see players online.[/color]\n", pname))
									time.Sleep(10 * time.Second)
									_, err = io.WriteString(glob.Pipe, fmt.Sprintf("/w %s [color=0,1,1]Check out our Discord server at: https://discord.gg/Ps2jnm7[/color]\n", pname))
									time.Sleep(10 * time.Second)
									_, err = io.WriteString(glob.Pipe, fmt.Sprintf("/w %s [color=0,1,1]Please report griefers on the Discord, so we can ban them![/color]\n", pname))

									if newusername != "" {
										time.Sleep(10 * time.Second)
										_, err = io.WriteString(glob.Pipe, fmt.Sprintf("/w %s [color=0,1,1]You are currently a new player on this map, and some options will be disabled for you.[/color]\n", pname))
									}

									if err != nil {
										ErrorLog(fmt.Errorf("%s: error sending greeting\nDetails: %s", time.Now(), err))
									}
								}()
							}
							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("`%-13s` **%s**%s\n*Logins today: %d, Players known: %d*", glob.Gametime, strings.Join(TmpList[3:], " "), newusername, glob.NumLogins, glob.PlayerListMax))
							if err != nil {
								ErrorLog(err)
							}

						}
						//Save on leave
						if strings.Contains(line.Text, "[LEAVE]") {
							_, err = io.WriteString(glob.Pipe, "/p o c\n")

							if err != nil {
								ErrorLog(fmt.Errorf("%s: error getting player count\nDetails: %s", time.Now(), err))
							}

							go func() {
								t := time.Now()
								// Don't save if we saved recently
								if t.Sub(glob.Sav_timer).Seconds() > 300 {

									_, err = io.WriteString(glob.Pipe, fmt.Sprintf("/server-save sav-%s\n", glob.Gametime))
									if err != nil {
										ErrorLog(fmt.Errorf("%s: Error when commanding LEAVE save.\nDetails: %s", time.Now(), err))
										glob.Running = false
									}
									glob.Sav_timer = time.Now()
								}

							}()
							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("`%-13s` *%s*", glob.Gametime, strings.Join(TmpList[3:], " ")))
							if err != nil {
								ErrorLog(err)
							}
						}
						//Send luamsg to Discord
						if strings.Contains(line.Text, "[MSG]") {

							TmpList := strings.Split(line.Text, " ")

							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("`%-13s` **(INFO)** %s", glob.Gametime, TmpList[1:]))
							if strings.Contains(line.Text, " was moved to trusted users.") {
								trustname := TmpList[1]
								trustname = strings.Replace(trustname, "[", "", -1)
								Log(trustname + " was promoted to trusted user.")
							}
							if err != nil {
								ErrorLog(err)
							}

						}
						if strings.Contains(line.Text, "[NFO]") {

							TmpList := strings.Split(line.Text, " ")

							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("`%-13s` %s", glob.Gametime, TmpList[1:]))
							if err != nil {
								ErrorLog(err)
							}

						}
						//For $online
						if strings.Contains(line.Text, "(online)") {

							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, line.Text)
							if err != nil {
								ErrorLog(err)
							}

						}

						//Loading map
						if strings.Contains(line.Text, "Loading map") {
							TmpList := strings.Split(line.Text, " ")

							//Strip file path
							fullpath := strings.Join(TmpList[4:7], " ")
							regaa := regexp.MustCompile(`\/.*?\/saves\/`)
							filename := regaa.ReplaceAllString(fullpath, "")
							filename = strings.Replace(filename, ":", "", -1)

							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("%s", filename))
							if err != nil {
								ErrorLog(err)
							}
						}
						//Loading mod
						//                                        if !strings.Contains(line.Text,"[CHAT]") && !strings.Contains(line.Text,"<server>") && strings.Contains(line.Text,"Loading mod") &&
						//					!strings.Contains(line.Text,"settings") && !strings.Contains(line.Text,"base") && !strings.Contains(line.Text, "core") {
						//                                                TmpList := strings.Split(line.Text, " ")
						//
						//                                                glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("(%s) %s", glob.Gametime, strings.Join(TmpList[4:8], " ")))
						//                                        }

						//Close message
						if !strings.Contains(line.Text, "[CHAT]") && !strings.Contains(line.Text, "<server>") && strings.Contains(line.Text, " Goodbye") {
							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, "Factorio is now offline.")
							if err != nil {
								ErrorLog(err)
							}
							glob.Running = false
							if glob.Reboot == true || glob.QueueReload == true {
								os.Exit(1)
							}
						}

						//Ready message
						if !strings.Contains(line.Text, "[CHAT]") && !strings.Contains(line.Text, "<server>") && strings.Contains(line.Text, " Matching server game ") && strings.Contains(line.Text, " has been created.") {
							_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, "Factorio is now online!")
							if err != nil {
								ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "/p o c\n")
							glob.Running = true
						}

						//Get in-game time
						ltl := strings.ToLower(line.Text)

						if (strings.Contains(ltl, " second") || strings.Contains(ltl, " minute") || strings.Contains(ltl, " hour") || strings.Contains(ltl, " day")) && !strings.Contains(line.Text, "[CHAT]") && !strings.Contains(line.Text, "<server>") {
							glob.Gametime = "gx-x-x-x"

							TmpList := strings.Split(ltl, " ")
							day := 0
							hour := 0
							minute := 0
							second := 0
							tmplen := len(TmpList)

							if tmplen > 1 {

								for x := 0; x < tmplen; x++ {
									if strings.Contains(TmpList[x], "day") {
										day, _ = strconv.Atoi(TmpList[x-1])
									} else if strings.Contains(TmpList[x], "hour") {
										hour, _ = strconv.Atoi(TmpList[x-1])
									} else if strings.Contains(TmpList[x], "minute") {
										minute, _ = strconv.Atoi(TmpList[x-1])
									} else if strings.Contains(TmpList[x], "second") {
										second, _ = strconv.Atoi(TmpList[x-1])
									}
								}
								glob.Gametime = fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)
							}

						}
						//Reset save timer
						if strings.Contains(line.Text, "Auto saving") || strings.Contains(line.Text, "Saving game") || strings.Contains(line.Text, "Saving Finished") || strings.Contains(line.Text, "[LEAVE]") {
							if !strings.Contains(line.Text, "[CHAT]") && !strings.Contains(line.Text, "<server>") {
								glob.Sav_timer = time.Now()
							}
						}

					} //End join/leave
					//Send chat to Discord
					if strings.Contains(line.Text, "[CHAT]") && !strings.Contains(line.Text, "<server>") {

						TmpList := strings.Split(line.Text, " ")
						TmpList[3] = strings.Replace(TmpList[3], ":", "", -1)

						cmess := strings.Join(TmpList[4:], " ")

						//Remove factorio tags
						rega := regexp.MustCompile(`\[/[^][]+\]`) //remove close tags [/color]

						regc := regexp.MustCompile(`\[color=(.*?)\]`) //remove [color=*]
						regd := regexp.MustCompile(`\[font=(.*?)\]`)  //remove [font=*]

						rege := regexp.MustCompile(`\[(.*?)=(.*?)\]`) //Sub others

						regf := regexp.MustCompile(`\*+`) //Remove discord markdown
						regg := regexp.MustCompile(`\~+`)
						regh := regexp.MustCompile(`\_+`)

						for regc.MatchString(cmess) || regd.MatchString(cmess) {
							//Remove colors/fonts
							cmess = regc.ReplaceAllString(cmess, "")
							cmess = regd.ReplaceAllString(cmess, "")
						}
						for rege.MatchString(cmess) {
							//Sub
							cmess = rege.ReplaceAllString(cmess, " [${1}: ${2}] ")
						}
						for rega.MatchString(cmess) {
							//Filter close tags
							cmess = rega.ReplaceAllString(cmess, "")
						}

						for regf.MatchString(cmess) || regg.MatchString(cmess) || regh.MatchString(cmess) {
							//Filter discord tags
							cmess = regf.ReplaceAllString(cmess, "")
							cmess = regg.ReplaceAllString(cmess, "")
							cmess = regh.ReplaceAllString(cmess, "")
						}

						if len(cmess) > 300 {
							cmess = fmt.Sprintf("%300s**... (message cut, too long!)**", cmess)
						}

						_, err := glob.DS.ChannelMessageSend(Config.FactorioChannelID, fmt.Sprintf("`%-13s` **%s:** %s", glob.Gametime, TmpList[3], cmess))
						if err != nil {
							ErrorLog(err)
						}

					} //End Chat
				} //End console filtered
			} //End Loop
		}
	}()
}
