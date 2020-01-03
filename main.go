package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
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

var noresponsecount int

func main() {
	glob.Sav_timer = time.Now()
	glob.Gametime = "na"
	glob.Running = false
	glob.Shutdown = false
	support.Config.LoadEnv()

	// Do not exit the app on this error.
	if err := os.Remove("factorio.log"); err != nil {
		support.Log("Factorio.log doesn't exist, continuing anyway")
	}

	logging, err := os.OpenFile("factorio.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to open factorio.log\nDetails: %s", time.Now(), err))
	}

	start_bot()
	time.Sleep(5 * time.Second)
	support.Chat()

	mwriter := io.MultiWriter(logging, os.Stdout)
	support.LoadPlayers()
	support.LoadRecord()

	go func() {
		for {
			time.Sleep(1 * time.Second)
			if (!glob.Running || glob.Shutdown) && (glob.Reboot || glob.QueueReload) {
				_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server offline, performing scheduled reload.")
				if err != nil {
					support.ErrorLog(err)
				}
				os.Exit(1)
			} else if glob.Running && !glob.Shutdown {
				_, err = io.WriteString(glob.Pipe, "/time\n")
				if err != nil {
					noresponsecount = noresponsecount + 1
					if noresponsecount == 60 {
						_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server has not responded for 60 seconds...")
						if err != nil {
							support.ErrorLog(err)
						}
					}
					if noresponsecount == 120 {
						noresponsecount = 0
						_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Server was unresponsive for 120 seconds... restarting it.")
						if err != nil {
							support.ErrorLog(err)
						}
						//Exit, to remove zombies
						os.Exit(1)
					}
				}
			} else if !glob.Running && !glob.Shutdown {
				if glob.GCMD != nil {
					glob.GCMD.Process.Kill()
					glob.GCMD.Process.Release()
				}

				number := 0
				number, _ = strconv.Atoi(support.Config.ChannelPos)
				foo := "abcdefghijklmnopqrstuvwxyz"
				arguments := "./bin/bash /home/fact/softmod-up.sh", string(foo[number])
				out, errs := exec.Command(arguments).Output()
				if errs != nil {
					support.ErrorLog(errs)
				}
				buf := fmt.Sprintf("ran: %s, out: %s\n", arguments, out)
				support.Log(buf)

				time.Sleep(5 * time.Second)
				cmd := exec.Command(support.Config.Executable, support.Config.LaunchParameters...)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				cmd.Stderr = os.Stderr
				cmd.Stdout = mwriter
				glob.Pipe, err = cmd.StdinPipe()
				glob.GCMD = cmd
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
				noresponsecount = 0
				_, err = glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Bot online, server booting...")
				if err != nil {
					support.ErrorLog(err)
				}

			}
		}
	}()

	go func() {
		Console := bufio.NewReader(os.Stdin)
		for {
			line, _, err := Console.ReadLine()
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to read the input to pass as input to the console\nDetails: %s", time.Now(), err))
				glob.Running = false
			}
			if len(line) > 1 {
				_, err = io.WriteString(glob.Pipe, fmt.Sprintf("%s\n", line))
				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass input to the console\nDetails: %s", time.Now(), err))
					glob.Running = false
				}
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		for {
			time.Sleep(15 * time.Second)
			_, err = io.WriteString(glob.Pipe, "/p o c\r\n")
		}
	}()

	go func() {
		for {
			time.Sleep(2 * time.Second)

			// Look for signal files
			if _, err := os.Stat(".upgrade"); !os.IsNotExist(err) {
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".upgrade"); err != nil {
					support.Log(".upgrade disappeared?")
				}
				if glob.Running {
					go func() {
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
					}()
				}

				glob.Shutdown = true
			} else if _, err := os.Stat(".queue"); !os.IsNotExist(err) {
				noresponsecount = 0

				if err := os.Remove(".queue"); err != nil {
					support.Log(".queue file disappeared?")
				}
				glob.QueueReload = true
				_, err := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "**bot reload queued!**")
				if err != nil {
					support.ErrorLog(err)
				}
			} else if _, err := os.Stat(".start"); !os.IsNotExist(err) {
				noresponsecount = 0

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
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".restart"); err != nil {
					support.Log(".restart file disappeared?")
				}
				if glob.Running {
					go func() {
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
					}()
				}

				glob.Shutdown = false
			} else if _, err := os.Stat(".qrestart"); !os.IsNotExist(err) {
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".qrestart"); err != nil {
					support.Log(".qrestart file disappeared?")
				}
				if glob.Running {
					go func() {
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
					}()
				}
				glob.Shutdown = false
			} else if _, err := os.Stat(".shutdown"); !os.IsNotExist(err) {
				noresponsecount = 0
				if err := os.Remove(".shutdown"); err != nil {
					support.Log(".shutdown disappeared?")
				}
				if glob.Running {
					glob.Shutdown = true
					go func() {
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
					}()
				}

				glob.Shutdown = true
			}
		}

	}()
	quithandle()
}

func start_bot() {
	glob.Sav_timer = time.Now()
	noresponsecount = 0

	// No hard coding the token }:<
	discordToken := support.Config.DiscordToken
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

	bot, err := discordgo.New("Bot " + discordToken)
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

	time.Sleep(10 * time.Second)

	bot.AddHandler(messageCreate)
	bot.UpdateStatus(0, support.Config.GameName)
}

func quithandle() {
	support.Log("Bot is now running.  Press CTRL-C to exit.")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
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
	noresponsecount = 0
	glob.Shutdown = true
	_, err = glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, "Service killed, shutting down.")
	if err != nil {
		support.ErrorLog(err)
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

	if m.Author.ID == s.State.User.ID {
		return
	}

	if m.ChannelID == support.Config.FactorioChannelID {
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
		if !strings.Contains(strings.ToLower(m.Content), "!clear") {
			_, err := io.WriteString(glob.Pipe, fmt.Sprintf("[color=0,1,1][DISCORD-CHAT][/color] [color=1,1,0]%s:[/color] [color=0,1,1]%s[/color]\n", m.Author.Username, m.ContentWithMentionsReplaced()))
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass Discord chat to in-game\nDetails: %s", time.Now(), err))
				glob.Running = false
			}
		}
		return
	}
}
