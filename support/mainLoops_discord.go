package support

import (
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func startGuildSyncLoop() {
	/***************************
	 * Get Guild information
	 * Needed for Discord roles
	 ***************************/
	go func() {
		for glob.ServerRunning {

			/* Get guild id, if we need it */

			if disc.Guild == nil && disc.DS != nil {
				var nguild *discordgo.Guild
				var err error

				/*  Attempt to get the guild from the state,
				 *  If there is an error, fall back to the restapi. */
				nguild, err = disc.DS.State.Guild(cfg.Global.Discord.Guild)
				if err != nil {
					nguild, err = disc.DS.Guild(cfg.Global.Discord.Guild)
					if err != nil {
						cwlog.DoLogCW("Failed to get valid guild data, giving up.")
						break
					}
				}

				if err != nil {
					cwlog.DoLogCW("Was unable to get guild data from GuildID: %s", err)

					break
				}
				if nguild == nil {
					disc.Guildname = constants.Unknown
					cwlog.DoLogCW("Guild data came back nil.")
					break
				} else {

					/* Guild found, exit loop */
					disc.Guild = nguild
					disc.Guildname = nguild.Name
					cwlog.DoLogCW("Guild data linked.")
					fact.LoadPlayers(true, false, false)
				}
			}

			/* Update role IDs */
			if disc.Guild != nil {
				roleMap := buildRoleMap()

				changed := false
				for _, role := range disc.Guild.Roles {
					if updateRoleCache(role, roleMap) {
						changed = true
					}
				}

				if changed {
					cwlog.DoLogCW("Role IDs updated.")
					cfg.WriteGCfg()
				}
			}

			time.Sleep(time.Minute)
		}
	}()
}

func startRoleRefreshLoop() {
	/*******************************
	 * Update patreon/nitro players
	 *******************************/
	go func() {
		time.Sleep(time.Minute)
		for glob.ServerRunning {

			if !isIdle() {
				disc.UpdateRoleList()

				/* Live update server description */
				if disc.RoleListUpdated {
					ConfigSoftMod()
					fact.GenerateFactorioConfig()
				}
				disc.RoleListUpdated = false
			}
			time.Sleep(time.Duration(cfg.Local.Options.RoleRefreshIntervalSec) * time.Second)
		}
	}()
}
