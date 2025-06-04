package fact

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/hako/durafmt"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
	"ChatWire/sclean"
)

const (
       MaxZipSize = 1024 * 1024 * 1024 //1gb
)

var (
	lastBanName     string
	banLock         sync.Mutex
	AutoLaunchLock  sync.Mutex
	FactRunningLock sync.Mutex
)

func GetFactUPS() (float64, float64, float64) {
	TickHistoryLock.Lock()
	var tenMin []TickInt
	var thirtyMin []TickInt
	var oneHour []TickInt

	tickHistoryLen := len(TickHistory) - 1
	var tenMinAvr, thirtyMinAvr, oneHourAvr float64
	if tickHistoryLen > 0 {
		end := TickHistory[tickHistoryLen]
		endInt := float64(end.Day*86400.0 + end.Hour*3600.0 + end.Min*60.0 + end.Sec)

		if tickHistoryLen >= 600 {
			tenMin = TickHistory[tickHistoryLen-600 : tickHistoryLen]

			for _, item := range tenMin {
				tenMinAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}
		if tickHistoryLen >= 1800 {
			thirtyMin = TickHistory[tickHistoryLen-1800.0 : tickHistoryLen]

			for _, item := range thirtyMin {
				thirtyMinAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}
		if tickHistoryLen >= 3600 {
			oneHour = TickHistory[tickHistoryLen-3600 : tickHistoryLen]

			for _, item := range oneHour {
				oneHourAvr += float64(endInt) - float64(item.Day*86400.0+item.Hour*3600.0+item.Min*60.0+item.Sec)
			}
		}

		tenMinAvr = tenMinAvr / 180300.0 * 60.0
		thirtyMinAvr = thirtyMinAvr / 1620900.0 * 60.0
		oneHourAvr = oneHourAvr / 6481800.0 * 60.0
	}
	TickHistoryLock.Unlock()

	return (tenMinAvr), (thirtyMinAvr), (oneHourAvr)
}

func WriteBanBy(name, reason, banBy string) {
	tNow := time.Now()
	banTimeFormat := "01-02-2006"

	outString := fmt.Sprintf("%v %v -- %v %v %v", name, reason, banBy, tNow.Format(banTimeFormat), cfg.GetGameLogURL())
	doBan(name, outString)
}

func WriteBan(name, reason string) {
	outString := fmt.Sprintf("%v %v", name, reason)
	doBan(name, outString)
}

func doBan(name, outString string) {
	banLock.Lock()
	defer banLock.Unlock()

	if name == lastBanName {
		return
	}

	lastBanName = name
	WriteFact("/ban " + outString)
	WriteFact("/purge " + name)
}

func WriteUnban(name string) {
	banLock.Lock()
	defer banLock.Unlock()

	if strings.EqualFold(lastBanName, name) {
		lastBanName = ""
	}
	WriteFact("/unban " + name)
}

func SetLastBan(name string) {
	banLock.Lock()
	defer banLock.Unlock()

	lastBanName = name
}

func CheckSave(path, name string, showError bool) (good bool, folder string) {
	zip, err := zip.OpenReader(path + "/" + name)
	if err != nil || zip == nil {
		buf := fmt.Sprintf("Save '%v' is not a valid zip file: '%v', trying next save.", name, err.Error())
		if showError {
			CMS(cfg.Local.Channel.ChatChannel, buf)
		}
		cwlog.DoLogCW(buf)
	} else {
		defer zip.Close()
		for _, file := range zip.File {
			fc, err := file.Open()

			if err != nil {

				buf := fmt.Sprintf("Save '%v' is corrupted or invalid: '%v'.", name, err.Error())
				if showError {
					CMS(cfg.Local.Channel.ChatChannel, buf)
				}
				cwlog.DoLogCW(buf)
				break
			} else {
				defer fc.Close()
				if strings.HasSuffix(file.Name, "level.dat0") {
					content, err := io.ReadAll(fc)
					if len(content) > constants.LevelDatMinSize && err == nil {
						return true, filepath.Dir(file.Name)
					} else {
						return false, ""
					}
				}
			}
		}
		buf := fmt.Sprintf("Save '%v' did not contain any save data.", name)
		if showError {
			CMS(cfg.Local.Channel.ChatChannel, buf)
		}
		cwlog.DoLogCW(buf)
	}

	return false, ""
}

func SetAutolaunch(autolaunch, report bool) {
	AutoLaunchLock.Lock()
	defer AutoLaunchLock.Unlock()

	if !autolaunch && FactAutoStart {
		FactAutoStart = false
		if report {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Notice", "Auto-reboot has been turned OFF.", glob.COLOR_GREEN)
		}
		cwlog.DoLogCW("Autolaunch disabled.")
	} else if autolaunch && !FactAutoStart {
		FactAutoStart = true
		cwlog.DoLogCW("Autolaunch enabled.")
		if report {
			glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Notice", "Auto-reboot has been ENABLED.", glob.COLOR_GREEN)
		}
	}

}

func SetFactRunning(run, report bool) {
	FactRunningLock.Lock()
	defer FactRunningLock.Unlock()

	wasrun := FactIsRunning
	FactIsRunning = run

	if run && glob.NoResponseCount >= 15 && !FactorioBootedAt.IsZero() && time.Since(FactorioBootedAt) > time.Minute {
		//CMS(cfg.Local.Channel.ChatChannel, "Server now appears to be responding again.")
		cwlog.DoLogCW("Server now appears to be responding again.")
	}
	glob.NoResponseCount = 0

	if wasrun != run {
		if !run {
			FactorioBooted = false
			FactorioBootedAt = time.Time{}
		}
		if report {
			if run {
				cwlog.DoLogGame("Factorio " + FactorioVersion + " is now online.")
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Ready", "Factorio "+FactorioVersion+" is now online.", glob.COLOR_GREEN)
			} else {
				cwlog.DoLogCW("Factorio has closed.")
				glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Offline", "Factorio is now offline.", glob.COLOR_RED)
			}
		}
		UpdateChannelName()
		return
	}
}

func GetGuildName() string {
	if disc.Guild == nil {
		return constants.Unknown
	} else {
		return disc.Guildname
	}
}

/* Whitelist a specific player. */
func WhitelistPlayer(pname string, level int) {
	if FactorioBooted && FactIsRunning {
		if cfg.Local.Options.CustomWhitelist {
			return
		}
		if cfg.Local.Options.MembersOnly {
			if level > 0 {
				WriteFact("/whitelist add %s", pname)
			}
		}
		if cfg.Local.Options.RegularsOnly {
			if level > 1 {
				WriteFact("/whitelist add %s", pname)
			}
		}
	}
}

/* Write a adminlist for a server, before it boots */
func WriteAdminlist() int {

	wpath := cfg.GetFactorioFolder() +
		constants.AdminlistName

	glob.PlayerListLock.RLock()

	var count = 0
	var buf = "[\n"

	//Add admins
	for _, player := range glob.PlayerList {
		if player.Level >= 254 {
			/* Add admins to whitelist for custom whitelists */
			if cfg.Local.Options.CustomWhitelist {
				WriteFact("/whitelist add %s", player.Name)
			}
			buf = buf + "\"" + player.Name + "\",\n"
			count = count + 1
		}
	}

	if count > 1 {
		lchar := len(buf)
		buf = buf[0 : lchar-2]
	}
	buf = buf + "\n]\n"
	glob.PlayerListLock.RUnlock()

	err := os.WriteFile(wpath, []byte(buf), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteAdminlist: WriteFile failure")
		return -1
	}
	return count
}

/* Write a full whitelist for a server, before it boots */
func WriteWhitelist() int {

	wpath := cfg.GetFactorioFolder() +
		constants.WhitelistName

	if cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly {
		glob.PlayerListLock.RLock()

		var buf = "[\n"
		var localPlayerList []*glob.PlayerData
		localPlayerList = make([]*glob.PlayerData, len(localPlayerList))

		for _, player := range glob.PlayerList {
			if cfg.Local.Options.RegularsOnly {
				if player.Level > 1 {
					localPlayerList = append(localPlayerList, player)
				}
			} else {
				if player.Level > 0 {
					localPlayerList = append(localPlayerList, player)
				}
			}
		}

		//Sort by last seen
		sort.Slice(localPlayerList, func(i, j int) bool {
			return localPlayerList[i].LastSeen < localPlayerList[j].LastSeen
		})

		l := len(localPlayerList) - 1
		var count = 0

		//Add admins
		for x := l; x > 0; x-- {
			var player = localPlayerList[x]
			if player.Level >= 255 {
				buf = buf + "\"" + player.Name + "\",\n"
				count = count + 1
			}
		}

		//Add veterans
		for x := l; x > 0; x-- {
			var player = localPlayerList[x]
			if player.Level == 3 {
				buf = buf + "\"" + player.Name + "\",\n"
				count = count + 1
			}
		}

		//Everyone else
		for x := l; x > 0; x-- {
			if count >= constants.MaxWhitelist {
				break
			}
			var player = localPlayerList[x]
			if player.Level < 3 {
				buf = buf + "\"" + player.Name + "\",\n"
				count = count + 1
			}
		}

		if count > 1 {
			lchar := len(buf)
			buf = buf[0 : lchar-2]
		}
		buf = buf + "\n]\n"
		glob.PlayerListLock.RUnlock()

		err := os.WriteFile(wpath, []byte(buf), 0644)

		if err != nil {
			cwlog.DoLogCW("WriteWhitelist: WriteFile failure")
			return -1
		}
		return count
	} else {
		_ = os.Remove(wpath)
	}

	return 0
}

/* Quit Factorio */
func QuitFactorio(message string) {
	if FactIsRunning {
		glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Notice", "Quitting Factorio: "+message, glob.COLOR_ORANGE)
	}

	if message == "" {
		message = "Server quitting."
	}

	glob.RelaunchThrottle = 0
	glob.NoResponseCount = 0

	/* Running but no players, just quit */
	if (FactorioBooted && FactIsRunning) && NumPlayers <= 0 {
		cwlog.DoLogCW("Quitting Factorio...")
		cwlog.DoLogGame("Quitting Factorio...")
		WriteFact("/quit")
		WaitFactQuit(false)

		/* Running, but players connected... Give them quick feedback. */
	} else if FactorioBooted && FactIsRunning && NumPlayers > 0 {
		FactChat("[color=red]" + message + "[/color]")
		FactChat("[color=green]" + message + "[/color]")
		FactChat("[color=blue]" + message + "[/color]")
		FactChat("[color=white]" + message + "[/color]")
		FactChat("[color=black]" + message + "[/color]")
		time.Sleep(time.Second * 3)

		cwlog.DoLogCW("Quitting Factorio...")
		cwlog.DoLogGame("Quitting Factorio...")
		WriteFact("/quit")
		WaitFactQuit(false)
	}

	if (FactorioBooted || FactIsRunning) && glob.FactorioCmd != nil {
		cwlog.DoLogCW("Factorio still has not closed, sending interrupt.")
		glob.FactorioCmd.Process.Signal(os.Interrupt)
		glob.FactorioCmd.Wait()
		SetFactRunning(false, false)
	}
}

/* Send a string to Factorio, via stdin */
func WriteFact(format string, args ...interface{}) {

	var input string
	if args == nil {
		input = format
	} else {
		input = fmt.Sprintf(format, args...)
	}

	PipeLock.Lock()
	defer PipeLock.Unlock()

	/* Clean string */
	buf := sclean.UnicodeCleanup(input)

	gpipe := Pipe
	if gpipe != nil {

		plen := len(buf)

               if plen > constants.MaxDiscordMsgLen {
			cwlog.DoLogCW("Message to Factorio, too long... Not sending.")
			return
		} else if plen <= 1 {
			cwlog.DoLogCW("Message for Factorio too short... Not sending.")
			return
		}

		_, err := io.WriteString(gpipe, buf+"\n")
		if err != nil {
			cwlog.DoLogCW("An error occurred when attempting to write to Factorio.\nError: %v Input: %v", err, input)
			SetFactRunning(false, true)
			if glob.FactorioCancel != nil {
				glob.FactorioCancel()
			}
			return
		}
		if buf != "/time" && !strings.HasPrefix(buf, "/cchat") && !strings.HasPrefix(buf, "/cwhisper") &&
			!strings.HasPrefix(buf, "/online") && !strings.HasPrefix(buf, "/p o c") {
			cwlog.DoLogCW("CW: %v", buf)
		}

	} else {
		//cwlog.DoLogCW("An error occurred when attempting to write to Factorio (nil pipe)")
		SetFactRunning(false, true)
		return
	}
}

func LevelToString(level int) string {

	name := "Invalid"

	if level <= -254 {
		name = "Deleted"
	} else if level == -1 {
		name = "Banned"
	} else if level == 0 {
		name = "New"
	} else if level == 1 {
		name = "Member"
	} else if level == 2 {
		name = "Regular"
	} else if level == 3 {
		name = "Veteran"
	} else if level >= 255 {
		name = "Moderator"
	}

	return name
}

func StringToLevel(in string) int {

	input := strings.ToLower(in)
	level := 0

	if strings.HasPrefix(input, "new") {
		level = 0
	} else if strings.HasPrefix(input, "members") {
		level = 1
	} else if strings.HasPrefix(input, "regulars") {
		level = 2
	} else if strings.HasPrefix(input, "veterans") {
		level = 3
	} else if strings.HasPrefix(input, "banished") {
		level = 0
	} else if strings.HasPrefix(input, "moderator") {
		level = 255
	}

	return level
}

/* Promote a player to the level they have, in Factorio and on Discord */
func AutoPromote(pname string, bootMode bool, doBan bool) string {
	playerName := " *(New Player)* "

	if pname != "" {
		plevel := PlayerLevelGet(pname, false)

		if !bootMode {
			if plevel <= -254 {
				playerName = " **(Deleted Player)** "

			} else if plevel == -1 {
				playerName = " **(Banned)**"

				if doBan {
					name := strings.ToLower(pname)
					glob.PlayerListLock.Lock()
					if glob.PlayerList[name] != nil {
						WriteBan(name, glob.PlayerList[name].BanReason)
					}
					glob.PlayerListLock.Unlock()
				}

			} else if plevel == 1 {
				playerName = " *(Member)*"
				WriteFact("/member %s", pname)

			} else if plevel == 2 {
				playerName = " *(Regular)*"

				WriteFact("/regular %s", pname)
			} else if plevel == 3 {
				playerName = " *(Veteran)*"

				WriteFact("/veteran %s", pname)
			} else if plevel == 255 {
				playerName = " *(Moderator)*"

				WriteFact("/promote %s", pname)
			}
		}

		discid := disc.GetDiscordIDFromFactorioName(pname)
		factname := disc.GetFactorioNameFromDiscordID(discid)

		if strings.EqualFold(factname, pname) {

			newrole := ""
			if plevel == 0 {
				newrole = cfg.Global.Discord.Roles.New
			} else if plevel == 1 {
				newrole = cfg.Global.Discord.Roles.Member
			} else if plevel == 2 {
				newrole = cfg.Global.Discord.Roles.Regular
			} else if plevel == 3 {
				newrole = cfg.Global.Discord.Roles.Veteran
			} else if plevel == 255 {
				newrole = cfg.Global.Discord.Roles.Moderator
			}

			if discid != "" {
				if disc.Guild != nil {

					errrole, regrole := disc.RoleExists(disc.Guild, newrole)

					if !errrole {
						cwlog.DoLogCW("Couldn't find role %v.", newrole)
					} else {
						errset := disc.SmartRoleAdd(cfg.Global.Discord.Guild, discid, regrole.ID)
						if errset != nil {
							cwlog.DoLogCW("Couldn't set role %v for %v.", newrole, discid)
						}
					}
				}
			}
		}
	}

	return playerName

}

/* Update our channel name, but don't send it yet */
func UpdateChannelName() {

	var newchname string
	nump := NumPlayers
	icon := "⚪"

	if cfg.Local.Options.CustomWhitelist {
		icon = "🔴"
	}
	if cfg.Local.Options.MembersOnly {
		icon = "🟢"
	}
	if cfg.Local.Options.RegularsOnly {
		icon = "🟠"
	}
	if nump == 0 {
		icon = "⚫"
	}

	_, _, hourUPSAvr := GetFactUPS()
	if hourUPSAvr > 1 && math.Round(hourUPSAvr) <= 57 {
		icon = "⬜"
		if cfg.Local.Options.CustomWhitelist {
			icon = "🟥"
		}
		if cfg.Local.Options.MembersOnly {
			icon = "🟩"
		}
		if cfg.Local.Options.RegularsOnly {
			icon = "🟧"
		}
		if nump == 0 {
			icon = "⬛"
		}
	}

	if nump == 0 {
		newchname = fmt.Sprintf("%v%v", icon, cfg.Local.Callsign+"-"+cfg.Local.Name)
	} else {
		newchname = fmt.Sprintf("%v%v%v", nump, icon, cfg.Local.Callsign+"-"+cfg.Local.Name)
	}

	disc.UpdateChannelLock.Lock()
	disc.NewChanName = newchname
	disc.UpdateChannelLock.Unlock()

}

var oldTopic string
var chPos int

/* When appropriate, actually update the channel name */
func DoUpdateChannelName() {

	var aerr error
	if disc.DS == nil {
		return
	}

	disc.UpdateChannelLock.Lock()
	chname := disc.NewChanName
	oldchname := disc.OldChanName
	disc.UpdateChannelLock.Unlock()

	URL, found := MakeSteamURL()
	var newTopic string

	if HasResetTime() {
		mpre := "MAP RESET"
		newTopic = fmt.Sprintf("%v: <t:%v:F>(LOCAL)", mpre, cfg.Local.Options.NextReset.UTC().Unix())
	}
	if found {
		newTopic = newTopic + ", CONNECT: " + URL
	}

	if (chname != oldchname || oldTopic != newTopic) &&
		cfg.Local.Channel.ChatChannel != "" &&
		cfg.Local.Channel.ChatChannel != "MY DISCORD CHANNEL ID" {
		disc.UpdateChannelLock.Lock()
		disc.OldChanName = disc.NewChanName
		disc.UpdateChannelLock.Unlock()

		ch, err := disc.DS.Channel(cfg.Local.Channel.ChatChannel)
		if err != nil {
			cwlog.DoLogCW("Unable to get chat channel information.")
			return
		}

		chPos = ch.Position
		_, aerr = disc.DS.ChannelEditComplex(cfg.Local.Channel.ChatChannel, &discordgo.ChannelEdit{Name: chname, Position: &chPos, Topic: newTopic})

		if aerr != nil {
			cwlog.DoLogCW("An error occurred when attempting to rename the Factorio discord channel. Details: %s", aerr)
			return
		} else {
			oldTopic = newTopic
		}
	}
}

func ShowMapList(i *discordgo.InteractionCreate, voteMode bool) {
	if disc.DS == nil {
		return
	}

	path := cfg.GetSavesFolder()

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read saves directory.")
		return
	}

	step := 1
	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files, and our map-change temp file.
		if strings.HasPrefix(f.Name(), "_autosave") && strings.HasSuffix(f.Name(), ".zip") && !strings.HasSuffix(f.Name(), "tmp.zip") && !strings.HasSuffix(f.Name(), cfg.Local.Name+"_new.zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		iInfo, _ := tempf[i].Info()
		jInfo, _ := tempf[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())

		//return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	var availableMaps []discordgo.SelectMenuOption

	numFiles := len(tempf)
	//Subtract num of static objects
	if numFiles > constants.MaxMapResults-2 {
		numFiles = constants.MaxMapResults - 2
	}

	availableMaps = append(availableMaps,
		discordgo.SelectMenuOption{

			Label:       "NEW-MAP",
			Description: "Archive the current map, and generate a new one.",
			Value:       "NEW-MAP",
			Emoji: &discordgo.ComponentEmoji{
				Name: "⭐",
			},
		},
	)

	if HasResetInterval() && HasResetTime() {
		if !cfg.Local.Options.SkipReset {
			availableMaps = append(availableMaps,
				discordgo.SelectMenuOption{

					Label:       "SKIP-RESET",
					Description: "Skip the next map reset.",
					Value:       "SKIP-RESET",
					Emoji: &discordgo.ComponentEmoji{
						Name: "❇️",
					},
				},
			)
		} else {
			availableMaps = append(availableMaps,
				discordgo.SelectMenuOption{

					Label:       "SKIP-RESET",
					Description: "ALREADY SKIPPED!",
					Value:       "ALREADY-SKIPPED",
					Emoji: &discordgo.ComponentEmoji{
						Name: "🚫",
					},
				},
			)
		}
	}

	for i := 0; i < numFiles; i++ {

		f := tempf[i]
		fName := f.Name()

		if strings.HasSuffix(fName, ".zip") {
			saveName := strings.TrimSuffix(fName, ".zip")
			step++

			units, err := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,μs:μs")
			if err != nil {
				panic(err)
			}

			/* Get mod date */
			info, _ := f.Info()
			modDate := time.Since(info.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(2).Format(units) + " ago"

			availableMaps = append(availableMaps,
				discordgo.SelectMenuOption{

					Label:       saveName,
					Description: modStr,
					Value:       saveName,
					Emoji: &discordgo.ComponentEmoji{
						Name: "💾",
					},
				},
			)
		}
	}

	if numFiles <= 0 {
		disc.InteractionEphemeralResponse(i, "Error:", "No saves were found.")
	} else {

		var response *discordgo.InteractionResponse
		if voteMode {
			response = &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Vote for 'new-map', 'skip-reset' or a specific save-game. (two votes needed):",
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									// Select menu, as other components, must have a customID, so we set it to this value.
									CustomID:    "VoteMap",
									Placeholder: "Select one",
									Options:     availableMaps,
								},
							},
						},
					},
				},
			}
		} else {
			response = &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Change Map:",
					Flags:   discordgo.MessageFlagsEphemeral,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{
							Components: []discordgo.MessageComponent{
								discordgo.SelectMenu{
									// Select menu, as other components, must have a customID, so we set it to this value.
									CustomID:    "ChangeMap",
									Placeholder: "Choose a save",
									Options:     availableMaps,
								},
							},
						},
					},
				},
			}
		}
		err := disc.DS.InteractionRespond(i.Interaction, response)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}
}

