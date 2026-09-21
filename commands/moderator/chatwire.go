package moderator

import (
	"github.com/bwmarrin/discordgo"

	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/support"
)

/* Reboots cw */
func ForceReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Force rebooting!")
	glob.RelaunchThrottle = 0
	_ = fact.SubmitLifecycleRequest(fact.Request{
		Kind:              fact.ActionRestartChatWire,
		Reason:            "Server rebooting...",
		ForceChatWireExit: true,
	})
}

/* Reboot when server is empty */
func QueReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Complete:", "Reboot has been queued. Server will reboot when map is unoccupied.")
	_ = fact.SubmitLifecycleRequest(fact.Request{
		Kind:      fact.ActionRestartChatWire,
		Reason:    "Server rebooting for maintenance.",
		WhenEmpty: true,
	})
}

/* Reboot when server is empty */
func QueFactReboot(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Complete:", "Factorio Reboot has been queued. Server will reboot when map is unoccupied.")
	_ = fact.SubmitLifecycleRequest(fact.Request{
		Kind:      fact.ActionRestartFactorio,
		Reason:    "Rebooting Factorio.",
		WhenEmpty: true,
	})
}

/*  Restart saves and restarts the server */
func RebootCW(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	disc.InteractionEphemeralResponse(i, "Status:", "Rebooting ChatWire...")
	glob.RelaunchThrottle = 0
	_ = fact.SubmitLifecycleRequest(fact.Request{
		Kind:   fact.ActionRestartChatWire,
		Reason: "Server rebooting...",
	})
}

/* Reload config files */
func ReloadConfig(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	if err := support.ReloadConfigFilesResult("Discord"); err != nil {
		disc.InteractionEphemeralResponse(i, "Error", err.Error())
		return
	}
	disc.InteractionEphemeralResponse(i, "Complete", "Config files have been reloaded.")
}
