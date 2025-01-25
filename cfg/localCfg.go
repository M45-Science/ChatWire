package cfg

import (
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinhoefling/goxkcdpwgen/xkcdpwgen"
)

/* LOCAL CONFIG */
type local struct {
	Callsign       string `form:"RO"`
	Name           string
	Port           int    `form:"RO"`
	Comment        string `form:"-"`
	RCONPass       string `form:"hidden" web:"Rcon Password"`
	LastSaveBackup int    `form:"RO" web:"Last Backup Slot"`

	Settings settings

	Channel     channel
	Options     localOptions
	ModPackList []ModPackData `form:"-"`
}

type ModPackData struct {
	Path    string
	Created time.Time
}

type settings struct {
	NewMap         bool `json:"-"`
	Scenario       string
	MapGenerator   string `web:"Map Generator Name"`
	MapPreset      string `web:"Map Preset Name"`
	Seed           int    `web:"Map Seed"`
	AFKMin         int    `web:"AFK Minutes"`
	AutosaveMin    int    `web:"Autosave Minutes"`
	AutoPause      bool   `web:"Pause When Empty"`
	AdminOnlyPause bool   `web:"Admin-Only Pause"`
}

type channel struct {
	Comment     string `form:"-"`
	ChatChannel string `web:"Channel ID"`
}

type localOptions struct {
	LocalDescription string `web:"Factorio Description"`
	Schedule         string `web:"Map Reset Schedule"`
	ResetDay         string `web:"Reset Day"`
	ResetDate        int    `web:"Reset Date"`
	ResetHour        int    `web:"Reset Hour"`
	ResetPingRole    string `form:"-"`
	PlayHourEnable   bool   `web:"Limit Open Hours"`
	PlayStartHour    int    `web:"Open Hour"`
	PlayEndHour      int    `web:"Close Hour"`

	AutoStart       bool    `web:"Auto Boot/Reboot Factorio"`
	AutoUpdate      bool    `web:"Auto Factorio Updates"`
	ExpUpdates      bool    `web:"Factorio Experimental Updates"`
	HideAutosaves   bool    `web:"Hide Autosaves"`
	HideResearch    bool    `web:"Hide Research on Discord"`
	RegularsOnly    bool    `web:"Regulars, Veterans only"`
	MembersOnly     bool    `web:"Members, Regulars, Veterans only"`
	CustomWhitelist bool    `web:"Private whitelist"`
	ModUpdate       bool    `web:"Auto update Factorio Mods"`
	SkipReset       bool    `web:"Will skip next map reset"`
	Speed           float32 `web:"Game Speed Factor"`

	Whitelist bool

	SoftModOptions softmodOptions
}

type softmodOptions struct {
	Restrict          bool   `web:"More Restrictions For New Players"`
	FriendlyFire      bool   `web:"Friendly Fire"`
	OneLife           bool   `web:"One-Life Permadeath"`
	DisableBlueprints bool   `web:"No Blueprints"`
	Cheats            bool   `web:"Cheats Sandbox Mode"`
	InjectSoftMod     bool   `web:"Inject Softmod"`
	SoftModPath       string `form:"-"`
}

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

func WriteLCfg() bool {
	tempPath := constants.CWLocalConfig + "." + Local.Callsign + ".tmp"
	finalPath := constants.CWLocalConfig

	Local.Channel.Comment = "ChannelID, if blank will attempt to create a new channel."
	Local.Comment = "RCONPass is random, generated each launch. Only here for other app to read."

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
