package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"
)

func main() {
	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	glob.DoRegisterCommands = flag.Bool("regCommands", false, "Register discord commands")
	glob.DoDeregisterCommands = flag.Bool("deregCommands", false, "Deregister discord commands")
	glob.LocalTestMode = flag.Bool("localTest", false, "Turn off public/auth mode for testing")
	glob.NoAutoLaunch = flag.Bool("noAutoLaunch", false, "Turn off auto-launch")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
	}

	/* Mark uptime start */
	glob.Uptime = time.Now().Round(time.Second)

	/* Start cw logs */
	cwlog.StartCWLog()
	cwlog.DoLogCW("\n Starting ChatWire Version: " + constants.Version)

	/* Handle lock file */
	bstr, err := ioutil.ReadFile("cw.lock")
	if err == nil {
		lastTimeStr := strings.TrimSpace(string(bstr))
		lastTime, err := time.Parse(time.RFC3339Nano, lastTimeStr)
		if err != nil {
			cwlog.DoLogCW("Unable to parse cw.lock: " + err.Error())
		} else {
			cwlog.DoLogCW("Lockfile found, last run was " + glob.Uptime.Sub(lastTime).String())

			/* Recent lockfile, probable crash loop */
			if lastTime.Sub(glob.Uptime) < (constants.RestartLimitMinutes * time.Minute) {
				msg := fmt.Sprintf("Recent lockfile found, possible crash. Sleeping for %v minutes.", constants.RestartLimitSleepMinutes)

				cwlog.DoLogCW(msg)
				go func(msg string) {
					for i := 0; i < constants.RestartLimitSleepMinutes*60*10 && glob.ServerRunning; i++ {
						if disc.DS == nil {
							time.Sleep(time.Millisecond * 100)
						}
					}
					disc.SmartWriteDiscord(cfg.Local.Channel.ChatChannel, msg)
				}(msg)

				time.Sleep(constants.RestartLimitMinutes * time.Minute)
				_ = os.Remove("cw.lock")
				cwlog.DoLogCW("Sleep done, exiting.")
				return
			}
		}
	}

	/* Make lockfile */
	lfile, err := os.OpenFile("cw.lock", os.O_CREATE, 0666)
	if err != nil {
		cwlog.DoLogCW("Couldn't create lock file!!!")
		return
		/* Okay, somthing is probably wrong */
	}
	lfile.Close()
	buf := fmt.Sprintf("%v\n", time.Now().Round(time.Second).Format(time.RFC3339Nano))
	err = ioutil.WriteFile("cw.lock", []byte(buf), 0644)
	if err != nil {
		cwlog.DoLogCW("Couldn't write lock file!!!")
		return
		/* Okay, somthing is probably wrong */
	}

	/* Create our maps */
	glob.AlphaValue = make(map[string]int)
	glob.ChatterList = make(map[string]time.Time)
	glob.ChatterSpamScore = make(map[string]int)
	glob.PlayerList = make(map[string]*glob.PlayerData)
	glob.PassList = make(map[string]*glob.PassData)
	glob.PlayerSus = make(map[string]int)

	pos := 10000
	for i := 'a'; i <= 'z'; i++ {
		glob.AlphaValue[string(i)] = pos
		pos++
	}

	for i := 'a'; i <= 'z'; i++ {
		for j := 'a'; j <= 'z'; j++ {
			glob.AlphaValue[string(i)+string(j)] = pos
			pos++
		}
	}

	/* Set up vote cooldown */
	now := time.Now()
	then := now.Add(time.Duration(-constants.MapCooldownMins+1) * time.Minute)
	glob.VoteBox.LastMapChange = then.Round(time.Second)

	/* Blank game time */
	fact.Gametime = (constants.Unknown)

	/* Read global and local configs, then write them back if they read correctly. */
	if cfg.ReadGCfg() {
		cfg.WriteGCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		return
	}
	if cfg.ReadLCfg() {
		cfg.WriteLCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		return
	}

	/* Setup cron */
	fact.SetupSchedule()

	/* Read in player list */
	fact.LoadPlayers()

	/* Read in cached discord role data */
	disc.ReadRoleList()
	banlist.ReadBanFile()

	/* Load old votes */
	fact.ReadVotes()

	/* Start game log */
	cwlog.StartGameLog()

	/* Main loop */
	go support.MainLoops()

	/* Loop to read Factorio stdout, runs in a goroutine */
	go support.HandleChat()

	/* Start Discord bot, don't wait for it.
	 * We want Factorio online even if Discord is down. */
	go startbot()

	if cfg.Local.Options.AutoStart {
		fact.FactAutoStart = true
	}

	/* Wait here for process signals */
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	//Bypass for faster shutdown
	commands.ClearCommands()

	_ = os.Remove("cw.lock")
	fact.FactAutoStart = false
	glob.DoRebootCW = false
	fact.QueueReload = false
	fact.QuitFactorio("Server quitting...")
	fact.WaitFactQuit()
	pprof.StopCPUProfile()
	fact.DoExit(false)
}

