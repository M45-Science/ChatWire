package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/banlist"
	"ChatWire/cfg"
	"ChatWire/commands"
	"ChatWire/commands/moderator"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"
	"ChatWire/support"
)

func main() {
	glob.DoRegisterCommands = flag.Bool("regCommands", false, "Register discord commands")
	glob.DoDeregisterCommands = flag.Bool("deregCommands", false, "Deregister discord commands and quit.")
	glob.LocalTestMode = flag.Bool("localTest", false, "Turn off public/auth mode for testing")
	glob.NoAutoLaunch = flag.Bool("noAutoLaunch", false, "Turn off auto-launch")
	cleanDB := flag.Bool("cleanDB", false, "Clean/minimize player database and exit.")
	flag.Parse()

	/* Start cw logs */
	cwlog.StartCWLog()
	cwlog.DoLogCW("\n Starting ChatWire Version: " + constants.Version)

	if *cleanDB {
		fact.LoadPlayers(true, true)
		fact.WritePlayers()
		fmt.Println("Database cleaned.")
		_ = os.Remove("cw.lock")
		return
	}

	initTime()
	if !*glob.LocalTestMode {
		checkLockFile()
	}
	initMaps()
	readConfigs()
	moderator.MakeFTPFolders()

	/* Start Discord bot, don't wait for it.
	 * We want Factorio online even if Discord is down. */
	go startbot()

	fact.SetupSchedule()
	fact.LoadPlayers(true, false)
	disc.ReadRoleList()
	banlist.ReadBanFile()
	fact.ReadVotes()
	cwlog.StartGameLog()
	go support.MainLoops()
	go support.HandleChat()

	if cfg.Local.Options.AutoStart {
		fact.FactAutoStart = true
	}

	/* Wait here for process signals */
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	commands.ClearCommands()

	_ = os.Remove("cw.lock")
	fact.FactAutoStart = false
	glob.DoRebootCW = false
	fact.QueueReload = false
	fact.QuitFactorio("Server quitting...")
	fact.WaitFactQuit()
	fact.DoExit(false)
}

var DiscordConnectAttempts int

func startbot() {

	/* Check if Discord token is set */
	if cfg.Global.Discord.Token == "" {
		cwlog.DoLogCW("Discord token not set, not starting.")
		return
	}

	/* Attempt to start bot */
	cwlog.DoLogCW("Starting Discord bot...")
	bot, erra := discordgo.New("Bot " + cfg.Global.Discord.Token)

	/*
	 * If we fail, keep attempting with increasing delay and maximum tries
	 * We do this, in case there is a failure.
	 * Discord will invalidate the token if there are too many connection attempts.
	 */
	if erra != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", erra))
		time.Sleep(time.Duration(DiscordConnectAttempts*5) * time.Second)
		DiscordConnectAttempts++

		if DiscordConnectAttempts < constants.MaxDiscordAttempts {
			startbot()
		}
		return
	}

	/* We need a few intents to detect discord users and roles */
	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	/* This is called when the connection is verified */
	bot.AddHandler(BotReady)
	errb := bot.Open()

	/* This handles error after the inital connection */
	if errb != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %v", errb))
		time.Sleep(time.Duration(DiscordConnectAttempts*5) * time.Second)
		DiscordConnectAttempts++

		if DiscordConnectAttempts < constants.MaxDiscordAttempts {
			startbot()
		}
		return
	}

	/* This drastically reduces log spam */
	bot.LogLevel = discordgo.LogWarning
}

