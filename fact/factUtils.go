package fact

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

type infoJSONData struct {
	Name             string
	Version          string
	Title            string
	Dependencies     []string
	Description      string
	Factorio_version string
	Homepage         string
}

type modData struct {
	Name    string
	Enabled bool
}

type modListData struct {
	Mods []modData
}

func readInfoJson(content []byte) (*infoJSONData, bool) {

	info := &infoJSONData{}
	err := json.Unmarshal(content, &info)
	if err != nil {
		cwlog.DoLogCW("readInfoJson: Unmarshal failure")
		return nil, false
	}
	return info, true
}

func readMod(filename string) (*infoJSONData, bool) {
	read, err := zip.OpenReader(filename)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return nil, false
	}

	for _, file := range read.File {
		fc, err := file.Open()
		if err != nil {
			cwlog.DoLogCW(err.Error())
			return nil, false
		}
		fileName := filepath.Base(file.Name)
		if fileName == "info.json" {
			content, err := io.ReadAll(fc)
			if err != nil {
				cwlog.DoLogCW(err.Error())
				return nil, false
			}
			if len(content) > 0 {
				return readInfoJson(content)
			} else {
				return nil, false
			}
		}
	}

	return nil, false
}

func readModList(path string) (*modListData, bool) {

	content, err := os.ReadFile(path + "mod-list.json")
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return nil, false
	}

	modList := &modListData{}
	err = json.Unmarshal(content, &modList)
	if err != nil {
		cwlog.DoLogCW("readModList: Unmarshal failure")
		return nil, false
	}

	return modList, true
}

func searchModlist(modList *modListData, modName string) (bool, bool) {
	if modList != nil && modList.Mods != nil && modName != "" {
		for _, mod := range modList.Mods {
			if mod.Name == modName {
				return true, true
			}
		}
		return false, true
	}

	return false, false
}

func checkModLoaded(modName string) bool {
	for _, mod := range ModList {
		if mod == modName {
			return true
		}
	}
	return false
}
func MakeModList() (string, error) {

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Mods + "/"

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW(err.Error())
		return "Error", err
	}

	/* Loop all files */
	var tempf []fs.DirEntry
	for _, f := range files {
		//Hide non-zip files, temp files
		if strings.HasSuffix(f.Name(), ".zip") {
			tempf = append(tempf, f)
		}
	}

	modList, _ := readModList(path)
	numFiles := len(tempf)

	buf := ""
	count := 1
	for x := 0; x < numFiles; x++ {
		file := tempf[x]

		number := fmt.Sprintf("#%3v ", count)
		info, found := readMod(path + file.Name())
		if found {
			buf = buf + number
			foundMod, foundList := searchModlist(modList, info.Name)

			if foundList {
				if foundMod {
					buf = buf + "( ON) "
				} else {
					buf = buf + "(OFF) "
				}
			} else {
				buf = buf + "( ? ) "
			}

			if checkModLoaded(info.Name) {
				buf = buf + "@"
			} else {
				buf = buf + " "
			}
			buf = buf + info.Name + " v" + info.Version

		} else {
			buf = buf + number + "Filename: " + strings.TrimSuffix(file.Name(), ".zip") + " (corrupt mod file?)"
		}
		buf = buf + "\n"
		count++
	}
	if count == 0 {
		buf = buf + "None."
	}
	return buf, err
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

		for _, file := range zip.File {
			fc, err := file.Open()

			if err != nil {
				buf := fmt.Sprintf("Save '%v' contains corrupt data: '%v', trying next save.", name, err.Error())
				if showError {
					CMS(cfg.Local.Channel.ChatChannel, buf)
				}
				cwlog.DoLogCW(buf)
				break
			} else {
				if strings.HasSuffix(file.Name, "level.dat0") {

					content, err := io.ReadAll(fc)
					if len(content) > (50*1024) && err == nil {
						return true, filepath.Dir(file.Name)
					} else {
						return false, ""
					}
				}
			}
		}
		buf := fmt.Sprintf("Save '%v' did not contain a level.dat0 file.", name)
		if showError {
			CMS(cfg.Local.Channel.ChatChannel, buf)
		}
		cwlog.DoLogCW(buf)
	}

	return false, ""
}