func ShowFullMapList(i *discordgo.InteractionCreate) {
	if disc.DS == nil {
		return
	}

	path := cfg.GetSavesFolder()

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.InteractionEphemeralResponse(i, "Error:", "Unable to read saves directory.")
	}

	step := 1
	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files, and our map-change temp file.
		if strings.HasSuffix(f.Name(), ".zip") && !strings.HasSuffix(f.Name(), "tmp.zip") && !strings.HasSuffix(f.Name(), cfg.Local.Name+"_new.zip") {
			tempf = append(tempf, f)
		}
	}

	sort.Slice(tempf, func(i, j int) bool {
		iInfo, _ := tempf[i].Info()
		jInfo, _ := tempf[j].Info()
		return iInfo.ModTime().After(jInfo.ModTime())
		//return tempf[i].ModTime().After(tempf[j].ModTime())
	})

	mapList := ""
	numFiles := len(tempf)
	if numFiles > constants.MaxFullMapResults {
		numFiles = constants.MaxFullMapResults
	}

	for i := 0; i < numFiles; i++ {

		f := tempf[i]
		fName := f.Name()

		if strings.HasSuffix(fName, ".zip") {
			saveName := strings.TrimSuffix(fName, ".zip")
			saveName = strings.TrimPrefix(saveName, "_autosave")
			step++

			units, err := durafmt.DefaultUnitsCoder.Decode("y:y,w:w,d:d,h:h,m:m,s:s,ms:ms,μs:μs")
			if err != nil {
				panic(err)
			}

			/* Get mod date */
			info, _ := f.Info()
			modDate := time.Since(info.ModTime())
			modDate = modDate.Round(time.Second)
			modStr := durafmt.Parse(modDate).LimitFirstN(2).Format(units)

			sepStr := ", "
			if i%2 == 0 {
				sepStr = "\n"
			}
			tempStr := fmt.Sprintf("%v: %v%v", saveName, modStr, sepStr)
			mapListLen := len(mapList)
			tempStrLen := len(tempStr)

			if mapListLen+tempStrLen > 4080 {
				mapList = mapList + "...(cut, max)"
				break
			} else {
				mapList = mapList + tempStr
			}
		}
	}

	if numFiles <= 0 {
		disc.InteractionEphemeralResponse(i, "Error:", "No saves were found.")
	} else {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Full map list", Description: mapList})

		respData := &discordgo.InteractionResponseData{Embeds: elist, Flags: discordgo.MessageFlagsEphemeral}
		resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
		err := disc.DS.InteractionRespond(i.Interaction, resp)
		if err != nil {
			return
		}
	}
}

