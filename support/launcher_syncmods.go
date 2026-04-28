package support

import (
	"bufio"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
)

func SyncMods(i *discordgo.InteractionCreate, optionalFileName string) bool {
	opToken := fact.BeginOperation("Mod Sync", "Syncing mods. This can take a while on large modpacks or slow downloads.")
	fact.SetModOperationInProgress(true)
	defer func() {
		fact.SetModOperationInProgress(false)
	}()

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
		fact.FailOperation(opToken, "Mod Sync", "Unable to write Factorio config for mod sync.", glob.COLOR_RED)
		return false
	}

	tempargs := []string{"--sync-mods", fileName}

	// Hard timeout for the full sync operation.
	ctx, cancel := context.WithTimeout(context.Background(), constants.SyncModsHardTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, fact.GetFactorioBinary(), tempargs...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
		fact.FailOperation(opToken, "Mod Sync", "Unable to read mod sync output.", glob.COLOR_RED)
		return false
	}

	if err := cmd.Start(); err != nil {
		cwlog.DoLogCW("Error starting command: %v", err)
		fact.FailOperation(opToken, "Mod Sync", "Unable to start mod sync.", glob.COLOR_RED)
		return false
	}

	lines := make(chan string, 64)
	scanErrCh := make(chan error, 1)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		scanErrCh <- scanner.Err()
	}()

	modsLoading := false
	lastProgress := time.Now()

	for {
		select {
		case <-ctx.Done():
			cwlog.DoLogCW("SyncMods: command context ended: %v", ctx.Err())
			fact.FailOperation(opToken, "Mod Sync", "Mod sync timed out or was canceled.", glob.COLOR_RED)
			return false
		case err := <-scanErrCh:
			if err != nil {
				cwlog.DoLogCW("SyncMods: Error reading stdout: %v\n", err)
				fact.FailOperation(opToken, "Mod Sync", "Mod sync output stream failed.", glob.COLOR_RED)
				return false
			}
			scanErrCh = nil
		case line, ok := <-lines:
			if !ok {
				lines = nil
				continue
			}
			line = strings.TrimSpace(line)
			now := time.Now()

			parts := strings.Fields(line)
			numParts := len(parts)
			if numParts > 1 && parts[1] == "Goodbye" {
				lastProgress = now
			}
			if numParts > 2 && parts[1] == "Loading" && parts[2] == "mod" {
				lastProgress = now
				if !modsLoading {
					modsLoading = true
					msg := "Loading mods"
					fact.UpdateOperationProgress(opToken, "Mod Sync", "Loading mods for sync. This may take a while.", glob.COLOR_CYAN)
					glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
						msg, glob.COLOR_CYAN))
					cwlog.DoLogCW(msg)
				}
			}
			if syncModDownloadLine(parts) {
				lastProgress = now
				if modName, ok := syncModDownloadName(parts); ok {
					msg := "Downloading: " + modName
					fact.UpdateOperationProgress(opToken, "Mod Sync", "Downloading mod data: "+modName, glob.COLOR_CYAN)
					glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
						msg, glob.COLOR_CYAN))
					cwlog.DoLogCW(msg)
				}
			}

			if line == "Mods synced" {
				lastProgress = now
				msg := "All mods synced."
				fact.UpdateOperationProgress(opToken, "Mod Sync", "All mods synced. Finalizing results.", glob.COLOR_CYAN)
				glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Sync",
					msg, glob.COLOR_CYAN))
				cwlog.DoLogCW(msg)
			}
		case <-time.After(time.Second):
			if time.Since(lastProgress) > constants.SyncModsIdleTimeout {
				cwlog.DoLogCW("SyncMods: idle timeout after %v without progress.", constants.SyncModsIdleTimeout)
				cancel()
				fact.FailOperation(opToken, "Mod Sync", "Mod sync timed out waiting for progress.", glob.COLOR_RED)
				return false
			}
		}

		if lines == nil && scanErrCh == nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		cwlog.DoLogCW("SyncMods: Command finished with error: %v\n", err)
		fact.FailOperation(opToken, "Mod Sync", "Mod sync failed: "+err.Error(), glob.COLOR_RED)
		return false
	}

	fact.CompleteOperation(opToken, "Mod Sync", "Mod sync complete.", glob.COLOR_GREEN)
	return true
}

func syncModDownloadLine(parts []string) bool {
	return len(parts) > 3 && parts[3] == "Downloading"
}

func syncModDownloadName(parts []string) (string, bool) {
	if !syncModDownloadLine(parts) || len(parts) <= 4 {
		return "", false
	}
	urlParts := strings.Split(parts[4], "/")
	for i, part := range urlParts {
		if part == "download" && i+1 < len(urlParts) && urlParts[i+1] != "" {
			return urlParts[i+1], true
		}
	}
	return "", false
}