func BotReady(s *discordgo.Session, r *discordgo.Ready) {
	if s != nil {
		/* Save Discord descriptor, we need it */
		disc.DS = s
	}

	/* Set the bot's Discord status message */
	botstatus := cfg.Global.Paths.URLs.Domain
	errc := s.UpdateGameStatus(0, botstatus)
	if errc != nil {
		cwlog.DoLogCW(errc.Error())
	}

	/* Register discord slash commands */
	go commands.RegisterCommands(s)
	/* Message and command hooks */
	s.AddHandler(MessageCreate)
	s.AddHandler(commands.SlashCommand)

	/* Update the string for the channel name and topic */
	fact.UpdateChannelName()
	/* Send the new string to discord */
	fact.DoUpdateChannelName()

	cwlog.DoLogCW("Discord bot ready.")

	/* This is untested, currently */
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

	//Reset attempt count, we are fully connected.
	DiscordConnectAttempts = 0
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	/* Ignore messages from self */
	if m.Author.ID == s.State.User.ID {
		return
	}

	message, _ := m.ContentWithMoreMentionsReplaced(s)
	if len(message) > 500 {
		message = fmt.Sprintf("%s(cut, too long!)", sclean.TruncateStringEllipsis(message, 500))
	}
	message = sclean.UnicodeCleanup(message)

	/* Protect players from dumb mistakes with registration codes, even on other maps */
	/* Do this before we reject bot messages, to catch factorio chat on different maps/channels */
	if support.ProtectIdiots(message) {
		/* If they manage to post it into chat in Factorio on a different server,
		the message will be seen in discord but not factorio... eh whatever it still gets invalidated */
		buf := "You are supposed to type that into Factorio, not Discord... Invalidating code. Please read the directions more carefully..."
		_, err := s.ChannelMessageSend(m.ChannelID, buf)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}

	/* Throw away messages from bots */
	if m.Author.Bot {
		return
	}

	/* Command handling
	 * Factorio channel ONLY */
	if strings.EqualFold(cfg.Local.Channel.ChatChannel, m.ChannelID) && cfg.Local.Channel.ChatChannel != "" {

		/*
		 * Chat message handling
		 *  Don't bother if Factorio isn't running...
		 */
		if fact.FactorioBooted && fact.FactIsRunning {
			cwlog.DoLogCW("[" + m.Author.Username + "] " + message)

			/* Used for name matching */
			alphafilter, _ := regexp.Compile("[^a-zA-Z]+")

			/* Remove control characters and discord markdown */
			cmess := sclean.RemoveDiscordMarkdown(message)

			/* Try to find factorio name, for registered players */
			dname := disc.GetFactorioNameFromDiscordID(m.Author.ID)
			nbuf := ""

			/* Name to lowercase */
			dnamelower := strings.ToLower(dname)
			fnamelower := strings.ToLower(m.Author.Username)

			/* Reduce names to letters only */
			dnamereduced := alphafilter.ReplaceAllString(dnamelower, "")
			fnamereduced := alphafilter.ReplaceAllString(fnamelower, "")

			/* Mark as seen, async */
			go func(factname string) {
				fact.UpdateSeen(factname)
			}(dname)

			/* Filter names... */
			corduser := sclean.UnicodeCleanup(m.Author.Username)
			cordnick := sclean.UnicodeCleanup(m.Member.Nick)
			factuser := sclean.UnicodeCleanup(dname)

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
					cordname = fnamereduced
				} else {
					cordname = fmt.Sprintf("ID#%s", m.Author.ID)
				}
			}

			/* Cap name length for safety/annoyance */
			cordname = sclean.TruncateString(cordname, 64)
			factuser = sclean.TruncateString(factuser, 64)

			/* Check if Discord name contains Factorio name, if not lets show both their names */
			if dname != "" && !strings.Contains(dnamereduced, fnamereduced) && !strings.Contains(fnamereduced, dnamereduced) {

				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] @%s (%s):[/color] %s", cordname, factuser, cmess)
			} else {
				nbuf = fmt.Sprintf("[color=0,0.5,1][Discord] %s:[/color] %s", cordname, cmess)
			}

			/* Send the final text to factorio */
			fact.FactChat(nbuf)

		}
	}
}

func checkLockFile() {
	/* Handle lock file */
	bstr, err := os.ReadFile("cw.lock")
	if err == nil {
		lastTimeStr := strings.TrimSpace(string(bstr))
		lastTime, err := time.Parse(time.RFC3339Nano, lastTimeStr)
		if err != nil {
			cwlog.DoLogCW("Unable to parse cw.lock: " + err.Error())
			_ = os.Remove("cw.lock")

		} else {
			cwlog.DoLogCW("Lockfile found, last run was " + glob.Uptime.Sub(lastTime).String())

			/* Recent lockfile, probable crash loop */
			if time.Since(lastTime) < (constants.RestartLimitMinutes * time.Minute) {
				msg := fmt.Sprintf("Recent lockfile found, possible crash. Sleeping for %v minutes.", constants.RestartLimitSleepMinutes)

				cwlog.DoLogCW(msg)

				time.Sleep(constants.RestartLimitMinutes * time.Minute)
				_ = os.Remove("cw.lock")
			}
		}
	}

	/* Make lockfile */
	lfile, err := os.OpenFile("cw.lock", os.O_CREATE, 0666)
	if err != nil {
		cwlog.DoLogCW("Couldn't create lock file!!!")
		os.Exit(1)
	}
	lfile.Close()
	buf := fmt.Sprintf("%v\n", time.Now().UTC().Round(time.Second).Format(time.RFC3339Nano))
	err = os.WriteFile("cw.lock", []byte(buf), 0644)
	if err != nil {
		cwlog.DoLogCW("Couldn't write lock file!!!")
		os.Exit(1)
	}
}

func initMaps() {
	/* Create our maps */
	glob.AlphaValue = make(map[string]int)
	glob.ChatterList = make(map[string]time.Time)
	glob.ChatterSpamScore = make(map[string]int)
	glob.PlayerList = make(map[string]*glob.PlayerData)
	glob.PassList = make(map[string]*glob.PassData)

	/* Generate number to alpha map, used for auto port assignment */
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

}

func initTime() {
	glob.LastSusWarning = time.Now().Add(time.Duration(-constants.SusWarningInterval) * time.Minute)
	now := time.Now()
	then := now.Add(time.Duration(-constants.MapCooldownMins+1) * time.Minute)
	glob.VoteBox.LastMapChange = then.Round(time.Second)
	fact.Gametime = (constants.Unknown)
	glob.PausedAt = time.Now()
	glob.Uptime = time.Now().UTC().Round(time.Second)
}

func readConfigs() {

	/* Read global and local configs, then write them back if they read correctly. */
	/* This cleans up formatting if manually edited, and verifies we can write the config */
	if cfg.ReadGCfg() {
		cfg.WriteGCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
	if cfg.ReadLCfg() {
		cfg.WriteLCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
}
