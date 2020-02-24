package commands

import (
	"fmt"
	"strings"

	"../commands/admin"
	"../commands/utils"
	"../glob"
	"../support"
	"github.com/bwmarrin/discordgo"
)

// Commands is a struct containing a slice of Command.
type Commands struct {
	CommandList []Command
}

// Command is a struct containing fields that hold command information.
type Command struct {
	Name    string
	Command func(s *discordgo.Session, m *discordgo.MessageCreate)
	Admin   bool
	Help    string
}

// CL is a Commands interface.
var CL Commands

// RegisterCommands registers the commands on start up.
func RegisterCommands() {
	// Admin Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Stop", Command: admin.StopServer, Admin: true, Help: "Stops server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Restart", Command: admin.Restart, Admin: true, Help: "Restarts server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Start", Command: admin.Restart, Admin: true, Help: "Starts server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Reload", Command: admin.Reload, Admin: true, Help: "Reloads bot & server"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Reboot", Command: admin.Reboot, Admin: true, Help: "Force restart bot"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Save", Command: admin.SaveServer, Admin: true, Help: "Tell server to save map"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Queue", Command: admin.Queue, Admin: true, Help: "Reloads bot when server is empty."})
	CL.CommandList = append(CL.CommandList, Command{Name: "Rand", Command: admin.RandomMap, Admin: true, Help: "Make a random map preview"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Gen", Command: admin.Generate, Admin: true, Help: "Makes a map from preview"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Stat", Command: admin.StatServer, Admin: true, Help: "Shows server stats"})

	// Util Commands
	CL.CommandList = append(CL.CommandList, Command{Name: "Online", Command: utils.PlayersOnline, Admin: false, Help: "Show players online"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Mods", Command: utils.ModsList, Admin: false, Help: "Show installed mods"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Access", Command: utils.AccessServer, Admin: false, Help: "Get promoted to regular on Discord, from Factorio (Incomplete)"})
	CL.CommandList = append(CL.CommandList, Command{Name: "Help", Command: Help, Admin: false, Help: "You are here"})
}

// RunCommand runs a specified command.
func RunCommand(name string, s *discordgo.Session, m *discordgo.MessageCreate) {
	for _, command := range CL.CommandList {
		//support.Log(command.Name + " " + name)
		if strings.ToLower(command.Name) == strings.ToLower(name) {
			if command.Admin && CheckAdmin(m.Author.ID) {
				command.Command(s, m)
			}

			if command.Admin == false {
				command.Command(s, m)
			}
			return
		}
	}
}

// CheckAdmin checks if the user attempting to run an admin command is an admin
func CheckAdmin(ID string) bool {
	for _, admin := range support.Config.AdminIDs {
		if ID == admin {
			return true
		}
	}
	return false
}

func Help(s *discordgo.Session, m *discordgo.MessageCreate) {

	buf := "```\n"

	for _, command := range CL.CommandList {
		admin := ""
		if command.Admin {
			admin = "(Admin-Only)"
		}
		buf = buf + fmt.Sprintf("%s%-12s %s %s\n", support.Config.Prefix, strings.ToLower(command.Name), admin, command.Help)
	}
	buf = buf + "\n```"

	_, errb := glob.DS.ChannelMessageSend(support.Config.FactorioChannelID, buf)
	if errb != nil {
		support.ErrorLog(errb)
	}
	return
}
