package fact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Distortions81/rcon"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

/* Generate a server-settings.json file for Factorio */
type VisData struct {
	Public bool `json:"public"`
	Lan    bool `json:"lan"`
	Steam  bool `json:"steam"`
}

type FactConf struct {
	Comment     string   `json:"_comment"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Max_players int      `json:"max_players"`
	Visibility  VisData  `json:"visibility"`

	Username                  string `json:"username"`
	Token                     string `json:"token"`
	Require_user_verification bool   `json:"require_user_verification"`
	Max_heartbeats_per_second int    `json:"max_heartbeats_per_second"`
	Allow_commands            string `json:"allow_commands"`

	Autosave_interval       int  `json:"autosave_interval"`
	Autosave_slots          int  `json:"autosave_slots"`
	Afk_autokick_interval   int  `json:"afk_autokick_interval"`
	Auto_pause              bool `json:"auto_pause"`
	Only_admins_can_pause   bool `json:"only_admins_can_pause_the_game"`
	Autosave_only_on_server bool `json:"autosave_only_on_server"`
}

/* Generate a server-settings.json file for Factorio */
func GenerateFactorioConfig() bool {

	tempPath := constants.ServSettingsName + ".tmp"
	finalPath := cfg.Global.Paths.Folders.ServersRoot +
		cfg.Global.Paths.ChatWirePrefix +
		cfg.Local.Callsign + "/" +
		cfg.Global.Paths.Folders.FactorioDir + "/" +
		constants.ServSettingsName

	var servName string

	cfg.ServerPrefix = ""
	servName = "\u0080[" + cfg.Global.GroupName + "] " + strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name

	/*
		if cfg.Local.Options.CustomWhitelist {
			cfg.ServerPrefix = constants.PrivatePrefix
			servName = "\u0080[" + cfg.Global.GroupName + "-" + cfg.ServerPrefix + "] " + strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name
		} else if cfg.Local.Options.MembersOnly || cfg.Local.Options.RegularsOnly {
			if cfg.Local.Options.RegularsOnly {
				cfg.ServerPrefix = constants.RegularsPrefix
			} else {
				cfg.ServerPrefix = constants.MembersPrefix
			}
			servName = "\u0080[" + cfg.Global.GroupName + "-" + cfg.ServerPrefix + "] " + strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name

		} else {
			cfg.ServerPrefix = ""
			servName = "\u0080 [" + cfg.Global.GroupName + "] " + strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name
		} */

	/* Setup some defaults */
	heartbeats := 60
	autosaves := 250
	autosave_interval := 15
	autokick := 30

	if cfg.Global.Options.AutosaveMax > 0 {
		autosaves = cfg.Global.Options.AutosaveMax
	}
	if cfg.Local.Settings.AutosaveMin > 0 {
		autosave_interval = cfg.Local.Settings.AutosaveMin
	}
	if cfg.Local.Settings.AFKMin > 0 {
		autokick = cfg.Local.Settings.AFKMin
	}

	/* Add some settings to descrLines, such as cheats, no blueprint, etc. */

	var descrLines []string

	if NextReset != "" {
		descrLines = append(descrLines, AddFactColor("orange", "MAP RESETS: "+NextReset+" ("+cfg.Local.Options.Schedule+")"))
	}

	if cfg.Local.Options.CustomWhitelist {
		descrLines = append(descrLines, AddFactColor("red", "Whitelist: INVITE-ONLY"))
	} else if cfg.Local.Options.RegularsOnly {
		descrLines = append(descrLines, AddFactColor("red", "Whitelist: REGULARS-ONLY"))
	} else if cfg.Local.Options.MembersOnly {
		descrLines = append(descrLines, AddFactColor("red", "Whitelist: MEMBERS-ONLY"))
	} else {
		descrLines = append(descrLines, AddFactColor("green", "OPEN-TO-PUBLIC"))
	}

	if cfg.Local.Options.SoftModOptions.FriendlyFire {
		descrLines = append(descrLines, AddFactColor("orange", "FRIENDLY FIRE"))
	}
	if cfg.Local.Options.SoftModOptions.Cheats {
		descrLines = append(descrLines, AddFactColor("yellow", "SANDBOX"))
	}
	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		descrLines = append(descrLines, AddFactColor("blue", "NO BLUEPRINTS"))
	}
	if !cfg.Local.Settings.AutoPause {
		descrLines = append(descrLines, AddFactColor("orange", "AUTO-PAUSE OFF"))
	}
	/*if cfg.Global.AuthServerBans {
		descrLines = append(descrLines, "Auth-server bans enabled")
	}*/
	var tags []string
	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
		descrLines = append(descrLines, "Map generator: "+cfg.Local.Settings.MapGenerator)
		tags = append(tags, "gen-"+cfg.Local.Settings.MapGenerator)
	} else if cfg.Local.Settings.MapPreset != "" {
		descrLines = append(descrLines, "Map preset: "+cfg.Local.Settings.MapPreset)
		tags = append(tags, "preset-"+cfg.Local.Settings.MapPreset)
	}
	/*if cfg.Global.FactorioData.Username != "" {
		descrLines = append(descrLines, "Server owner: "+cfg.Global.FactorioData.Username)
	}*/
	descrLines = append(descrLines, AddFactColor("green", fmt.Sprintf("Direct connect: %v:%v", cfg.Global.Paths.URLs.Domain, cfg.Local.Port)))

	gdesc := strings.Split(cfg.Global.Options.Description, "\n")
	descrLines = append(descrLines, gdesc...)

	ldesc := strings.Split(cfg.Local.Options.LocalDescription, "\n")
	descrLines = append(descrLines, ldesc...)

	tags = append(tags, cfg.Global.GroupName)

	if cfg.Local.Options.CustomWhitelist {
		tags = append(tags, "INVITE-ONLY")
	} else if cfg.Local.Options.MembersOnly {
		tags = append(tags, "MEMBERS-ONLY")
	} else if cfg.Local.Options.RegularsOnly {
		tags = append(tags, "REGULARS-ONLY")
	} else {
		tags = append(tags, "PUBLIC")
	}
	tags = append(tags, cfg.Global.Paths.URLs.Domain)

	serverDescString := strings.Join(descrLines, "\n") + "\n[color=purple]Supporters: " + strings.Join(disc.RoleList.Patreons, ", ") + ", " + strings.Join(disc.RoleList.Supporters, ", ") + "[/color]\n[color=cyan]Nitro Boosters: " + strings.Join(disc.RoleList.NitroBooster, ", ") + "[/color]\n"

	normalMode := true
	if *glob.LocalTestMode {
		normalMode = false
	}

	conf := FactConf{
		Comment:     "Auto-generated! DO NOT MODIFY! Changes will be overwritten!",
		Name:        servName,
		Description: serverDescString,
		Tags:        tags,
		Max_players: 0,
		Visibility: VisData{
			Public: normalMode, /* DEBUG ONLY */
			Lan:    false,
			Steam:  normalMode, /* DEBUG ONLY */
		},

		Username:                  cfg.Global.Factorio.Username,
		Token:                     cfg.Global.Factorio.Token,
		Require_user_verification: normalMode, /* DEBUG ONLY */
		Max_heartbeats_per_second: heartbeats,
		Allow_commands:            "admins-only",

		Autosave_interval:       autosave_interval,
		Autosave_slots:          autosaves,
		Afk_autokick_interval:   autokick,
		Auto_pause:              cfg.Local.Settings.AutoPause,
		Only_admins_can_pause:   cfg.Local.Settings.AdminOnlyPause,
		Autosave_only_on_server: true,
	}

	c := "/config set"
	if FactorioBooted && FactIsRunning {
		/* Send over rcon, to preserve newlines */
		portstr := fmt.Sprintf("%v", cfg.Local.Port+cfg.Global.Options.RconOffset)
		remoteConsole, err := rcon.Dial("localhost"+":"+portstr, glob.RCONPass)
		if err != nil || remoteConsole == nil {
			cwlog.DoLogCW(fmt.Sprintf("Error: `%v`\n", err))
		}

		_, err = remoteConsole.Write(c + " name " + servName)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		_, err = remoteConsole.Write(c + " description " + serverDescString)
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		/* 		remoteConsole.Write(c + " max-players " + "0")
		 * 		remoteConsole.Write(c + " visibility-public " + "true")
		 * 		remoteConsole.Write(c + " visibility-steam " + "true")
		 * 		remoteConsole.Write(c + " visibility-lan " + "false")
		 * 		remoteConsole.Write(c + " require-user-verification " + "true")
		 * 		remoteConsole.Write(c + " allow-commands " + "admins-only") */
		_, err = remoteConsole.Write(c + " autosave-interval " + strconv.Itoa(autosave_interval))
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		_, err = remoteConsole.Write(c + " afk-auto-kick " + strconv.Itoa(autokick))
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
		/* 		remoteConsole.Write(c + " only-admins-can-pause " + "true")
		 * 		remoteConsole.Write(c + " autosave-only-on-server " + "true") */
		err = remoteConsole.Close()
		if err != nil {
			cwlog.DoLogCW(err.Error())
		}
	}

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(conf); err != nil {
		cwlog.DoLogCW("GenerateFactorioConfig: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		cwlog.DoLogCW("GenerateFactorioConfig: os.Create failure")
		return false
	}

	err = os.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		cwlog.DoLogCW("GenerateFactorioConfig: WriteFile failure")
		return false
	}

	err = os.Rename(tempPath, finalPath)
	if err != nil {
		cwlog.DoLogCW("GenerateFactorioConfig: Rename failure: " + err.Error() + ", " + tempPath + ", " + finalPath)
		return false
	}

	//cwlog.DoLogCW("Server settings json written.")
	return true
}
