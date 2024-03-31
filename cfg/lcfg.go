package cfg

import (
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

/* LOCAL CONFIG */
type local struct {
	Callsign       string
	Name           string
	Port           int
	RCONPass       string
	LastSaveBackup int

	Settings settings

	Channel     channel
	Options     localOptions
	ModPackList []ModPackData
}

type ModPackData struct {
	Path    string
	Created time.Time
}

type settings struct {
	MapGenerator   string
	MapPreset      string
	Seed           int
	AFKMin         int
	AutosaveMin    int
	AutoPause      bool
	AdminOnlyPause bool
}

type channel struct {
	ChatChannel string
}

type localOptions struct {
	LocalDescription string
	Schedule         string
	ResetDay         string
	ResetDate        int
	ResetHour        int
	ResetPingRole    string
	PlayHourEnable   bool
	PlayStartHour    int
	PlayEndHour      int

	AutoStart       bool
	AutoUpdate      bool
	ExpUpdates      bool
	HideAutosaves   bool
	HideResearch    bool
	RegularsOnly    bool
	MembersOnly     bool
	CustomWhitelist bool
	ModUpdate       bool
	SkipReset       bool
	Speed           float32

	Whitelist bool

	SoftModOptions softmodOptions
}

type softmodOptions struct {
	Restrict          bool
	FriendlyFire      bool
	DisableBlueprints bool
	Cheats            bool
	InjectSoftMod     bool
	SoftModPath       string
}

func WriteLCfg() bool {
	tempPath := constants.CWLocalConfig + "." + Local.Callsign + ".tmp"
	finalPath := constants.CWLocalConfig

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")

	if err := enc.Encode(Local); err != nil {
		cwlog.DoLogCW("WriteLCfg: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("WriteLCfg: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("WriteLCfg: WriteFile failure")
	}

	err = os.Rename(tempPath, finalPath)

	if err != nil {
		cwlog.DoLogCW("Couldn't rename Lcfg file.")
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
		ex, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exPath := filepath.Dir(ex)
		exPath = strings.TrimPrefix(exPath, "cw-")
		if len(exPath) > 0 && len(exPath) < 4 {
			Local.Callsign = exPath
		} else {
			Local.Callsign = "a"
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

		os.Mkdir(path, os.ModePerm)
	}
	if !Local.Options.RegularsOnly {
		Local.Settings.AdminOnlyPause = true
	} else {
		Local.Settings.AdminOnlyPause = false
	}
}

func ReadLCfg() bool {

	_, err := os.Stat(constants.CWLocalConfig)
	notfound := os.IsNotExist(err)

	if notfound {
		cwlog.DoLogCW("ReadLCfg: os.Stat failed, auto-defaults generated.")
		newcfg := CreateLCfg()
		Local = newcfg
		setLocalDefaults()
		if !Local.Settings.AutoPause {
			Local.Settings.AutoPause = true
		}
		Local.Settings.AdminOnlyPause = true
		WriteLCfg() /* Write the defaults */
		return true
	} else { /* Just read the config */

		file, err := os.ReadFile(constants.CWLocalConfig)

		if file != nil && err == nil {
			newcfg := CreateLCfg()

			err := json.Unmarshal([]byte(file), &newcfg)
			if err != nil {
				cwlog.DoLogCW("ReadLCfg: Unmarshal failure")
				cwlog.DoLogCW(err.Error())
				return false
			}

			Local = newcfg
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

			if newcfg.Options.RegularsOnly {
				ServerPrefix = constants.RegularsPrefix
			} else if newcfg.Options.MembersOnly {
				ServerPrefix = constants.MembersPrefix
			} else {
				ServerPrefix = ""
			}

			return true
		} else {
			cwlog.DoLogCW("ReadLCfg: ReadFile failure")
			return false
		}

	}
}

func CreateLCfg() local {
	newcfg := local{}
	return newcfg
}