var DiscordConnectAttempts int

func startbot() {

	if cfg.Global.Discord.Token == "" {
		cwlog.DoLogCW("Discord token not set, not starting.")
		return
	}

	cwlog.DoLogCW("Starting Discord bot...")
	bot, erra := discordgo.New("Bot " + cfg.Global.Discord.Token)

	if erra != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", erra))
		time.Sleep(time.Minute * (5 * constants.MaxDiscordAttempts))
		DiscordConnectAttempts++

		if DiscordConnectAttempts < constants.MaxDiscordAttempts {
			startbot()
		}
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	bot.AddHandler(BotReady)
	errb := bot.Open()

	if errb != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", errb))
		time.Sleep(time.Minute * (5 * constants.MaxDiscordAttempts))
		DiscordConnectAttempts++

		if DiscordConnectAttempts < constants.MaxDiscordAttempts {
			startbot()
		}
		return
	}

	bot.LogLevel = discordgo.LogWarning
}

func BotReady(s *discordgo.Session, r *discordgo.Ready) {

	botstatus := cfg.Global.Paths.URLs.Domain
	errc := s.UpdateGameStatus(0, botstatus)
	if errc != nil {
		cwlog.DoLogCW(errc.Error())
	}

	go commands.RegisterCommands(s)
	s.AddHandler(MessageCreate)
	s.AddHandler(commands.SlashCommand)

	if s != nil {
		/* Save Discord descriptor here */
		disc.DS = s
	}

	fact.UpdateChannelName()
	fact.DoUpdateChannelName()

	cwlog.DoLogCW("Discord bot ready.")

	if cfg.Local.Channel.ChatChannel == "" {
		cwlog.DoLogCW("No chat channel set, attempting to creating one.")
		chname := fmt.Sprintf("%v-%v", cfg.Local.Callsign, cfg.Local.Name)
		channelid, err := s.GuildChannelCreate(cfg.Global.Discord.Guild, chname, discordgo.ChannelTypeGuildText)
		if err != nil {
			cwlog.DoLogCW(fmt.Sprintf("Couldn't create chat channel: %v", err))
			return
		} else if channelid != nil {
			cwlog.DoLogCW("Created chat channel.")
			cfg.Local.Channel.ChatChannel = channelid.ID
			cfg.WriteLCfg()
		}
		return
	}
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID {
		return
	}

	/* Protect players from dumb mistakes with registration codes, even on other maps */
	if support.ProtectIdiots(m.Content) {
		/* If they manage to post it into chat in Factorio on a different server,
		the message will be seen in discord but not factorio... eh whatever it still gets invalidated */
		buf := "You are supposed to type that into Factorio, not Discord... Invalidating code. Please read the directions more carefully..."
		_, err := disc.DS.ChannelMessageSend(m.ChannelID, buf)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}

	if m.Author.Bot {
		return
	}

	/* Command handling
	 * Factorio channel ONLY */
	if strings.EqualFold(cfg.Local.Channel.ChatChannel, m.ChannelID) && cfg.Local.Channel.ChatChannel != "" {
		input, _ := m.ContentWithMoreMentionsReplaced(s)
		ctext := sclean.StripControlAndSubSpecial(input)

		/* Chat message handling
		 *  Don't bother if Factorio isn't running... */
		if fact.FactorioBooted && fact.FactIsRunning {
			cwlog.DoLogCW("[" + m.Author.Username + "] " + ctext)

			alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

			cmess := sclean.StripControlAndSubSpecial(ctext)
			cmess = sclean.RemoveDiscordMarkdown(cmess)
			dname := disc.GetFactorioNameFromDiscordID(m.Author.ID)
			nbuf := ""

			/* Name to lowercase */
			dnamelower := strings.ToLower(dname)
			fnamelower := strings.ToLower(m.Author.Username)

			/* Reduce names to letters only */
			dnamereduced := alphafilter.ReplaceAllString(dnamelower, "")
			fnamereduced := alphafilter.ReplaceAllString(fnamelower, "")

			go func(factname string) {
				fact.UpdateSeen(factname)
			}(dname)

			/* Filter names... */
			corduser := sclean.StripControlAndSubSpecial(m.Author.Username)
			cordnick := sclean.StripControlAndSubSpecial(m.Member.Nick)
			factuser := sclean.StripControlAndSubSpecial(dname)

			corduserlen := len(corduser)
			cordnicklen := len(cordnick)

			cordname := corduser

			/* On short names, try nickname... if not add number, if no name... discordID */
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

			/* Cap name length */
			cordname = sclean.TruncateString(cordname, 64)
			factuser = sclean.TruncateString(factuser, 64)

			/* Check if Discord name contains Factorio name, if not lets show both their names */
			if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {

				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] @%s (%s):[/color] %s", cordname, factuser, cmess)
			} else {
				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] %s:[/color] %s", cordname, cmess)
			}

			fact.FactChat(nbuf)

		}
	}
}
