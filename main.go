package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"./commands"
	//"./commands/admin"
	"./glob"
	"./support"
	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	glob.MaxMapTypes = len(glob.MapTypes) //Just to make it easier

	glob.Sav_timer = time.Now()
	glob.Gametime = "na"
	glob.Running = false
	glob.Shutdown = false
	support.Config.LoadEnv()

	if support.Config.AutoStart == "false" {
		glob.Shutdown = true
		glob.Running = false
		support.Log("Autostart disabled, not loading factorio.")
	}

	t := time.Now()
	glob.OurLogname = fmt.Sprintf("logs/log-%v.log", t.UnixNano())

	os.MkdirAll("logs", os.ModePerm)

	// Do not exit the app on this error.
	if err := os.Remove(glob.OurLogname); err != nil {
		support.Log("Factorio.log doesn't exist, continuing anyway")
	}

	logging, err := os.OpenFile(glob.OurLogname, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to open factorio.log\nDetails: %s", time.Now(), err))
		os.Exit(1)
	}

	mwriter := io.MultiWriter(logging, os.Stdout)
	//Pre-init
	cmd := exec.Command("/usr/bin/time", "")
	cmd.Stderr = os.Stderr
	cmd.Stdout = mwriter
	glob.Pipe, err = cmd.StdinPipe()

	startbot()
	support.Chat()

	support.LoadPlayers()
	support.LoadRecord()

	_, err = glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Bot online.")
	if err != nil {
		support.ErrorLog(err)
	}

	//Wait for discord before trying
	time.Sleep(5 * time.Second)

	go func() {
		for {
			time.Sleep(time.Second)
			if glob.DS == nil {
				continue //Wait if bot isn't ready yet
			}

			if (!glob.Running || glob.Shutdown) && (glob.Reboot || glob.QueueReload) { //Reboot whole bot if set to
				if glob.QueueReload {
					_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server offline, performing scheduled reload.")
					if err != nil {
						support.ErrorLog(err)
					}
				}
				os.Exit(1)
			} else if glob.Running && !glob.Shutdown { //Currently running normally
				glob.NoResponseCount = glob.NoResponseCount + 1

				if glob.Running {
					_, err := io.WriteString(glob.Pipe, "/time\n")
					if err != nil {
						support.ErrorLog(err)
					}
					//glob.NoResponseCount = glob.NoResponseCount + 1
					if glob.NoResponseCount == 60 {
						_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server has not responded for 60 seconds...")
						if err != nil {
							support.ErrorLog(err)
						}
					}
					if glob.NoResponseCount == 120 {
						glob.NoResponseCount = 0
						_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server was unresponsive for 120 seconds... restarting it.")
						if err != nil {
							support.ErrorLog(err)
						}
						//Exit, to remove zombies
						os.Exit(1)
					}
				}
			} else if !glob.Running && !glob.Shutdown { //Isn't running, but we aren't supposed to be shutdown.

				command := support.Config.ZipScript
				out, errs := exec.Command(command, support.Config.ServerLetter).Output()
				if errs != nil {
					support.ErrorLog(errs)
				}
				if out != nil {
					//buf := fmt.Sprintf("ran: %s args: %s out: %s\n", command, arguments, out)
					//support.Log(buf)
				}
				//Relaunch Throttle
				if glob.RelaunchThrottle > 0 {

					delay := glob.RelaunchThrottle * glob.RelaunchThrottle * 10
					glob.RelaunchThrottle = glob.RelaunchThrottle + 1

					buf := fmt.Sprintf("Sleeping for %d seconds, before auto-relaunch.", delay)
					_, err = glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, buf)
					if err != nil {
						support.ErrorLog(err)
					}

					time.Sleep(time.Duration(delay) * time.Second)

				}

				cmd := exec.Command(support.Config.Executable, support.Config.LaunchParameters...)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				cmd.Stderr = os.Stderr
				cmd.Stdout = mwriter
				glob.Pipe, err = cmd.StdinPipe()
				if err != nil {
					support.ErrorLog(err)
				}
				defer cmd.Wait() //Zombie Fix

				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to execute cmd.StdinPipe()\nDetails: %s", time.Now(), err))
					glob.Running = false
				}

				err := cmd.Start()

				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to start the server\nDetails: %s", time.Now(), err))
					glob.Running = false
				}
				//This is okay, because if server doesn't respond it will be auto-rebooted.
				glob.Running = true

				glob.Gametime = "na"
				glob.Sav_timer = time.Now()
				glob.NoResponseCount = 0
				_, err = glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio booting...")
				if err != nil {
					support.ErrorLog(err)
				}
			}
		}
	}()

	go func() {
		Console := bufio.NewReader(os.Stdin)
		for {
			time.Sleep(100 * time.Millisecond)
			line, _, err := Console.ReadLine()
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to read the input to pass as input to the console\nDetails: %s", time.Now(), err))
				glob.Running = false
			}
			if len(line) > 1 {
				if glob.Running {
					_, err = io.WriteString(glob.Pipe, fmt.Sprintf("%s\n", line))
					if err != nil {
						support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass input to the console\nDetails: %s", time.Now(), err))
						glob.Running = false
					}
				}
			}
		}
	}()

	go func() {
		for {
			time.Sleep(15 * time.Second)
			if glob.Running {
				_, err = io.WriteString(glob.Pipe, "/p o c\n")
			}
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Second)

			// Look for signal files
			if _, err := os.Stat(".upgrade"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0
				glob.Shutdown = true

				if err := os.Remove(".upgrade"); err != nil {
					support.Log(".upgrade disappeared?")
				}
				if glob.Running {
					go func() {
						if glob.Running {
							_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Updating Factorio!")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,1,0]Factorio is shutting down in 30 seconds for a version upgrade![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,0,0]Factorio is shutting down in 30 seconds for a version upgrade!![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=0,1,1]Factorio is shutting down in 30 seconds for a version upgrade!!![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							time.Sleep(30 * time.Second)
							_, err = io.WriteString(glob.Pipe, "/quit\n")
							if err != nil {
								support.ErrorLog(err)
							}
						}
					}()
				}

				glob.Shutdown = true
			} else if _, err := os.Stat(".queue"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0

				if err := os.Remove(".queue"); err != nil {
					support.Log(".queue file disappeared?")
				}
				glob.QueueReload = true
				_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "**bot reload queued!**")
				if err != nil {
					support.ErrorLog(err)
				}
			} else if _, err := os.Stat(".start"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0

				if err := os.Remove(".start"); err != nil {
					support.Log(".start file disappeared?")
				}
				if !glob.Running {
					if err := os.Remove(".offline"); err != nil {
						support.Log(".offline missing...")
					}
					glob.Shutdown = false
					_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio starting!")
					if err != nil {
						support.ErrorLog(err)
					}
				}
			} else if _, err := os.Stat(".restart"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0
				glob.Shutdown = true

				if err := os.Remove(".restart"); err != nil {
					support.Log(".restart file disappeared?")
				}
				if glob.Running {
					go func() {
						if glob.Running {
							_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio restarting!")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,1,0]Server restarting in 30 seconds.[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,0,0]Server restarting in 30 seconds..[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=0,1,1]Server restarting in 30 seconds...[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}

							time.Sleep(30 * time.Second)
							_, err = io.WriteString(glob.Pipe, "/quit\n")
							if err != nil {
								support.ErrorLog(err)
							}
						}
					}()
				}

				glob.Shutdown = false
			} else if _, err := os.Stat(".qrestart"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0
				glob.Shutdown = true

				if err := os.Remove(".qrestart"); err != nil {
					support.Log(".qrestart file disappeared?")
				}
				if glob.Running {
					go func() {
						if glob.Running {
							_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Quick restarting!")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,1,0]Server quick restarting.[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,0,1]Server quick restarting..[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=0,1,1]Server quick restarting...[/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							time.Sleep(5 * time.Second)
							_, err = io.WriteString(glob.Pipe, "/quit\n")
							if err != nil {
								support.ErrorLog(err)
							}
						}
					}()
				}
				glob.Shutdown = false
			} else if _, err := os.Stat(".shutdown"); !os.IsNotExist(err) {
				glob.NoResponseCount = 0
				if err := os.Remove(".shutdown"); err != nil {
					support.Log(".shutdown disappeared?")
				}
				if glob.Running {
					glob.Shutdown = true
					go func() {
						if glob.Running {
							_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio is shutting down for maintenance!")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,1,0]Factorio is shutting down in 30 seconds for system maintenance![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=1,0,0]Factorio is shutting down in 30 seconds for system maintenance!![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							_, err = io.WriteString(glob.Pipe, "[color=0,1,1]Factorio is shutting down in 30 seconds for system maintenance!!![/color]\n")
							if err != nil {
								support.ErrorLog(err)
							}
							time.Sleep(30 * time.Second)
							_, err = io.WriteString(glob.Pipe, "/quit\n")
							if err != nil {
								support.ErrorLog(err)
							}
						}
					}()
				}

				glob.Shutdown = true
			}
		}

	}()
	quithandle()
}

