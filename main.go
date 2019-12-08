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
	"sync"
	"syscall"
	"time"

	"./commands"
	//"./commands/admin"
	"./glob"
	"./support"
	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

// Running is the boolean that tells if the server is running or not
//var Running bool
//var Shutdown bool
var last_players_online string
var players_online string
var noresponsecount int

// Pipe is an WriteCloser interface
// var Pipe io.WriteCloser

// Session is a discordgo session
var Session *discordgo.Session

func main() {
	glob.Sav_timer = time.Now()
	glob.Gametime = "na"
	glob.Running = false
	support.Config.LoadEnv()

	// Do not exit the app on this error.
	if err := os.Remove("factorio.log"); err != nil {
		fmt.Println("Factorio.log doesn't exist, continuing anyway")
	}

	logging, err := os.OpenFile("factorio.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to open factorio.log\nDetails: %s", time.Now(), err))
	}

	mwriter := io.MultiWriter(logging, os.Stdout)
	var wg sync.WaitGroup
	wg.Add(4)

	go func() {
		defer wg.Done()
		for {
			time.Sleep(1 * time.Second)
			if glob.Running && !glob.Shutdown {
				_, err = io.WriteString(glob.Pipe, "/time\n")
				if err != nil {
					noresponsecount = noresponsecount + 1
					if ( noresponsecount == 30 ) {
						Session.ChannelMessageSend(support.Config.FactorioChannelID,"Server has not responded for 30 seconds...")
					}
					if ( noresponsecount == 60 )  {
						noresponsecount = 0
						Session.ChannelMessageSend(support.Config.FactorioChannelID, "Server was unresponsive for 60 seconds... restarting it.")
						//Exit, to remove zombies
						os.Exit(1)
					}
				}
			} else if !glob.Running && !glob.Shutdown {
				cmd := exec.Command(support.Config.Executable, support.Config.LaunchParameters...)
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				cmd.Stderr = os.Stderr
				cmd.Stdout = mwriter
				glob.Pipe, err = cmd.StdinPipe()
				defer cmd.Wait() //Zombie Fix

				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to execute cmd.StdinPipe()\nDetails: %s", time.Now(), err))
				}

				err := cmd.Start()

				if err != nil {
					support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to start the server\nDetails: %s", time.Now(), err))
				}
				//This is okay, because if server doesn't respond it will be auto-rebooted.
				glob.Running = true
				glob.Gametime = "na"
				glob.Sav_timer = time.Now()
				noresponsecount = 0
				Session.ChannelMessageSend(support.Config.FactorioChannelID,"Bot online, server booting...")

			}
		}
	}()

	go func() {
		time.Sleep(1 * time.Second)
		Console := bufio.NewReader(os.Stdin)
		for {
			line, _, err := Console.ReadLine()
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to read the input to pass as input to the console\nDetails: %s", time.Now(), err))
			}
			_, err = io.WriteString(glob.Pipe, fmt.Sprintf("%s\n", line))
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass input to the console\nDetails: %s", time.Now(), err))
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	go func() {
		time.Sleep(60 * time.Second)
		for {
			support.CacheDiscordMembers(Session)
			time.Sleep(15 * time.Minute)
		}
	}()

	go func() {
		for {
			time.Sleep(1 * time.Second)

			// Look for signal files
			if _, err := os.Stat(".upgrade"); !os.IsNotExist(err) {
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".upgrade"); err != nil {
					fmt.Println(".upgrade disappeared?")
				}
				if glob.Running {
					go func() {
						Session.ChannelMessageSend(support.Config.FactorioChannelID, "Updating Factorio!")
						io.WriteString(glob.Pipe, "[color=1,1,0]Factorio is shutting down in 30 seconds for a version upgrade![/color]\n")
						io.WriteString(glob.Pipe, "[color=1,0,0]Factorio is shutting down in 30 seconds for a version upgrade!![/color]\n")
						io.WriteString(glob.Pipe, "[color=0,1,1]Factorio is shutting down in 30 seconds for a version upgrade!!![/color]\n")
					        time.Sleep(30 * time.Second)
						io.WriteString(glob.Pipe, "/quit\n")
					}()
				}

				glob.Shutdown = true

			} else if _, err := os.Stat(".start"); !os.IsNotExist(err) {
				noresponsecount = 0

				if err := os.Remove(".start"); err != nil {
					fmt.Println(".start file disappeared?")
				}
				if !glob.Running {
					if err := os.Remove(".offline"); err != nil {
						fmt.Println(".offline missing...")
					}
					glob.Shutdown = false
					Session.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio starting!")
				}
			} else if _, err := os.Stat(".restart"); !os.IsNotExist(err) {
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".restart"); err != nil {
					fmt.Println(".restart file disappeared?")
				}
				if glob.Running {
					go func() {
						Session.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio restarting!")
						io.WriteString(glob.Pipe, "[color=1,1,0]Server restarting in 30 seconds.[/color]\n")
						io.WriteString(glob.Pipe, "[color=1,0,0]Server restarting in 30 seconds..[/color]\n")
						io.WriteString(glob.Pipe, "[color=0,1,1]Server restarting in 30 seconds...[/color]\n")

					        time.Sleep(30 * time.Second)
						io.WriteString(glob.Pipe, "/quit\n")
					}()
				}

				glob.Shutdown = false
			} else if _, err := os.Stat(".qrestart"); !os.IsNotExist(err) {
				noresponsecount = 0
				glob.Shutdown = true

				if err := os.Remove(".qrestart"); err != nil {
					fmt.Println(".qrestart file disappeared?")
				}
				if glob.Running {
					go func() {
					        Session.ChannelMessageSend(support.Config.FactorioChannelID, "Quick restarting!")
						io.WriteString(glob.Pipe, "[color=1,1,0]Server quick restarting.[/color]\n")
						io.WriteString(glob.Pipe, "[color=1,0,1]Server quick restarting..[/color]\n")
						io.WriteString(glob.Pipe, "[color=0,1,1]Server quick restarting...[/color]\n")
						time.Sleep(5 * time.Second)
						io.WriteString(glob.Pipe, "/quit\n")
					}()
				}
				glob.Shutdown = false
			} else if _, err := os.Stat(".shutdown"); !os.IsNotExist(err) {
				noresponsecount = 0
				if err := os.Remove(".shutdown"); err != nil {
					fmt.Println(".shutdown disappeared?")
				}
				if glob.Running {
					glob.Shutdown = true
					go func() {
						Session.ChannelMessageSend(support.Config.FactorioChannelID, "Factorio is shutting down for maintenance!")
						io.WriteString(glob.Pipe, "[color=1,1,0]Factorio is shutting down in 30 seconds for system maintenance![/color]\n")
						io.WriteString(glob.Pipe, "[color=1,0,0]Factorio is shutting down in 30 seconds for system maintenance!![/color]\n")
						io.WriteString(glob.Pipe, "[color=0,1,1]Factorio is shutting down in 30 seconds for system maintenance!!![/color]\n")
					        time.Sleep(30 * time.Second)
						io.WriteString(glob.Pipe, "/quit\n")
					}()
				}

				glob.Shutdown = true
			}
		}



	}()
	discord()
}

