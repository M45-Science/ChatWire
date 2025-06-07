package commands

import "ChatWire/glob"

// ListModAdminCommands returns all moderator and admin commands.
func ListModAdminCommands() []glob.CommandData {
	var out []glob.CommandData
	for _, c := range cmds {
		if c.ModeratorOnly || c.AdminOnly {
			out = append(out, c)
		}
	}
	return out
}

// ListAllCommands returns all registered slash commands.
func ListAllCommands() []glob.CommandData {
	return cmds
}
