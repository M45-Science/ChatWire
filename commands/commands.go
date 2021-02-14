package commands

import (
	"fmt"
	"strings"

	"../cfg"
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
	CL.CommandList = append(CL.CommandList, Command{Name: "Debug", Command: admin.Debug, Admin: true, Help: "Shows debug info"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Pset", Command: admin.SetPlayerLevel, Admin: true, Help: "Set: player level"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Regular", Command: admin.SetPlayerRegular, Admin: true, Help: "Set: players to regular"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Whitelist", Command: admin.SendWhitelist, Admin: true, Help: "Send whitelist to server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Archive", Command: admin.ArchiveMap, Admin: true, Help: "Archive current map"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Update", Command: admin.Update, Admin: true, Help: "Update Factorio or type CANCEL"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Locks", Command: admin.ShowLocks, Admin: true, Help: "Check Mutex Locks"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Cfg", Command: admin.ReloadConfig, Admin: true, Help: "Reload config file"})
	CL.CommandList = append(CL.CommandList, Command{Name: "NewMap", Command: admin.NewMap, Admin: true, Help: "Quickly stop server, archive, generate map and reboot."})

	// Util Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Whois", Command: user.Whois, Admin: false, Help: "Show player info"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Online", Command: user.PlayersOnline, Admin: false, Help: "Show players online"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Info", Command: user.Info, Admin: false, Help: "Show game & server info"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Mods", Command: user.ModsList, Admin: false, Help: "Show installed game-mods"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Register", Command: user.AccessServer, Admin: false, Help: "Get promoted to members, or regulars role on Discord."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Help", Command: Help, Admin: false, Help: "You are here"})
}

// RunCommand runs a specified command.
func RunCommand(name string, s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	for _, command := range CL.CommandList {
		if strings.EqualFold(command.Name, name) {
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

	fact.CMS(m.ChannelID, "Invalid command, try "+cfg.Global.DiscordCommandPrefix+"help")
}

// CheckAdmin checks if the user attempting to run an admin command is an admin
func CheckAdmin(ID string) bool {
	for _, admin := range cfg.Global.AdminData.IDs {
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
			buf = buf + fmt.Sprintf("%s%-12s %s %s\n", cfg.Global.
				DiscordCommandPrefix, strings.ToLower(command.Name), admin, command.Help)
		}
	} else {
		for _, command := range CL.CommandList {
			if command.Admin == false {
				buf = buf + fmt.Sprintf("%s%-12s %s\n", cfg.Global.
					DiscordCommandPrefix, strings.ToLower(command.Name), command.Help)
			}
		}
	}

	buf = buf + "\n```"

	fact.CMS(m.ChannelID, buf)
}