func startbot() {
	glob.Sav_timer = time.Now()
	glob.NoResponseCount = 0

	// No hard coding the token }:<
	commands.RegisterCommands()
	support.Log("Starting bot...")

	//Delete old signal files
	if err := os.Remove(".start"); err != nil {
		//support.Log(".restart not found... ")
	}
	if err := os.Remove(".restart"); err != nil {
		//support.Log(".restart not found... ")
	}
	if err := os.Remove(".qrestart"); err != nil {
		//support.Log(".qrestart not found... ")
	}
	if err := os.Remove(".shutdown"); err != nil {
		//support.Log(".shutdown not found... ")
	}
	if err := os.Remove(".upgrade"); err != nil {
		//support.Log(".upgrade not found... ")
	}
	if err := os.Remove(".queue"); err != nil {
		//support.Log(".upgrade not found... ")
	}

	bot, err := discordgo.New("Bot " + support.Config.DiscordToken)
	glob.DS = bot
	if err != nil {
		support.Log("Error creating Discord session. ")
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to create the Discord session\nDetails: %s", time.Now(), err))
		os.Exit(1)
		return
	}

	err = bot.Open()

	if err != nil {
		support.Log("error opening connection")
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to connect to Discord\nDetails: %s", time.Now(), err))
		os.Exit(1)
		return
	}

	bot.AddHandler(messageCreate)
	bot.UpdateStatus(0, support.Config.GameName)
}