func DoChangeMap(arg string) {

	if strings.EqualFold(arg, "new-map") {
		Map_reset(false)

		return
	}
	if strings.EqualFold(arg, "skip-reset") {
		if HasResetInterval() && HasResetTime() {
			if !cfg.Local.Options.SkipReset {
				cfg.Local.Options.SkipReset = true
				cfg.WriteLCfg()
				LogGameCMS(true, cfg.Local.Channel.ChatChannel, "❇️ NOTICE: The upcoming map reset has been skipped.")
				AdvanceReset()
			}
		}
		return
	}
	if strings.EqualFold(arg, "already-skipped") {
		return
	}

	path := cfg.GetSavesFolder()

	/* Check if file is valid and found */
	saveStr := fmt.Sprintf("%v.zip", arg)
	good, _ := CheckSave(path, saveStr, false)
	if !good {
		msg := "DoChangeMap: Attempted to load an invalid save."
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		FactChat(msg)
		return
	}

	SetAutolaunch(false, false)
	QuitFactorio("Server rebooting for map vote!")
	WaitFactQuit(false)
	selSaveName := path + "/" + saveStr
	from, erra := os.Open(selSaveName)
	if erra != nil {
		msg := "An error occurred when attempting to open the selected save."
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		FactChat(msg)
		return
	}
	defer from.Close()

	newmappath := path + "/" + cfg.Local.Name + "_new.zip"
	_, err := os.Stat(newmappath)
	if !os.IsNotExist(err) {
		err = os.Remove(newmappath)
		if err != nil {
			msg := "An error occurred when attempting to open the selected save."
			LogCMS(cfg.Local.Channel.ChatChannel, msg)
			FactChat(msg)
			return
		}
	}
	to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
	if errb != nil {
		msg := "An error occurred when attempting to create the save file. "
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		FactChat(msg)
		return
	}
	defer to.Close()

	_, errc := io.Copy(to, from)
	if errc != nil {
		msg := "An error occurred when attempting to write the save file."
		LogCMS(cfg.Local.Channel.ChatChannel, msg)
		FactChat(msg)
		return
	}

	msg := fmt.Sprintf("Loading save: %v", arg)
	cwlog.DoLogGame(msg)
	glob.RelaunchThrottle = 0
	SetAutolaunch(true, false)

}

