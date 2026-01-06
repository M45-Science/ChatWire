package support

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

const (
	dotInterval      = time.Second * 5
	progressInterval = time.Second * 10
	syncModsTimeout  = time.Minute * 15
)

func SyncMods(i *discordgo.InteractionCreate, optionalFileName string) bool {

	glob.FactorioLock.Lock()
	defer glob.FactorioLock.Unlock()

	fileName := ""
	if optionalFileName == "" {
		_, fileName, _ = GetSaveGame(true)
	} else {
		fileName = optionalFileName
	}

	if !fact.GenerateFactorioConfig() {
		cwlog.DoLogCW("Unable to write factorio config file.")
		return false
	}

	tempargs := []string{"--sync-mods", fileName}

	// Create a context with the timeout
	ctx, cancel := context.WithTimeout(context.Background(), syncModsTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, fact.GetFactorioBinary(), tempargs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
		return false
	}

	if err := cmd.Start(); err != nil {
		cwlog.DoLogCW("Error starting command: %v", err)
	}

	scanner := bufio.NewScanner(stdout)

	modsLoading := false
	var lastDot time.Time

	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		//cwlog.DoLogCW("Received line: %s\n", line)
		parts := strings.Split(line, " ")
		numParts := len(parts)
		if numParts > 1 {
			if parts[1] == "Goodbye" {
				break
			}
		}
		if numParts > 2 {
			if parts[1] == "Loading" && parts[2] == "mod" {
				if !modsLoading {
					modsLoading = true
					msg := "Loading mods"
					glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
						msg, glob.COLOR_CYAN))
					cwlog.DoLogCW(msg)
				} else if time.Since(lastDot) > dotInterval {
					lastDot = time.Now()
					if glob.GetUpdateMessage() != nil && len(glob.GetUpdateMessage().Embeds) > 0 {
						embed := glob.GetUpdateMessage().Embeds[0]
						embed.Description = embed.Description + " ."
						disc.DS.ChannelMessageEditEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage().ID, embed)
					}
				}
			}
		}
		if numParts > 3 {
			if parts[3] == "Downloading" {
				urlParts := strings.Split(parts[4], "/")
				if len(urlParts) > 1 {
					if urlParts[1] == "download" {
						msg := "Downloading: " + urlParts[2]
						glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
							msg, glob.COLOR_CYAN))
						cwlog.DoLogCW(msg)
					}
				}
			}
		}

		if line == "Mods synced" {
			msg := "All mods synced."
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
				msg, glob.COLOR_CYAN))
			cwlog.DoLogCW(msg)
		}
	}

	if err := scanner.Err(); err != nil {
		cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
		return false
	}

	if err := cmd.Wait(); err != nil {
		cwlog.DoLogCW("SyncMods: Command finished with error: %v\n", err)
		return false
	}

	return true
}
