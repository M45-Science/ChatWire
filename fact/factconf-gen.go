package fact

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"ChatWire/constants"
)

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
	Only_admins_can_pause   bool `json:"only_admins_can_pause"`
	Autosave_only_on_server bool `json:"autosave_only_on_server"`
}

func GenerateFactorioConfig() bool {

	tempPath := constants.ServSettingsName + ".tmp"
	finalPath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + constants.ServSettingsName

	servName := "\u0080 [" + cfg.Global.GroupName + "] " + strings.ToUpper(cfg.Local.ServerCallsign) + "-" + cfg.Local.Name
	servDesc := cfg.Global.GroupName + "\n" + cfg.Global.FactorioData.ServerDescription

	heartbeats := 60
	autosaves := 240
	autosave_interval := 15
	autokick := 30

	if cfg.Local.DefaultUPSRate > 0 {
		heartbeats = cfg.Local.DefaultUPSRate
	}
	if cfg.Global.FactorioData.Autosaves > 0 {
		autosaves = cfg.Global.FactorioData.Autosaves
	}
	if cfg.Local.FactorioData.Autosave_interval > 0 {
		autosave_interval = cfg.Local.FactorioData.Autosave_interval
	}

	//for pos, patreon := range glob.PlayerList {
	//patreon logic
	//}

	//Add some settings to tags, such as cheats, no blueprint, etc.

	var tags []string

	if cfg.Local.SoftModOptions.DoWhitelist {
		tags = append(tags, "MEMBERS-ONLY")
	}
	if cfg.Local.SoftModOptions.FriendlyFire {
		tags = append(tags, "NO FRIENDLY FIRE")
	}
	if cfg.Local.MapGenPreset != "" {
		tags = append(tags, "Map gen: "+cfg.Local.MapGenPreset)
	} else if cfg.Local.MapPreset != "" {
		tags = append(tags, "Map preset: "+cfg.Local.MapPreset)
	}
	if cfg.Local.ResetScheduleText != "" {
		tags = append(tags, "Map resets: "+cfg.Local.ResetScheduleText)
	}
	if cfg.Local.EnableCheats {
		tags = append(tags, "CHEATS ON")
		tags = append(tags, "Sandbox")
	}
	if cfg.Local.DisableBlueprints {
		tags = append(tags, "NO BLUEPRINTS")
	}
	if cfg.Local.SlowConnect.SlowConnect {
		tags = append(tags, "slow-connect on")
	}
	if cfg.Local.FactorioData.Autopause {
		tags = append(tags, "autopause on")
	}
	if cfg.Local.FactorioData.Autosave_interval > 0 {
		tags = append(tags, "autosaves: "+fmt.Sprintf("%dm", cfg.Local.FactorioData.Autosave_interval))
	}
	if cfg.Global.AuthServerBans {
		tags = append(tags, "Auth-server bans enabled")
	}
	if cfg.Global.FactorioData.Username != "" {
		tags = append(tags, "Owner: "+cfg.Global.FactorioData.Username)
	}
	tags = append(tags, fmt.Sprintf("%v:%v", cfg.Global.Domain, cfg.Local.Port))

	conf := FactConf{
		Comment:     "auto-generated! DO NOT MODIFY! Changes will be overwritten!",
		Name:        servName,
		Description: servDesc,
		Tags:        tags,
		Max_players: 0,
		Visibility: VisData{
			Public: true,
			Lan:    false,
			Steam:  true,
		},

		Username:                  cfg.Global.FactorioData.Username,
		Token:                     cfg.Global.FactorioData.Token,
		Require_user_verification: true,
		Max_heartbeats_per_second: heartbeats,
		Allow_commands:            "admins-only",

		Autosave_interval:       autosave_interval,
		Autosave_slots:          autosaves,
		Afk_autokick_interval:   autokick,
		Auto_pause:              cfg.Local.FactorioData.Autopause,
		Only_admins_can_pause:   true,
		Autosave_only_on_server: true,
	}

	outbuf := new(bytes.Buffer)
	enc := json.NewEncoder(outbuf)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false)

	if err := enc.Encode(conf); err != nil {
		botlog.DoLog("GenerateFactorioConfig: enc.Encode failure")
		return false
	}

	_, err := os.Create(tempPath)

	if err != nil {
		botlog.DoLog("GenerateFactorioConfig: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(tempPath, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("GenerateFactorioConfig: WriteFile failure")
		return false
	}

	err = os.Rename(tempPath, finalPath)
	if err != nil {
		botlog.DoLog("GenerateFactorioConfig: Rename failure")
		return false
	}

	botlog.DoLog("Server settings json written.")
	return true
}
