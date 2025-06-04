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
	"ChatWire/util"
)

/* Generate a server-settings.json file for Factorio */
func GenerateFactorioConfig() bool {

	tempPath := constants.ServSettingsName + ".tmp"
	finalPath := util.GetFactorioFolder() +
		constants.ServSettingsName

	servName := "~[" + cfg.Global.GroupName + "] " + strings.ToUpper(cfg.Local.Callsign) + "-" + cfg.Local.Name

	/* Setup some defaults */
	heartbeats := 60
	autosaves := 250
	autosave_interval := 15
	autokick := 30
	maxUpload := 12500
	maxUploadSlots := 2

	if cfg.Local.Settings.Heartbeats >= 6 && cfg.Local.Settings.Heartbeats <= 240 {
		heartbeats = cfg.Local.Settings.Heartbeats
	}
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

	nextReset := FormatResetTime()
	resetInterval := FormatResetInterval()
	if nextReset != "" {
		descrLines = append(descrLines, AddFactColor("orange", "MAP RESETS: "+nextReset+" ("+resetInterval+")"))
	}

	if cfg.Local.Options.CustomWhitelist {
		descrLines = append(descrLines, AddFactColor("red", "Whitelist: invite-only"))
	} else if cfg.Local.Options.RegularsOnly {
		descrLines = append(descrLines, AddFactColor("orange", "Whitelist: regulars-only"))
	} else if cfg.Local.Options.MembersOnly {
		descrLines = append(descrLines, AddFactColor("yellow", "Whitelist: members-only"))
	} else {
		descrLines = append(descrLines, AddFactColor("green", "No whitelist: public"))
	}

	if len(disc.RoleList.Patreons) > 0 || len(disc.RoleList.Supporters) > 0 {
		supportersString := AddFactColor("purple", "Supporters: "+strings.Join(disc.RoleList.Patreons, ", ")+", "+strings.Join(disc.RoleList.Supporters, ", "))
		descrLines = append(descrLines, supportersString)
	}
	if len(disc.RoleList.NitroBooster) > 0 {
		nitrosString := AddFactColor("cyan", "Nitro Boosters: "+strings.Join(disc.RoleList.NitroBooster, ", "))
		descrLines = append(descrLines, nitrosString)
	}

	if len(cfg.Local.Options.LocalDescription) > 0 {
		ldesc := strings.Split(cfg.Local.Options.LocalDescription, "\n")
		descrLines = append(descrLines, ldesc...)
	}

	if cfg.Local.Options.SoftModOptions.FriendlyFire {
		descrLines = append(descrLines, AddFactColor("orange", "FRIENDLY FIRE: BLOCKED"))
	}

	if cfg.Global.Options.UseAuthserver {
		descrLines = append(descrLines, "Auth-server bans enabled")
	}

	if cfg.Local.Options.SoftModOptions.OneLife {
		descrLines = append(descrLines, AddFactColor("red", "Permadeath"))
		descrLines = append(descrLines, AddFactColor("red", "One-life"))
		descrLines = append(descrLines, AddFactColor("red", "No-respawn"))
	}

	if cfg.Local.Options.SoftModOptions.Cheats {
		descrLines = append(descrLines, AddFactColor("yellow", "Sandbox"))
	}

	if cfg.Local.Options.SoftModOptions.DisableBlueprints {
		descrLines = append(descrLines, AddFactColor("blue", "No blueprints"))
	}

	if !cfg.Local.Settings.AutoPause {
		descrLines = append(descrLines, AddFactColor("orange", "Auto-pause off"))
	}

	var tags []string
	if cfg.Local.Settings.MapGenerator != "" && !strings.EqualFold(cfg.Local.Settings.MapGenerator, "none") {
		descrLines = append(descrLines, "Map generator: "+cfg.Local.Settings.MapGenerator)
	} else if cfg.Local.Settings.MapPreset != "" && cfg.Local.Settings.MapPreset != "default" {
		descrLines = append(descrLines, "Map preset: "+cfg.Local.Settings.MapPreset)
	}

	if !strings.EqualFold(cfg.Local.Settings.Scenario, "none") && cfg.Local.Settings.Scenario != "" {
		descrLines = append(descrLines, "Scenario: "+cfg.Local.Settings.Scenario)
	}

	if cfg.Global.Paths.URLs.Domain != "" {
		descrLines = append(descrLines, AddFactColor("green", fmt.Sprintf("Direct connect: %v:%v", cfg.Global.Paths.URLs.Domain, cfg.Local.Port)))
	}

	if cfg.Global.Factorio.Username != "" {
		descrLines = append(descrLines, "Server owner: "+cfg.Global.Factorio.Username)
	}

	if len(cfg.Global.Options.Description) > 0 {
		gdesc := strings.Split(cfg.Global.Options.Description, "\n")
		descrLines = append(descrLines, gdesc...)
	}

	//Final string
	serverDescString := strings.Join(descrLines, "\n")

	//TAGS
	tags = append(tags, cfg.Global.GroupName)

	if cfg.Local.Options.CustomWhitelist {
		tags = append(tags, "INVITE-ONLY")
	} else if cfg.Local.Options.MembersOnly {
		tags = append(tags, "Members-Only")
	} else if cfg.Local.Options.RegularsOnly {
		tags = append(tags, "Regulars-Only")
	} else {
		tags = append(tags, "Public")
	}
	tags = append(tags, cfg.Global.Paths.URLs.Domain)
	tags = append(tags, "ChatWire")

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
		Only_admins_can_pause:   true,
		Autosave_only_on_server: true,

		Max_upload_slots:                   maxUploadSlots,
		Max_upload_in_kilobytes_per_second: maxUpload,
		Auto_pause_when_players_connect:    cfg.Local.Options.RegularsOnly,
		Non_blocking_saving:                cfg.Global.Options.NonBlockSave,
	}

	c := "/config set"
	if FactorioBooted && FactIsRunning {
		/* Send over rcon, to preserve newlines */
		portstr := fmt.Sprintf("%v", cfg.Local.Port+cfg.Global.Options.RconOffset)
		remoteConsole, err := rcon.Dial("localhost"+":"+portstr, glob.RCONPass)
		if err != nil || remoteConsole == nil {
			cwlog.DoLogCW("Error: `%v`\n", err)
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

	err := os.WriteFile(tempPath, outbuf.Bytes(), 0644)

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
