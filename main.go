package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Distortions81/M45-ChatWire/cfg"
	"github.com/Distortions81/M45-ChatWire/commands"
	"github.com/Distortions81/M45-ChatWire/constants"
	"github.com/Distortions81/M45-ChatWire/disc"
	"github.com/Distortions81/M45-ChatWire/fact"
	"github.com/Distortions81/M45-ChatWire/glob"
	"github.com/Distortions81/M45-ChatWire/sclean"
	"github.com/Distortions81/M45-ChatWire/support"
	"github.com/bwmarrin/discordgo"
)

func main() {

	log.SetFlags(log.Lmicroseconds | log.Lshortfile)
	log.Println("Version: " + constants.Version)
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
	_, err := cmdb.CombinedOutput()
	if err != nil {
		log.Println(err)
	}

	//Create our log file names
	glob.GameLogName = fmt.Sprintf("log/game-%v.log", t.Unix())
	glob.BotLogName = fmt.Sprintf("log/bot-%v.log", t.Unix())

	//Make log directory
	errr := os.MkdirAll("log", os.ModePerm)
	if errr != nil {
		log.Println(errr)
	}

	//Open log files
	gdesc, erra := os.OpenFile(glob.GameLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	bdesc, errb := os.OpenFile(glob.BotLogName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	//Save descriptors, open/closed elsewhere
	glob.GameLogDesc = gdesc
	glob.BotLogDesc = bdesc

	//Send stdout and stderr to our logfile, to capture panic errors and discordgo errors
	//platform.CaptureErrorOut(bdesc)

	//Handle file errors
	if erra != nil {
		log.Println(fmt.Sprintf("An error occurred when attempting to create game log. Details: %s", erra))
		fact.DoExit()
	}

	if errb != nil {
		log.Println(fmt.Sprintf("An error occurred when attempting to create bot log. Details: %s", errb))
		fact.DoExit()
	}

	//Set bot log file
	log.SetOutput(bdesc)

	//Start discord bot, start reading stdout
	go func() {
		startbot()
	}()
	support.Chat()

	//Load player database and max online records
	fact.LoadPlayers()
	fact.LoadRecord()

	if err := os.Remove("cw.lock"); err == nil {
		log.Println("Lockfile found, bot crashed?")
	}

	lfile, err := os.OpenFile("cw.lock", os.O_CREATE, 0666)
	if err != nil {
		log.Println("Couldn't create lock file!!!")
	}
	lfile.Close()

	go func() {
		fact.CheckFactUpdate(true)
	}()

	//All threads/loops in here.
	support.MainLoops()
}

func startbot() {

	log.Println("Starting bot...")

	bot, erra := discordgo.New("Bot " + cfg.Global.DiscordData.Token)

	if erra != nil {
		log.Println(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %s", erra))
		time.Sleep(30 * time.Second)
		startbot()
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	errb := bot.Open()

	if errb != nil {
		log.Println(fmt.Sprintf("An error occurred when attempting to connect to Discord. Details: %s", errb))
		time.Sleep(30 * time.Second)
		startbot()
		return
	}

	if bot != nil && erra == nil && errb == nil {
		//Save discord descriptor here
		glob.DS = bot
	}

	bot.LogLevel = discordgo.LogWarning

	time.Sleep(2 * time.Second)
	commands.RegisterCommands()
	bot.AddHandler(MessageCreate)
	botstatus := fmt.Sprintf("%vhelp", cfg.Global.DiscordCommandPrefix)
	errc := bot.UpdateGameStatus(0, botstatus)
	if errc != nil {
		log.Println(errc)
	}

	bstring := "Loading: CW *v" + constants.Version + "*"
	log.Println(bstring)
	fact.CMS(cfg.Local.ChannelData.ChatID, bstring)
	fact.UpdateChannelName()
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	input, _ := m.ContentWithMoreMentionsReplaced(s)
	ctext := sclean.StripControlAndSubSpecial(input)
	log.Print("[" + m.Author.Username + "] " + ctext)

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	//Command stuff
	if m.ChannelID == cfg.Local.ChannelData.ChatID && m.ChannelID != "" { //Factorio channel
		if strings.HasPrefix(ctext, cfg.Global.DiscordCommandPrefix) {
			empty := []string{}

			slen := len(ctext)

			if slen > 1 {

				args := strings.Split(ctext, " ")
				arglen := len(args)

				if arglen > 0 {
					name := strings.ToLower(args[0])
					if arglen > 1 {
						commands.RunCommand(name[1:], s, m, args[1:arglen])
					} else {
						commands.RunCommand(name[1:], s, m, empty)
					}
				}
			}
			return
		}
	}

	//Block everything but chat
	if m.ChannelID != cfg.Local.ChannelData.ChatID {
		return
	}

	//Chat message handling
	if fact.IsFactorioBooted() { // Don't bother if we arne't running...
		if !strings.HasPrefix(ctext, "!") { //block mee6 commands

			alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

			cmess := sclean.StripControlAndSubSpecial(ctext)
			cmess = sclean.RemoveDiscordMarkdown(cmess)
			//cmess = sclean.RemoveFactorioTags(cmess)
			dname := disc.GetFactorioNameFromDiscordID(m.Author.ID)
			nbuf := ""

			//Name to lowercase
			dnamelower := strings.ToLower(dname)
			fnamelower := strings.ToLower(m.Author.Username)

			//Reduce names to letters only
			dnamereduced := alphafilter.ReplaceAllString(dnamelower, "")
			fnamereduced := alphafilter.ReplaceAllString(fnamelower, "")

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(dname)

			//Filter names...
			corduser := sclean.StripControlAndSubSpecial(m.Author.Username)
			cordnick := sclean.StripControlAndSubSpecial(m.Member.Nick)
			factuser := sclean.StripControlAndSubSpecial(dname)

			corduserlen := len(corduser)
			cordnicklen := len(cordnick)

			cordname := corduser

			//On short names, try nickname... if not add number, if no name... discordID
			if corduserlen < 5 {
				if cordnicklen >= 4 && cordnicklen < 18 {
					cordname = cordnick
				}
				cordnamelen := len(cordname)
				if cordnamelen > 0 {
					cordname = fmt.Sprintf("%s#%s", fnamereduced, m.Author.Discriminator)
				} else {
					cordname = fmt.Sprintf("ID#%s", m.Author.ID)
				}
			}

			//Cap name length
			cordname = sclean.TruncateString(cordname, 25)
			factuser = sclean.TruncateString(factuser, 25)

			//Check if discord name contains factorio name, if not lets show both their names
			if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {

				nbuf = fmt.Sprintf("/cchat [color=0,1,1][DISCORD][/color] [color=1,1,0]@%s[/color] [color=0,0.5,0](%s):[/color] %s%s[/color]", cordname, factuser, fact.RandomColor(false), cmess)
			} else {
				nbuf = fmt.Sprintf("/cchat [color=0,1,1][DISCORD][/color] [color=1,1,0]@%s:[/color] %s%s[/color]", cordname, fact.RandomColor(false), cmess)
			}

			fact.WriteFact(nbuf)
		}
	}
}