func SetFactRunning(run bool) {
	wasrun := FactIsRunning
	FactIsRunning = run

	if run && glob.NoResponseCount >= 15 && !FactorioBootedAt.IsZero() && time.Since(FactorioBootedAt) > time.Minute {
		//CMS(cfg.Local.Channel.ChatChannel, "Server now appears to be responding again.")
		cwlog.DoLogCW("Server now appears to be responding again.")
	}
	glob.NoResponseCount = 0

	if wasrun != run {
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

/* Whitelist a specifc player. */
func WhitelistPlayer(pname string, level int) {
	if FactorioBooted && FactIsRunning {
		if !cfg.Local.Options.CustomWhitelist {
			return
		}
		if cfg.Local.Options.MembersOnly {
			if level > 0 {
				WriteFact(fmt.Sprintf("/whitelist add %s", pname))
			}
		}
		if cfg.Local.Options.RegularsOnly {
			if level > 1 {
				WriteFact(fmt.Sprintf("/whitelist add %s", pname))
			}
		}
	}
}

/* Write a full whitelist for a server, before it boots */
func WriteAdminlist() int {

	wpath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.AdminlistName

	glob.PlayerListLock.RLock()

	var count = 0
	var buf = "[\n"

	//Add admins
	for _, player := range glob.PlayerList {
		if player.Level >= 254 {
			/* Add admins to whitelist for custom whitelists */
			if cfg.Local.Options.CustomWhitelist {
				WriteFact(fmt.Sprintf("/whitelist add %s", player.Name))
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

	_, err := os.Create(wpath)

	if err != nil {
		cwlog.DoLogCW("WriteAdminlist: os.Create failure")
		return -1
	}

	err = os.WriteFile(wpath, []byte(buf), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteAdminlist: WriteFile failure")
		return -1
	}
	return count
}

/* Write a full whitelist for a server, before it boots */
func WriteWhitelist() int {

	wpath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.WhitelistName

	if cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly {
		glob.PlayerListLock.RLock()

		var count = 0
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

		//Add admins
		for _, player := range glob.PlayerList {
			if player.Level >= 254 {
				buf = buf + "\"" + player.Name + "\",\n"
				count = count + 1
			}
		}

		//Print sorted list
		l := len(localPlayerList) - 1
		for x := l; x > 0; x-- {
			if count >= constants.MaxWhitelist {
				break
			}
			var player = localPlayerList[x]

			buf = buf + "\"" + player.Name + "\",\n"
			count = count + 1
		}

		if count > 1 {
			lchar := len(buf)
			buf = buf[0 : lchar-2]
		}
		buf = buf + "\n]\n"
		glob.PlayerListLock.RUnlock()

		_, err := os.Create(wpath)

		if err != nil {
			cwlog.DoLogCW("WriteWhitelist: os.Create failure")
			return -1
		}

		err = os.WriteFile(wpath, []byte(buf), 0644)

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

	if message == "" {
		message = "Server quitting."
	}

	glob.RelaunchThrottle = 0
	glob.NoResponseCount = 0

	/* Running but no players, just quit */
	if (FactorioBooted && FactIsRunning) && NumPlayers <= 0 {
		WriteFact("/quit")

		/* Running, but players connected... Give them quick feedback. */
	} else if FactorioBooted && FactIsRunning && NumPlayers > 0 {
		FactChat("[color=red]" + message + "[/color]")
		FactChat("[color=green]" + message + "[/color]")
		FactChat("[color=blue]" + message + "[/color]")
		FactChat("[color=white]" + message + "[/color]")
		FactChat("[color=black]" + message + "[/color]")
		time.Sleep(time.Second * 3)
		WriteFact("/quit")
	}
}

/* Send a string to Factorio, via stdin */
func WriteFact(input string) {
	PipeLock.Lock()
	defer PipeLock.Unlock()

	/* Clean string */
	buf := sclean.StripControlAndSubSpecial(input)

	gpipe := Pipe
	if gpipe != nil {

		plen := len(buf)

		if plen > 2000 {
			cwlog.DoLogCW("Message to Factorio, too long... Not sending.")
			return
		} else if plen <= 1 {
			cwlog.DoLogCW("Message for Factorio too short... Not sending.")
			return
		}

		_, err := io.WriteString(gpipe, buf+"\n")
		if err != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to write to Factorio.\nError: %v Input: %v", err, input))
			SetFactRunning(false)
			return
		}
		if buf != "/time" {
			cwlog.DoLogCW(fmt.Sprintf("CW: %v", buf))
		}

	} else {
		//cwlog.DoLogCW("An error occurred when attempting to write to Factorio (nil pipe)")
		SetFactRunning(false)
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
	} else if level >= 255 {
		name = "Moderator"
	}

	return name
}

func StringToLevel(input string) int {

	level := 0

	if strings.EqualFold(input, "new") {
		level = 0
	} else if strings.EqualFold(input, "members") {
		level = 1
	} else if strings.EqualFold(input, "regulars") {
		level = 2
	} else if strings.EqualFold(input, "banished") {
		level = 0
	} else if strings.EqualFold(input, "moderator") {
		level = 255
	}

	return level
}

/* Promote a player to the level they have, in Factorio and on Discord */
func AutoPromote(pname string) string {
	playerName := " *(New Player)* "

	if pname != "" {
		plevel := PlayerLevelGet(pname, false)

		if plevel <= -254 {
			playerName = " **(Deleted Player)** "

		} else if plevel == -1 {
			playerName = " **(Banned)**"

			name := strings.ToLower(pname)
			glob.PlayerListLock.Lock()
			if glob.PlayerList[name] != nil {
				WriteFact("/ban " + pname + " " + glob.PlayerList[name].BanReason)
			}
			glob.PlayerListLock.Unlock()

		} else if plevel == 1 {
			playerName = " *(Member)*"
			WriteFact(fmt.Sprintf("/member %s", pname))

		} else if plevel == 2 {
			playerName = " *(Regular)*"

			WriteFact(fmt.Sprintf("/regular %s", pname))
		} else if plevel == 255 {
			playerName = " *(Moderator)*"

			WriteFact(fmt.Sprintf("/promote %s", pname))
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
			} else if plevel == 255 {
				newrole = cfg.Global.Discord.Roles.Moderator
			}

			guild := disc.Guild

			if guild != nil && disc.DS != nil {

				errrole, regrole := disc.RoleExists(guild, newrole)

				if !errrole {
					cwlog.DoLogCW(fmt.Sprintf("Couldn't find role %v.", newrole))
				} else {
					errset := disc.SmartRoleAdd(cfg.Global.Discord.Guild, discid, regrole.ID)
					if errset != nil {
						cwlog.DoLogCW(fmt.Sprintf("Couldn't set role %v for %v.", newrole, discid))
					}
				}
			} else {

				cwlog.DoLogCW("No guild data.")
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

	if NextResetUnix > 0 {
		mpre := "MAP RESET"
		if cfg.Local.Options.SkipReset {
			mpre = "(SKIP)"
		}
		newTopic = fmt.Sprintf("%v: <t:%v:F>(LOCAL)", mpre, NextResetUnix)
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

		_, aerr = disc.DS.ChannelEditComplex(cfg.Local.Channel.ChatChannel, &discordgo.ChannelEdit{Name: chname, Position: ch.Position, Topic: newTopic})

		if aerr != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to rename the Factorio discord channel. Details: %s", aerr))
			return
		} else {
			oldTopic = newTopic
		}
	}
}

func ShowMapList(s *discordgo.Session, i *discordgo.InteractionCreate, voteMode bool) {

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		log.Fatal(err)
		disc.EphemeralResponse(s, i, "Error:", "Unable to read saves directory.")
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
			Emoji: discordgo.ComponentEmoji{
				Name: "⭐",
			},
		},
		discordgo.SelectMenuOption{

			Label:       "SKIP-RESET",
			Description: "Skip the next map reset.",
			Value:       "SKIP-RESET",
			Emoji: discordgo.ComponentEmoji{
				Name: "🚫",
			},
		},
	)

	for i := 0; i < numFiles; i++ {

		f := tempf[i]
		fName := f.Name()

		if strings.HasSuffix(fName, ".zip") {
			saveName := strings.TrimSuffix(fName, ".zip")
			step++

			units, err := durafmt.DefaultUnitsCoder.Decode("yr:yrs,wk:wks,day:days,hr:hrs,min:mins,sec:secs,ms:ms,μs:μs")
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
					Emoji: discordgo.ComponentEmoji{
						Name: "💾",
					},
				},
			)
		}
	}

	if numFiles <= 0 {
		disc.EphemeralResponse(s, i, "Error:", "No saves were found.")
	} else {

		var response *discordgo.InteractionResponse
		if voteMode {
			response = &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Vote for 'new-map', 'skip-reset' or a specific save-game. (two votes needed):",
					Flags:   1 << 6,
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
					Flags:   1 << 6,
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
		err := s.InteractionRespond(i.Interaction, response)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}
}

func ShowFullMapList(s *discordgo.Session, i *discordgo.InteractionCreate) {

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	files, err := os.ReadDir(path)
	/* We can't read saves dir */
	if err != nil {
		cwlog.DoLogCW(err.Error())
		disc.EphemeralResponse(s, i, "Error:", "Unable to read saves directory.")
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
		disc.EphemeralResponse(s, i, "Error:", "No saves were found.")
	} else {
		var elist []*discordgo.MessageEmbed
		elist = append(elist, &discordgo.MessageEmbed{Title: "Full map list", Description: mapList})

		//1 << 6 is ephemeral/private, don't use disc.EphemeralResponse (logged)
		respData := &discordgo.InteractionResponseData{Embeds: elist, Flags: 1 << 6}
		resp := &discordgo.InteractionResponse{Type: discordgo.InteractionResponseChannelMessageWithSource, Data: respData}
		err := s.InteractionRespond(i.Interaction, resp)
		if err != nil {
			return
		}
	}
}

func DoChangeMap(s *discordgo.Session, arg string) {

	if strings.EqualFold(arg, "new-map") {

		/* Turn off skip reset flag */
		cfg.Local.Options.SkipReset = false
		cfg.WriteLCfg()

		go Map_reset("", false)
		return
	} else if strings.EqualFold(arg, "skip-reset") {
		cfg.Local.Options.SkipReset = true
		cfg.WriteLCfg()

		buf := "VOTE: The next map reset will be skipped."
		FactChat(buf)
		CMS(cfg.Local.Channel.ChatChannel, buf)
		return
	}

	path := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		cfg.Global.Paths.Folders.Saves

	/* Check if file is valid and found */
	saveStr := fmt.Sprintf("%v.zip", arg)
	good, _ := CheckSave(path, saveStr, false)
	if !good {
		cwlog.DoLogCW("DoChangeMap: Attempted to load an invalid save.")
		return
	}

	FactAutoStart = false
	QuitFactorio("Server rebooting for map vote...")
	WaitFactQuit()
	selSaveName := path + "/" + saveStr
	from, erra := os.Open(selSaveName)
	if erra != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to open the selected save. Details: %s", erra))
	}
	defer from.Close()

	newmappath := path + "/" + cfg.Local.Name + "_new.zip"
	_, err := os.Stat(newmappath)
	if !os.IsNotExist(err) {
		err = os.Remove(newmappath)
		if err != nil {
			cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to remove the temp save file. Details: %s", err))
			return
		}
	}
	to, errb := os.OpenFile(newmappath, os.O_RDWR|os.O_CREATE, 0666)
	if errb != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to create the save file. Details: %s", errb))
		return
	}
	defer to.Close()

	_, errc := io.Copy(to, from)
	if errc != nil {
		cwlog.DoLogCW(fmt.Sprintf("An error occurred when attempting to write the save file. Details: %s", errc))
		return
	}

	CMS(cfg.Local.Channel.ChatChannel, fmt.Sprintf("Loading save: %v", arg))
	glob.RelaunchThrottle = 0
	FactAutoStart = true

}
