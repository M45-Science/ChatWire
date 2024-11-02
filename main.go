package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path/filepath"
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
	"ChatWire/support"
)

func main() {
	glob.DoRegisterCommands = flag.Bool("regCommands", false, "Register discord commands")
	glob.DoDeregisterCommands = flag.Bool("deregCommands", false, "Deregister discord commands and quit.")
	glob.LocalTestMode = flag.Bool("localTest", false, "Turn off public/auth mode for testing")
	glob.NoAutoLaunch = flag.Bool("noAutoLaunch", false, "Turn off auto-launch")
	cleanDB := flag.Bool("cleanDB", false, "Clean/minimize player database and exit.")
	cleanBans := flag.Bool("cleanBans", false, "Clean/minimize player database, along with bans and exit.")
	flag.Parse()

	/* Start cw logs */
	cwlog.StartCWLog()
	cwlog.AutoRotateLogs()
	cwlog.DoLogCW("\n Starting %v Version: %v", constants.ProgName, constants.Version)

	data, err := os.ReadFile(constants.BootMsgIDFile)
	if err == nil && data != nil {
		err = json.Unmarshal([]byte(data), &glob.BootMessage)
		if err == nil && len(glob.BootMessage.Embeds) > 0 {
			glob.BootMessage.Embeds[0].Description = glob.BootMessage.Embeds[0].Description + "\n"
		}
	}

	initTime()
	if !*glob.LocalTestMode {
		checkLockFile()
	}
	initMaps()
	readConfigs()

	if *cleanDB || *cleanBans {
		fact.LoadPlayers(false, true, *cleanBans)
		fact.WritePlayers()
		fmt.Println("Database cleaned.")
		_ = os.Remove("cw.lock")
		return
	}

	moderator.MakeFTPFolders()

	/* Start Discord bot, don't wait for it.
	 * We want Factorio online even if Discord is down. */
	go startbot()

	fact.SetupSchedule()
	fact.LoadPlayers(true, false, false)
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

	_ = os.Remove("cw.lock")
	fact.FactAutoStart = false
	glob.DoRebootCW = false
	fact.QueueReboot = false
	fact.QueueFactReboot = false
	fact.QuitFactorio("Server quitting...")
	fact.WaitFactQuit(false)

	if glob.BootMessage != nil {
		outbuf := new(bytes.Buffer)
		enc := json.NewEncoder(outbuf)
		enc.SetIndent("", "\t")

		err := enc.Encode(glob.BootMessage)
		if err == nil {
			os.WriteFile(constants.BootMsgIDFile, outbuf.Bytes(), 0644)
		}
	}

	time.Sleep(time.Second * 2)
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
		cwlog.DoLogCW("An error occurred when attempting to create the Discord session. Details: %v", erra)
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
		cwlog.DoLogCW("An error occurred when attempting to create the Discord session. Details: %v", errb)
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
	go func() {
		commands.DeregisterCommands()
		commands.RegisterCommands()
	}()

	/* Message and command hooks */
	s.AddHandler(handleDiscordMessages)
	s.AddHandler(commands.SlashCommand)

	/* Update the string for the channel name and topic */
	fact.UpdateChannelName()
	/* Send the new string to discord */
	fact.DoUpdateChannelName()

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

	//Reset attempt count, we are fully connected.
	DiscordConnectAttempts = 0
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

	if cfg.ReadGCfg() {
		//cfg.WriteGCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
	if cfg.ReadLCfg() {
		//cfg.WriteLCfg()
	} else {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
		os.Exit(1)
	}
}

func getBinaryPath() string {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Unable to executable info.")
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		log.Fatal("Unable to detect binary path.")
	}
	return filepath.Dir(exePath) + "/"
}