func discord() {
	noresponsecount = 0

	// No hard coding the token }:<
	discordToken := support.Config.DiscordToken
	commands.RegisterCommands()
	fmt.Println("Starting bot...")

	//Delete old signal files
	if err := os.Remove(".restart"); err != nil {
		fmt.Println(".restart not found... ", err)
	}
	if err := os.Remove(".qrestart"); err != nil {
		fmt.Println(".qrestart not found... ", err)
	}
	if err := os.Remove(".shutdown"); err != nil {
		fmt.Println(".shutdown not found... ", err)
	}
	if err := os.Remove(".upgrade"); err != nil {
		fmt.Println(".upgrade not found... ", err)
	}

	bot, err := discordgo.New("Bot " + discordToken)
	Session = bot
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to create the Discord session\nDetails: %s", time.Now(), err))
		return
	}

	err = bot.Open()

	if err != nil {
		fmt.Println("error opening connection,", err)
		support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to connect to Discord\nDetails: %s", time.Now(), err))
		return
	}

	bot.AddHandler(messageCreate)
	bot.AddHandlerOnce(support.Chat)
	bot.UpdateStatus(0, support.Config.GameName)
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	//Session.ChannelMessageSend(support.Config.FactorioChannelID,"Bot online, server booting...")
	glob.Sav_timer = time.Now()
	noresponsecount = 0

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	io.WriteString(glob.Pipe, "Server killed.\n")
	io.WriteString(glob.Pipe, "Server killed..\n")
	io.WriteString(glob.Pipe, "Server killed...\n")
	io.WriteString(glob.Pipe, "/quit\n")
	noresponsecount = 0
	glob.Shutdown = true
	Session.ChannelMessageSend(support.Config.FactorioChannelID, "Service killed, shutting down.")
	//Wait for save to finish if possible, 30 seconds max
	for x := 0; x < 30 && glob.Running; x++ {
		time.Sleep ( 100 * time.Millisecond )
	}
	// Cleanly close down the Discord session.
	bot.Close()
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
			if ( len(command) >= 2 ) {
				myarg = command[1]
				glob.CharName = myarg
			}
			commands.RunCommand(name, s, m)
			return
		}
		if !strings.Contains(strings.ToLower(m.Content),"!clear" ) {
			_, err := io.WriteString(glob.Pipe, fmt.Sprintf("[color=0,1,1][DISCORD-CHAT][/color] [color=1,1,0]%s:[/color] [color=0,1,1]%s[/color]\n", m.Author.Username, m.ContentWithMentionsReplaced()))
			if err != nil {
				support.ErrorLog(fmt.Errorf("%s: An error occurred when attempting to pass Discord chat to in-game\nDetails: %s", time.Now(), err))
			}
		}
		return
	}
}
