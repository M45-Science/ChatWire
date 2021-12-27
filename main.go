package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"ChatWire/botlog"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"

	"github.com/bwmarrin/discordgo"
)

func main() {
	botlog.StartBotLog()
	botlog.StartGameLog()

	//Saves a ton of space!
	cmdb := exec.Command(cfg.Global.PathData.ShellPath, cfg.Global.PathData.LogCompScriptPath)
	_, err := cmdb.CombinedOutput()
	if err != nil {
		botlog.DoLog(err.Error())
	}

	botlog.DoLog("Version: " + constants.Version)
	//Randomize starting color
	var src = rand.NewSource(time.Now().UnixNano())
	var r = rand.New(src)
	glob.LastColor = r.Intn(constants.NumColors - 1)

	//Create our maps
	playlist := make(map[string]*glob.PlayerData)
	passlist := make(map[string]*glob.PassData)

	glob.PlayerList = playlist
	glob.PassList = passlist

	//Find & set map types size
	glob.MaxMapTypes = len(constants.MapTypes)

	fact.SetGameTime(constants.Unknown)

	glob.Uptime = time.Now()

	//Read global and local configs, then write them back if they read correctly.
	if cfg.ReadGCfg() {
		cfg.WriteGCfg()
	} else {
		botlog.DoLog("ReadGCfg failed")
		return
	}
	if cfg.ReadLCfg() {
		cfg.WriteLCfg()
	} else {
		botlog.DoLog("ReadLCfg failed")
		return
	}

	//Set autostart mode from config
	if cfg.Local.AutoStart {
		fact.SetAutoStart(true)
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
		botlog.DoLog("Lockfile found, bot crashed?")
	}

	lfile, err := os.OpenFile("cw.lock", os.O_CREATE, 0666)
	if err != nil {
		botlog.DoLog("Couldn't create lock file!!!")
	}
	lfile.Close()

	go func() {
		fact.CheckFactUpdate(true)
	}()

	//All threads/loops in here.
	support.MainLoops()
}

func startbot() {

	botlog.DoLog("Starting bot...")

	bot, erra := discordgo.New("Bot " + cfg.Global.DiscordData.Token)

	if erra != nil {
		botlog.DoLog(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %s", erra))
		time.Sleep(30 * time.Second)
		startbot()
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	errb := bot.Open()

	if errb != nil {
		botlog.DoLog(fmt.Sprintf("An error occurred when attempting to connect to Discord. Details: %s", errb))
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
		botlog.DoLog(errc.Error())
	}

	bstring := "Loading: CW *v" + constants.Version + "*"
	botlog.DoLog(bstring)
	fact.CMS(cfg.Local.ChannelData.ChatID, bstring)
	fact.UpdateChannelName()
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	//Command stuff
	if m.ChannelID == cfg.Local.ChannelData.ChatID && m.ChannelID != "" { //Factorio channel
		input, _ := m.ContentWithMoreMentionsReplaced(s)
		ctext := sclean.StripControlAndSubSpecial(input)
		botlog.DoLog("[" + m.Author.Username + "] " + ctext)

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
}