func FileHasZipBomb(path string) bool {
	//Open file
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	return BytesHasZipBomb(data)
}

func BytesHasZipBomb(data []byte) bool {
	// Create a reader from the byte array
	byteReader := bytes.NewReader(data)

	// Create a zip reader
	zipReader, err := zip.NewReader(byteReader, int64(len(data)))
	if err != nil {
		return false
	}

	var totalSize uint64

	for _, file := range zipReader.File {
		size := file.UncompressedSize64
		if size > MaxZipSize {
			return true
		}
		totalSize += size
	}
	return totalSize > MaxZipSize
}

func IsPlayerOnline(who string) bool {

	if len(who) <= 0 {
		return false
	}

	OnlinePlayersLock.RLock()
	defer OnlinePlayersLock.RUnlock()

	for _, p := range glob.OnlinePlayers {
		if strings.EqualFold(p.Name, who) {
			return true
		}
	}

	return false
}

/* Send chat to factorio */
func FactChat(format string, args ...interface{}) {

	var input string
	if args == nil {
		input = format
	} else {
		input = fmt.Sprintf(format, args...)
	}

	if input == "" {
		return
	}

	/* Limit length, Discord does this... but just in case */
	input = sclean.TruncateStringEllipsis(input, 2048)

	/*
		* If we are running our softmod, use the much safer
		 * /cchat command
	*/
	if glob.SoftModVersion != constants.Unknown {
		WriteFact("/cchat " + input)
	} else {
		/*
		 * Just in case there is no soft-mod,
		 * filter out potential threats
		 */
		input = sclean.UnicodeCleanup(input)
		input = sclean.RemoveFactorioTags(input)
		input = sclean.RemoveDiscordMarkdown(input)

		/* Attempt to prevent anyone from running a command. */
		strlen := len(input)
		for z := 0; z < strlen; z++ {
			input = strings.TrimLeft(input, " ")
			input = strings.TrimLeft(input, "/")
			input = strings.TrimRight(input, " ")
		}
		WriteFact(input)
	}
}

