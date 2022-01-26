package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strings"
	"time"

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

	"github.com/bwmarrin/discordgo"
)

func main() {
	/* Randomize starting color */
	var src = rand.NewSource(time.Now().UnixNano())
	var r = rand.New(src)
	glob.LastColor = r.Intn(constants.NumColors - 1)

	/* Set up rewind cooldown */
	now := time.Now()
	then := now.Add(time.Duration(-constants.RewindCooldownMinutes+1) * time.Minute)
	glob.VoteBox.LastRewindTime = then.Round(time.Second)

	/* Create our maps */
	playlist := make(map[string]*glob.PlayerData)
	passlist := make(map[string]*glob.PassData)

	/* Assign to globals */
	glob.PlayerList = playlist
	glob.PassList = passlist

	/* Blank game time */
	fact.SetGameTime(constants.Unknown)
	/* Mark uptime start */
	glob.Uptime = time.Now().Round(time.Second)

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

	/* Start logs */
	cwlog.StartCWLog()
	cwlog.StartGameLog()
	cwlog.DoLogCW("Version: " + constants.Version)

	/* Read in cached discord role data */
	disc.ReadRoleList()
	banlist.ReadBanFile()

	/* Set autostart mode from config */
	if cfg.Local.AutoStart {
		fact.SetAutoStart(true)
	}

	/* Start Discord bot, don't wait for it.
	 * We want Factorio online even if Discord is down. */
	go startbot()
	/* Loop to read Factorio stdout, runs in a goroutine */
	support.Chat()

	/* Load player database and max-online records */
	fact.LoadPlayers()
	fact.LoadRecord()

	/* Load old votes */
	fact.ReadRewindVotes()

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
					for disc.DS == nil {
						time.Sleep(time.Second)
					}
					disc.SmartWriteDiscord(cfg.Local.ChannelData.ChatID, msg)
				}(msg)

				_ = os.Remove("cw.lock")
				time.Sleep(constants.RestartLimitMinutes * time.Minute)
				cwlog.DoLogCW("Sleep done, exiting.")
				return
			}
		}
	}

	/* If lockfile found, we are already running or crashed */
	if err := os.Remove("cw.lock"); err == nil {
		//return
		/* Proceed anyway, process is managed by systemd */
	} else {
		if !os.IsNotExist(err) {
			cwlog.DoLogCW("Unable to delete old lockfile, exiting.")
			return
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
	ioutil.WriteFile("cw.lock", []byte(buf), 0644)

	/* All threads/loops in here.
	 * If autostart is enabled, we will boot Factorio. */
	support.MainLoops()
}

func startbot() {

	cwlog.DoLogCW("Starting bot...")

	bot, erra := discordgo.New("Bot " + cfg.Global.DiscordData.Token)

	if erra != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the Discord session. Details: %s", erra))
		time.Sleep(time.Minute)
		startbot()
		return
	}

	bot.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsAllWithoutPrivileged | discordgo.IntentsGuildPresences | discordgo.IntentsGuildMembers)

	errb := bot.Open()

	if errb != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to connect to Discord. Details: %s", errb))
		time.Sleep(time.Minute)
		startbot()
		return
	}

	if bot != nil && erra == nil && errb == nil {
		/* Save Discord descriptor here */
		disc.DS = bot
	}

	bot.LogLevel = discordgo.LogWarning

	time.Sleep(2 * time.Second)
	commands.RegisterCommands()
	bot.AddHandler(MessageCreate)
	botstatus := fmt.Sprintf("%vhelp", cfg.Global.DiscordCommandPrefix)
	errc := bot.UpdateGameStatus(0, botstatus)
	if errc != nil {
		cwlog.DoLogCW(errc.Error())
	}

	bstring := "Loading: CW *v" + constants.Version + "*"
	cwlog.DoLogCW(bstring)
	//fact.CMS(cfg.Local.ChannelData.ChatID, bstring)
	fact.UpdateChannelName()
}

func MessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	/* Command handling
	 * Factorio channel ONLY */
	if m.ChannelID == cfg.Local.ChannelData.ChatID && m.ChannelID != "" {
		input, _ := m.ContentWithMoreMentionsReplaced(s)
		ctext := sclean.StripControlAndSubSpecial(input)
		cwlog.DoLogCW("[" + m.Author.Username + "] " + ctext)

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

		/* Chat message handling
		 *  Don't bother if Factorio isn't running... */
		if fact.IsFactorioBooted() {
			/* block mee6 commands */
			if !strings.HasPrefix(ctext, "!") {

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

					nbuf = fmt.Sprintf("/cchat [color=0,1,1][Discord][/color] [color=1,1,0]@%s[/color] [color=0,0.5,0](%s):[/color] %s%s[/color]", cordname, factuser, fact.RandomColor(false), cmess)
				} else {
					nbuf = fmt.Sprintf("/cchat [color=0,1,1][Discord][/color] [color=1,1,0]@%s:[/color] %s%s[/color]", cordname, fact.RandomColor(false), cmess)
				}

				fact.WriteFact(nbuf)
			}
		}
	}
}
