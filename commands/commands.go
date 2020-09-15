package commands

import (
	"fmt"
	"strings"

	"../config"
	"../fact"
	"./admin"
	"./user"

	"github.com/bwmarrin/discordgo"
)

// Commands is a struct containing a slice of Command.
type Commands struct {
	CommandList []Command
}

// Command is a struct containing fields that hold command information.
type Command struct {
	Name    string
	Command func(s *discordgo.Session, m *discordgo.MessageCreate, args []string)
	Admin   bool
	Help    string
}

// CL is a Commands interface.
var CL Commands

// RegisterCommands registers the commands on start up.
func RegisterCommands() {
	// Admin Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Stop", Command: admin.StopServer, Admin: true, Help: "Stops Factorio"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Restart", Command: admin.Restart, Admin: true, Help: "Restarts Factorio"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Start", Command: admin.Restart, Admin: true, Help: "Restarts Factorio"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Reload", Command: admin.Reload, Admin: true, Help: "Reloads bot & Factorio"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Reboot", Command: admin.Reboot, Admin: true, Help: "Force close bot."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Save", Command: admin.SaveServer, Admin: true, Help: "Tell Factorio to save map"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Queue", Command: admin.Queue, Admin: true, Help: "Reloads bot, and Factorio when server is empty."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Rand", Command: admin.RandomMap, Admin: true, Help: "Make a new random map preview"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Gen", Command: admin.Generate, Admin: true, Help: "Makes a new map from preview"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Stat", Command: admin.StatServer, Admin: true, Help: "Shows bot stats"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Pset", Command: admin.SetPlayerLevel, Admin: true, Help: "Set: player level"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Regular", Command: admin.SetPlayerRegular, Admin: true, Help: "Set: players to regular"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Whitelist", Command: admin.SendWhitelist, Admin: true, Help: "Send whitelist to server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Archive", Command: admin.ArchiveMap, Admin: true, Help: "Archive current map"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Update", Command: admin.Update, Admin: true, Help: "Update Factorio"})

	// Util Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Whois", Command: user.Whois, Admin: false, Help: "Show player info"})

	CL.CommandList = append(CL.CommandList, Command{Name: "Online", Command: user.PlayersOnline, Admin: false, Help: "Show players online"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Version", Command: user.GameVersion, Admin: false, Help: "Show Factorio version"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Mods", Command: user.ModsList, Admin: false, Help: "Show installed game-mods"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Register", Command: user.AccessServer, Admin: false, Help: "Get promoted to members, or regulars role on Discord."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Help", Command: Help, Admin: false, Help: "You are here"})
}

// RunCommand runs a specified command.
func RunCommand(name string, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	for _, command := range CL.CommandList {
		if strings.ToLower(command.Name) == strings.ToLower(name) {
			if command.Admin && CheckAdmin(m.Author.ID) {
				command.Command(s, m, args)
				return
			}

			if command.Admin == false {
				command.Command(s, m, args)
			}
			return
		}
	}

	fact.CMS(m.ChannelID, "Invalid command.")
}

// CheckAdmin checks if the user attempting to run an admin command is an admin
func CheckAdmin(ID string) bool {
	for _, admin := range config.Config.AdminIDs {
		if ID == admin {
			return true
		}
	}
	return false
}

func Help(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	buf := "Help:\n```\n"

	if CheckAdmin(m.Author.ID) == true {
		for _, command := range CL.CommandList {
			admin := ""
			if command.Admin {
				admin = "(Admin)"
			}
			buf = buf + fmt.Sprintf("%s%-12s %s %s\n", config.Config.
				Prefix, strings.ToLower(command.Name), admin, command.Help)
		}
	} else {
		for _, command := range CL.CommandList {
			if command.Admin == false {
				buf = buf + fmt.Sprintf("%s%-12s %s\n", config.Config.
					Prefix, strings.ToLower(command.Name), command.Help)
			}
		}
	}

	buf = buf + "\n```"

	fact.CMS(m.ChannelID, buf)
	return
}