/* Send chat to factorio */
func FactWhisper(player, format string, args ...interface{}) {

	var input string
	if args == nil {
		input = format
	} else {
		input = fmt.Sprintf(format, args...)
	}

	/* Limit length, Discord does this... but just in case */
	input = sclean.TruncateStringEllipsis(input, 2048)

	/*
		* If we are running our softmod, use the much safer
		 * /cwhisper command
	*/
	if glob.SoftModVersion != constants.Unknown {
		WriteFact("/cwhisper %v %v", player, input)
	} else {
		/*
		 * Just in case there is no soft-mod,
		 * filter out potential threats
		 */
		input = sclean.UnicodeCleanup(input)
		input = sclean.RemoveFactorioTags(input)
		input = sclean.RemoveDiscordMarkdown(input)

		/* Attempt to prevent anyone from running a command. */
		strlen := len(input)
		for z := 0; z < strlen; z++ {
			input = strings.TrimLeft(input, " ")
			input = strings.TrimLeft(input, "/")
			input = strings.TrimRight(input, " ")
		}
		WriteFact("/whisper %v %v", player, input)
	}
}

func WaitFactQuit(waiting bool) {
	if waiting {
		for FactIsRunning && FactorioBooted && glob.ServerRunning {
			time.Sleep(time.Millisecond * 100)
		}
	} else {
		for x := 0; x < constants.MaxFactorioCloseWait && FactIsRunning && FactorioBooted && glob.ServerRunning; x++ {
			time.Sleep(time.Millisecond * 100)
		}
	}
}

