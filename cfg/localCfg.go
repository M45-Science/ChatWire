package cfg

import (
	"encoding/json"

	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"ChatWire/util"
	"ChatWire/watcher"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

var executableLookup = os.Executable

// GetGameLogURL builds the web URL to the current game log.
func GetGameLogURL() string {
	if Global.Paths.URLs.LogsPathWeb == "" {
		return ""
	}
	return fmt.Sprintf("https://%v%v%v%v%v",
		Global.Paths.URLs.Domain,
		Global.Paths.URLs.PathPrefix,
		Global.Paths.URLs.LogsPathWeb,
		Local.Callsign+"/",
		strings.TrimPrefix(glob.GameLogName, "log/"))
}

// WriteLCfg writes the local configuration to disk.
// It returns true on success.
func WriteLCfg() bool {
	finalPath := constants.CWLocalConfig

	Local.Channel.Comment = "ChannelID, if blank will attempt to create a new channel."

	if err := util.WriteJSONAtomic(finalPath, Local, 0644); err != nil {
		cwlog.DoLogCW("WriteLCfg: " + err.Error())
		return false
	}

	return true
}

func setLocalDefaults() {
	/* Automatic local defaults */
	if Local.Name == "" {
		g := xkcdpwgen.NewGenerator()
		g.SetNumWords(1)
		g.SetCapitalize(false)
		g.SetDelimiter("")
		Local.Name = g.GeneratePasswordString()
	}
	if Local.Callsign == "" {
		ex, err := executableLookup()
		if err != nil {
			cwlog.DoLogCW("setLocalDefaults: unable to resolve executable path: %v", err)
			Local.Callsign = "cw"
		} else {
			dir := filepath.Dir(ex)
			candidate := ""
			for depth := 0; depth < 3; depth++ {
				base := filepath.Base(dir)
				base = strings.TrimPrefix(base, "cw-")
				base = strings.ToLower(base)
				if _, ok := glob.AlphaValue[base]; ok {
					candidate = base
					break
				}
				parent := filepath.Dir(dir)
				if parent == dir {
					break
				}
				dir = parent
			}
			if candidate != "" {
				Local.Callsign = candidate
			} else {
				Local.Callsign = "a"
			}
		}
	}
	if Local.Port <= 0 {
		Local.Port = glob.AlphaValue[strings.ToLower(Local.Callsign)]
	}
	if Local.Settings.AFKMin <= 0 {
		Local.Settings.AFKMin = 15
	}
	if Local.Options.Speed <= 0 {
		Local.Options.Speed = 1
	}
	if Local.Settings.AutosaveMin <= 0 {
		Local.Settings.AutosaveMin = 15
	}
	if Local.Options.PlayerPollIntervalSec <= 0 {
		Local.Options.PlayerPollIntervalSec = 300
	}
	if Local.Options.RoleRefreshIntervalSec <= 0 {
		Local.Options.RoleRefreshIntervalSec = 300
	}
	if Local.Options.MapResetCheckIntervalSec <= 0 {
		Local.Options.MapResetCheckIntervalSec = 300
	}
	if Local.Channel.ChatChannel == "" {
		cwlog.DoLogCW("ReadLCfg: ChatID not set, this MUST be set to a valid Discord channel ID!")
		Local.Channel.ChatChannel = "MY DISCORD CHANNEL ID"
	}
	if Local.Options.SoftModOptions.SoftModPath == "" {
		path := Global.Paths.Folders.ServersRoot +
			Global.Paths.ChatWirePrefix +
			Local.Callsign + "/" +
			Global.Paths.Folders.FactorioDir + "/" +
			"softmod/"
		Local.Options.SoftModOptions.SoftModPath = path

		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			cwlog.DoLogCW("ReadLCfg: unable to create softmod path %v: %v", path, err)
		}
	}
	if !Local.Options.RegularsOnly {
		Local.Settings.AdminOnlyPause = true
	} else {
		Local.Settings.AdminOnlyPause = false
	}
}

// ReadLCfg loads the local configuration from disk, creating defaults if needed.
func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadLCfg: os.Stat failed, auto-defaults generated.")
		newcfg := createLCfg()
		Local = newcfg
		Local.Channel.ChatChannel = util.TrimPrefixIgnoreCase(strings.TrimSpace(Local.Channel.ChatChannel), "channel id:")
		setLocalDefaults()
		if !Local.Settings.AutoPause {
			Local.Settings.AutoPause = true
		}
		Local.Settings.AdminOnlyPause = true
		WriteLCfg() /* Write the defaults */
		return true
	}

	file, err := os.ReadFile(constants.CWLocalConfig)
	if file != nil && err == nil {
		newcfg := createLCfg()

		err := json.Unmarshal([]byte(file), &newcfg)
		if err != nil {
			cwlog.DoLogCW("ReadLCfg: Unmarshal failure")
			cwlog.DoLogCW(err.Error())
			return false
		}

		Local = newcfg
		Local.Channel.ChatChannel = util.TrimPrefixIgnoreCase(strings.TrimSpace(Local.Channel.ChatChannel), "channel id:")
		setLocalDefaults()

		/* Automatic local defaults */
		found := false
		for _, t := range constants.MapTypes {
			if strings.EqualFold(Local.Settings.MapPreset, t) {
				found = true
			}
		}
		if !found {
			Local.Settings.MapPreset = constants.MapTypes[1]
			cwlog.DoLogCW("ReadLCfg: MapPreset not valid, setting to " + Local.Settings.MapPreset)
		}

		//Migrate old setting
		if newcfg.Options.Whitelist {
			newcfg.Options.MembersOnly = true
			newcfg.Options.Whitelist = false
		}
		if newcfg.Options.RegularsOnly {
			newcfg.Options.MembersOnly = false
		}

		return true
	}

	cwlog.DoLogCW("ReadLCfg: ReadFile failure")
	return false
}

// createLCfg returns a new empty local configuration structure.
func createLCfg() local {
	newcfg := local{}
	return newcfg
}

// WatchLCfg monitors the local configuration file for changes.
func WatchLCfg() {
	watcher.Watch(constants.CWLocalConfig, 5*time.Second, &glob.ServerRunning, func() {
		time.Sleep(time.Second)
		glob.LocalCfgUpdatedLock.Lock()
		glob.LocalCfgUpdated = true
		glob.LocalCfgUpdatedLock.Unlock()
	})
}
