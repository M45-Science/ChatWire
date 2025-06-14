package main

import (
	"flag"
	"fmt"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"sync"
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
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modupdate"
	"ChatWire/panel"
	"ChatWire/support"
	"ChatWire/util"
)

var (
	discordConnectAttempts int
	messageHandlerLock     sync.Mutex
	botReadyAdded          bool
	hooksAdded             bool
)

func main() {
	glob.DoRegisterCommands = flag.Bool("regCommands", false, "Register discord commands")
	glob.DoDeregisterCommands = flag.Bool("deregCommands", false, "Deregister discord commands and quit.")
	glob.LocalTestMode = flag.Bool("localTest", false, "Disable public/auth mode for testing")
	glob.NoAutoLaunch = flag.Bool("noAutoLaunch", false, "Disable auto-launch")
	glob.NoDiscord = flag.Bool("noDiscord", false, "Disable Discord")
	glob.PanelFlag = flag.Bool("panel", false, "Enable web panel")
	cleanDB := flag.Bool("cleanDB", false, "Clean/minimize player database and exit.")
	cleanBans := flag.Bool("cleanBans", false, "Clean/minimize player database, along with bans and exit.")
	glob.ProxyURL = flag.String("proxy", "", "http caching proxy url. Request format: proxy/http://example.doamin/path")
	flag.Parse()

	/* Start cw logs */
	cwlog.StartCWLog()
	cwlog.StartAuditLog()
	cwlog.AutoRotateLogs()
	cwlog.DoLogCW("\n Starting %v Version: %v", constants.ProgName, constants.Version)

	initTime()
	if !*glob.LocalTestMode {
		checkLockFile()
	}
	initMaps()
	readConfigs()
	modupdate.ReadModHistory()

	if *cleanDB || *cleanBans {
		fact.LoadPlayers(false, true, *cleanBans)
		fact.WritePlayers()
		fmt.Println("Database cleaned.")
		_ = os.Remove("cw.lock")
		return
	}

	/* Start Discord bot, don't wait for it.
	 * We want Factorio online even if Discord is down. */
	if !*glob.NoDiscord {
		go startbotA()
	}

	fact.LoadPlayers(true, false, false)
	disc.ReadRoleList()
	banlist.ReadBanFile(true)
	fact.ReadVotes()
	cwlog.StartGameLog()
	if *glob.PanelFlag {
		panel.Start()
	}
	if !*glob.NoDiscord {
		go support.MainLoops()
		go support.HandleChat()
	}

	//If autolaunch is off, get current factorio version
	if cfg.Local.Options.AutoStart && !*glob.NoAutoLaunch {
		fact.SetAutolaunch(true, false)
	} else if *glob.NoAutoLaunch {
		info := &factUpdater.InfoData{Xreleases: cfg.Local.Options.ExpUpdates, Build: "headless", Distro: "linux64"}
		factUpdater.GetFactorioVersion(info)
		fact.FactorioVersion = info.VersInt.IntToString()
		cwlog.DoLogCW("Factorio version: " + fact.FactorioVersion)
	}

	/* Wait here for process signals */
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	_ = os.Remove("cw.lock")
	/*
		fact.SetAutolaunch(false, false)
		glob.DoRebootCW = false
		fact.QueueReboot = false
		fact.QueueFactReboot = false
		fact.QuitFactorio("Server quitting...")
		fact.WaitFactQuit(false)

		fact.DoExit(false)
	*/
}

func startbotA() {

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
		cwlog.DoLogCW("An error occurred when attempting to CREATE the Discord session. Details: %v", erra)
		time.Sleep(time.Duration(discordConnectAttempts*5) * time.Second)
		discordConnectAttempts++

		if discordConnectAttempts < constants.MaxDiscordAttempts {
			go startbotA()
		}
		return
	}
	startbotB(bot)
}

func startbotB(bot *discordgo.Session) {
	/* We need a few intents to detect discord users and roles */
	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	/* This is called when the connection is verified */
	if !botReadyAdded {
		botReadyAdded = true
		bot.AddHandler(botReady)
	}
	errb := bot.Open()

	/* This handles error after the initial connection */
	if errb != nil {
		cwlog.DoLogCW("An error occurred when attempting to OPEN the Discord session. Details: %v", errb)
		time.Sleep(time.Duration(discordConnectAttempts*5) * time.Second)
		discordConnectAttempts++

		if discordConnectAttempts < constants.MaxDiscordAttempts {
			go startbotB(bot)
		}
		return
	}

	/* This drastically reduces log spam */
	bot.LogLevel = discordgo.LogWarning
}

func botReady(s *discordgo.Session, r *discordgo.Ready) {
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
	go func() {
		commands.DeregisterCommands()
		commands.RegisterCommands()
	}()

	/* Message and command hooks */
	if !hooksAdded {
		hooksAdded = true
		s.AddHandler(handleDiscordMessages)
		s.AddHandler(commands.SlashCommand)
	}

	go func() {
		/* Update the string for the channel name and topic */
		fact.UpdateChannelName()
		/* Send the new string to discord */
		fact.DoUpdateChannelName()
	}()

	cwlog.DoLogCW("Discord bot ready.")

	if cfg.Local.Channel.ChatChannel == "" || cfg.Local.Channel.ChatChannel == "MY DISCORD CHANNEL ID" {
		cwlog.DoLogCW("No chat channel set, attempting to creating one.")
		chname := fmt.Sprintf("%v-%v", cfg.Local.Callsign, cfg.Local.Name)
		channelid, err := s.GuildChannelCreate(cfg.Global.Discord.Guild, chname, discordgo.ChannelTypeGuildText)
		if err != nil {
			cwlog.DoLogCW("Couldn't create chat channel: %v", err)
			return
		} else if channelid != nil {
			cwlog.DoLogCW("Created chat channel.")
			cfg.Local.Channel.ChatChannel = channelid.ID
			cfg.WriteLCfg()
		}
		return
	}

	//Only on first connect
	if !support.BotIsReady {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", constants.ProgName+" "+constants.Version+" is now online.", glob.COLOR_GREEN)
		if fact.FactIsRunning || fact.FactorioBooted {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Ready", "Factorio "+fact.FactorioVersion+" is online.", glob.COLOR_GREEN)
		}
	}

	//Reset attempt count, we are fully connected.
	discordConnectAttempts = 0
	support.BotIsReady = true
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

				fact.LogGameCMS(false, cfg.Local.Channel.ChatChannel, fmt.Sprintf("Recent lockfile found, possible crash. Sleeping for %v minutes.", constants.RestartLimitSleepMinutes))

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
	glob.PanelTokens = make(map[string]*glob.PanelTokenData)

	/* Generate number to alpha map, used for auto port assignment starting at constants.RconPortOffset */
	pos := constants.RconPortOffset

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

	if cfg.ReadGCfg() {
		//cfg.WriteGCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
	if cfg.ReadLCfg() {
		util.SetTempFilePrefix(cfg.Local.Callsign + "-")
		//cfg.WriteLCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
}