/* Auto generates a steam connect URL */
func MakeSteamURL() (string, bool) {
	if cfg.Global.Paths.URLs.Domain != "localhost" && cfg.Global.Paths.URLs.Domain != "" {
		buf := fmt.Sprintf("https://go-game.net/gosteam/427520.--mp-connect%%20%v:%v", cfg.Global.Paths.URLs.Domain, cfg.Local.Port)
		return buf, true
	} else {
		return "(not configured)", false
	}
}

/* Program shutdown */
func DoExit(delay bool) {

	glob.BootMessage = disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.BootMessage, "Status", constants.ProgName+" shutting down.", glob.COLOR_RED)

	//Wait a few seconds for CMS to finish
	for i := 0; i < 300; i++ {
		if len(disc.CMSBuffer) > 0 {
			time.Sleep(time.Millisecond * 100)
		} else {
			break
		}
	}

	time.Sleep(time.Second * 2)

	/* This kills all loops! */
	glob.ServerRunning = false

	cwlog.DoLogCW("CW closing, load/save db.")
	WritePlayers()

	/* File locks */
	glob.PlayerListWriteLock.Lock()

	cwlog.DoLogCW("Closing log files.")
	glob.GameLogDesc.Close()
	glob.CWLogDesc.Close()

	_ = os.Remove("cw.lock")
	/* Logs are closed, don't report */

	os.Remove("log/newest.log")
	fmt.Println("Goodbye.")
	if delay {
		time.Sleep(constants.ErrorDelayShutdown * time.Second)
	}

	if disc.DS != nil {
		disc.DS.Close()
	}
	os.Exit(1)
}

