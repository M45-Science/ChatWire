package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"./commands"
	"./config"
	"./constants"
	"./fact"
	"./glob"
	"./logs"
	"./platform"
	"./support"

	"github.com/bwmarrin/discordgo"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	t := time.Now()

	//Randomize starting color
	var src = rand.NewSource(time.Now().UnixNano())
	var r = rand.New(src)
	glob.LastColor = r.Intn(constants.NumColors - 1)

	//Find & set map types size
	glob.MaxMapTypes = len(constants.MapTypes)

	fact.SetSaveTimer()
	fact.SetGameTime(constants.Unknown)

	glob.Uptime = time.Now()

	//Load settings
	config.Config.LoadEnv()

	//Append whitelist to launch params if we are in that mode
	if strings.ToLower(config.Config.DoWhitelist) == "yes" ||
		strings.ToLower(config.Config.DoWhitelist) == "true" {
		config.Config.LaunchParameters = append(config.Config.LaunchParameters, "--use-server-whitelist")
		glob.WhitelistMode = true
	}

	//Set autostart mode from config
	if strings.ToLower(config.Config.AutoStart) == "yes" ||
		strings.ToLower(config.Config.AutoStart) == "true" {
		fact.SetAutoStart(true)
	}

	//Saves a ton of space!
	cmdb := exec.Command("/bin/sh", config.Config.CompressScript)
	cmdb.CombinedOutput()

	//Create our log file names
	glob.GameLogName = fmt.Sprintf("log/game-%v.log", t.Unix())
	glob.BotLogName = fmt.Sprintf("log/bot-%v.log", t.Unix())

	//Make log directory
	os.MkdirAll("log", os.ModePerm)

	//Open log files
	gdesc, erra := os.OpenFile(glob.GameLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	bdesc, errb := os.OpenFile(glob.BotLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//Save descriptors, open/closed elsewhere
	glob.GameLogDesc = gdesc
	glob.BotLogDesc = bdesc

	//Send stdout and stderr to our logfile, to capture panic errors and discordgo errors
	platform.CaptureErrorOut(bdesc)

	//Handle file errors
	if erra != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to create game log. Details: %s", erra))
		fact.DoExit()
	}

	if errb != nil {
		logs.Log(fmt.Sprintf("An error occurred when attempting to create bot log. Details: %s", errb))
		fact.DoExit()
	}

	//Start discord bot, start reading stdout
	go func() {
		startbot()
	}()
	support.Chat()

	//Load player database and max online records
	fact.LoadPlayers()
	fact.LoadRecord()

	if err := os.Remove("cw.lock"); err == nil {
		logs.Log("Lockfile found, bot crashed?")
	}

	lfile, err := os.OpenFile("cw.lock", os.O_CREATE, 0666)
	if err != nil {
		logs.Log("Couldn't create lock file!!!")
	}
	lfile.Close()

	go func() {
		fact.CheckFactUpdate(true)
	}()

	//All threads/loops in here.
	support.MainLoops()
}

func startbot() {

	logs.LogWithoutEcho("Starting bot...")

	bot, erra := discordgo.New("Bot " + config.Config.DiscordToken)

	if erra != nil {
		logs.LogWithoutEcho(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %s", erra))
		time.Sleep(30 * time.Second)
		startbot()
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	errb := bot.Open()

	if errb != nil {
		logs.LogWithoutEcho(fmt.Sprintf("An error occurred when attempting to connect to Discord. Details: %s", errb))
		time.Sleep(30 * time.Second)
		startbot()
		return
	}

	if bot != nil && erra == nil && errb == nil {
		//Save discord descriptor here
		glob.DS = bot
	}

	bot.LogLevel = discordgo.LogInformational

	time.Sleep(2 * time.Second)
	commands.RegisterCommands()
	bot.AddHandler(MessageCreate)
	botstatus := fmt.Sprintf("%vhelp", config.Config.Prefix)
	bot.UpdateStatus(0, botstatus)

	logs.Log("Bot online. *v" + constants.Version + "*")
	fact.UpdateChannelName()
}
