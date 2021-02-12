package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"./cfg"
	"./commands"
	"./constants"
	"./fact"
	"./glob"
	"./logs"
	"./platform"
	"./support"

	"github.com/bwmarrin/discordgo"
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

	//Read global and local configs
	cfg.ReadGCfg()
	cfg.ReadLCfg()

	//Re-Write global and local configs
	cfg.WriteGCfg()
	cfg.WriteLCfg()

	//Set autostart mode from config
	if cfg.Local.AutoStart {
		fact.SetAutoStart(true)
	}

	//Saves a ton of space!
	cmdb := exec.Command(cfg.Global.PathData.ShellPath, cfg.Global.PathData.LogCompScriptPath)
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

	bot, erra := discordgo.New("Bot " + cfg.Global.DiscordData.Token)

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
	botstatus := fmt.Sprintf("%vhelp", cfg.Global.DiscordCommandPrefix)
	bot.UpdateGameStatus(0, botstatus)

	logs.Log("Bot online. *v" + constants.Version + "*")
	fact.UpdateChannelName()

	//Update aux channel name on reboot
	glob.DS.ChannelEditComplex(cfg.Local.ChannelData.LogID, &discordgo.ChannelEdit{Name: cfg.Local.ChannelData.Name, Position: cfg.Local.ChannelData.Pos})
}