func AddFactColor(color string, text string) string {
	text = sclean.UnicodeCleanup(text)
	text = sclean.RemoveDiscordMarkdown(text)
	text = sclean.RemoveFactorioTags(text)

	color = sclean.UnicodeCleanup(color)
	color = sclean.RemoveFactorioTags(color)
	color = strings.TrimSpace(color)
	color = strings.ToLower(color)

	return fmt.Sprintf("[color=%v]%v[/color]", color, text)
}

/* Generate full path to Factorio binary */
func GetFactorioBinary() string {
	bloc := ""
	if strings.HasPrefix(cfg.Global.Paths.Binaries.FactBinary, "/") {
		/* Absolute path */
		bloc = cfg.Global.Paths.Binaries.FactBinary
	} else {
		/* Relative path */
		bloc = cfg.Global.Paths.Folders.ServersRoot +
			cfg.Global.Paths.ChatWirePrefix +
			cfg.Local.Callsign + "/" +
			cfg.Global.Paths.Folders.FactorioDir + "/" +
			cfg.Global.Paths.Binaries.FactBinary
	}
	return bloc
}

func GetUpdateCachePath() string {
	return cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" + "UpdateCache/"
}

/* Write a Discord message to the buffer */
func CMS(channel string, text string) {

       text = sclean.TruncateStringEllipsis(text, constants.MaxDiscordMsgLen)
	/* Split at newlines, so we can batch neatly */
	lines := strings.Split(text, "\n")

	disc.CMSBufferLock.Lock()

	for _, line := range lines {
		var item disc.CMSBuf
		item.Channel = channel
		item.Text = line

		disc.CMSBuffer = append(disc.CMSBuffer, item)
	}

	disc.CMSBufferLock.Unlock()
}

/* Log AND send this message to Discord */
func LogCMS(channel string, text string) {
	cwlog.DoLogCW(text)
	CMS(channel, text)
}

/* Log AND send this message to Discord  and Factorio chat */
func LogGameCMS(chat bool, channel string, text string) {
	cwlog.DoLogGame(text)
	CMS(channel, text)
	if chat {
		FactChat(text)
	}
}
