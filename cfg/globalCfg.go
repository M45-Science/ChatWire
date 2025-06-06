package cfg

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/util"
	"ChatWire/watcher"
)

var (
	// Local holds local server configuration.
	Local local
	// Global holds global configuration shared across servers.
	Global global
)

// WriteGCfg writes the global configuration to disk.
// It returns true on success.
func WriteGCfg() bool {
	finalPath := constants.CWGlobalConfig

	Global.Discord.Comment = "RoleIDs to ping for suspicious or banishment events, if any."
	Global.Discord.Roles.Comment = "These are role names, ChatWire will auto-resolve the IDs and cache them."
	Global.Discord.Roles.RoleCache.Comment = "Cached Role IDs, in case lookup is slow or fails."
	Global.Options.Comment = "RoleID to ping on map resets, if any."

	if err := util.WriteJSONAtomic(finalPath, Global, 0644); err != nil {
		cwlog.DoLogCW("WriteGCfg: " + err.Error())
		return false
	}

	return true
}

func setGlobalDefaults() {
	/* Automatic global defaults */
	if Global.Paths.DataFiles.DBFile == "" {
		Global.Paths.DataFiles.DBFile = "playerdb.json"
		if err := os.WriteFile(Global.Paths.DataFiles.DBFile, []byte("{}"), 0644); err != nil {
			cwlog.DoLogCW("setGlobalDefaults: Could not create " + Global.Paths.DataFiles.DBFile)
		}
	}
	if Global.Paths.Folders.MapGenerators == "" {
		Global.Paths.Folders.MapGenerators = "map-gen-json"

		err := os.MkdirAll(Global.Paths.Folders.ServersRoot+"/"+Global.Paths.Folders.MapGenerators, os.ModePerm)
		if err != nil {
			cwlog.DoLogCW("Could not create map-gen-json directory.")
		}
	}
	if Global.Paths.URLs.Domain == "" {
		Global.Paths.URLs.Domain = "localhost"
	}
	if Global.Options.RconOffset == 0 {
		// Apply default RCON port offset
		Global.Options.RconOffset = constants.RconPortOffset
	}
	if Global.GroupName == "" {
		Global.GroupName = glob.RandomBase64String(3)
		cwlog.DoLogCW("No group name specified. Random one generated.")
	}

	if Global.Paths.Folders.ServersRoot == "" {
		ex, err := os.Executable()
		if err == nil {
			exPath := filepath.Dir(ex)
			p := filepath.Clean(filepath.Join(exPath, ".."))
			Global.Paths.Folders.ServersRoot = p + "/"
		}
	}

	if Global.Paths.ChatWirePrefix == "" {
		Global.Paths.ChatWirePrefix = "cw-"
	}
	if Global.Paths.Folders.Saves == "" {
		Global.Paths.Folders.Saves = "saves"
	}
	if Global.Paths.Folders.Mods == "" {
		Global.Paths.Folders.Mods = "mods"
	}
	if Global.Paths.Folders.MapArchives == "" {
		Global.Paths.Folders.MapArchives = Global.Paths.Folders.ServersRoot + "www/public_html/archive/"
	}
	if Global.Paths.Folders.ModPack == "" {
		Global.Paths.Folders.ModPack = Global.Paths.Folders.ServersRoot + "www/public_html/modpack/"
	}
	if Global.Paths.Folders.FactorioDir == "" {
		Global.Paths.Folders.FactorioDir = "factorio"
	}
	if Global.Paths.URLs.PathPrefix == "" {
		currentUser, err := user.Current()
		if err == nil {
			Global.Paths.URLs.PathPrefix = "/u/" + currentUser.Name
		} else {
			Global.Paths.URLs.PathPrefix = "/~username"
		}
	}
	if Global.Paths.URLs.LogPath == "" {
		Global.Paths.URLs.LogPath = "/logs/"
	}
	if Global.Paths.URLs.LogsPathWeb == "" {
		Global.Paths.URLs.LogsPathWeb = "/current-logs/"
	}
	if Global.Paths.URLs.ArchivePath == "" {
		Global.Paths.URLs.ArchivePath = "/archive/"
	}
	if Global.Paths.URLs.ModPackPath == "" {
		Global.Paths.URLs.ModPackPath = "/modpack/"
	}
	if Global.Paths.Binaries.FactBinary == "" {
		Global.Paths.Binaries.FactBinary = "bin/x64/factorio"
	}
	if Global.Discord.Guild == "" {
		cwlog.DoLogCW("No Discord Guild ID specified. This MUST be set!")
		Global.Discord.Guild = "MY DISCORD GUILD ID"
	}
	if Global.Discord.Application == "" {
		Global.Discord.Application = "MY DISCORD APP ID"
	}
	if Global.Discord.Token == "" {
		cwlog.DoLogCW("No Discord Token specified. This MUST be set!")
		Global.Discord.Token = "MY DISCORD BOT TOKEN"
	}
	if Global.Factorio.Username == "" {
		cwlog.DoLogCW("No Factorio Username specified. This MUST be set!")
		Global.Factorio.Username = "MY FACTORIO USERNAME"
	}
	if Global.Factorio.Token == "" {
		cwlog.DoLogCW("No Factorio Token specified. This MUST be set!")
		Global.Factorio.Token = "MY FACTORIO TOKEN"
	}
	if Global.Options.AutosaveMax == 0 {
		Global.Options.AutosaveMax = 250
	}
	if Global.PrimaryServer == "" {
		Global.PrimaryServer = "a"
	}
}

// ReadGCfg loads the global configuration from disk.
// It creates a default configuration when none exists.
func ReadGCfg() bool {

	_, err := os.Stat(constants.CWGlobalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadGCfg: os.Stat failed, auto-defaults generated.")
		newcfg := createGCfg()
		Global = newcfg

		setGlobalDefaults()
		WriteGCfg()
		return true
	}

	file, err := os.ReadFile(constants.CWGlobalConfig)
	if file != nil && err == nil {
		newcfg := createGCfg()
		err := json.Unmarshal([]byte(file), &newcfg)
		if err != nil {
			cwlog.DoLogCW("ReadGCfg: Unmarshal failure")
			cwlog.DoLogCW(err.Error())
			return false
		}

		Global = newcfg
		setGlobalDefaults()

		return true
	}

	cwlog.DoLogCW("ReadGCfg: ReadFile failure")
	return false
}

// createGCfg returns a new empty global configuration structure.
func createGCfg() global {
	newcfg := global{}
	return newcfg
}

// WatchGCfg monitors the global configuration file for changes.
func WatchGCfg() {
	watcher.Watch(constants.CWGlobalConfig, 5*time.Second, &glob.ServerRunning, func() {
		glob.GlobalCfgUpdatedLock.Lock()
		glob.GlobalCfgUpdated = true
		glob.GlobalCfgUpdatedLock.Unlock()
	})
}