func quithandle() {
	support.Log("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	if glob.Running {
		_, err := io.WriteString(glob.Pipe, "Server killed.\n")
		if err != nil {
			support.ErrorLog(err)
		}
		_, err = io.WriteString(glob.Pipe, "Server killed..\n")
		if err != nil {
			support.ErrorLog(err)
		}
		_, err = io.WriteString(glob.Pipe, "Server killed...\n")
		if err != nil {
			support.ErrorLog(err)
		}
		_, err = io.WriteString(glob.Pipe, "/quit\n")
		if err != nil {
			support.ErrorLog(err)
		}
	}
	glob.NoResponseCount = 0
	glob.Shutdown = true
	_, errb := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Service killed, shutting down.")
	if errb != nil {
		support.ErrorLog(errb)
	}
	//Wait for save to finish if possible, 30 seconds max
	for x := 0; x < 30 && glob.Running; x++ {
		time.Sleep(100 * time.Millisecond)
	}
	// Cleanly close down the Discord session.
	glob.DS.Close()
	os.Exit(1)
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Print("[" + m.Author.Username + "] " + m.Content)

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	if m.ChannelID == support.Config.FactorioChannelID {
		//Command stuff
		if strings.HasPrefix(m.Content, support.Config.Prefix) {
			myarg := ""

			command := strings.Split(m.Content[1:len(m.Content)], " ")
			name := strings.ToLower(command[0])
			if len(command) >= 2 {
				myarg = command[1]
				glob.CharName = myarg
			}
			commands.RunCommand(name, s, m)
			return
		}

		//Chat message handling
		if glob.Running { // Don't bother if we arne't running...
			if !strings.Contains(strings.ToLower(m.Content), "!clear") {

				//Clean strings
				cleanedstr := m.Content
				cleanedstr = strings.Replace(cleanedstr, "\n", " ", -1)
				cleanedstr = strings.Replace(cleanedstr, "\r", " ", -1)
				cleanedstr = strings.Replace(cleanedstr, "\t", " ", -1)

				_, err := io.WriteString(glob.Pipe, fmt.Sprintf("[color=0,1,1][DISCORD-CHAT][/color] [color=1,1,0]%s:[/color] [color=0,1,1]%s[/color]\n", m.Author.Username, cleanedstr))
				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass Discord chat to in-game\nDetails: %s", time.Now(), err))
					glob.Running = false
				}
			}
		}
		return
	}
}
