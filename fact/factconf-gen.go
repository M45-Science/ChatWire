package fact

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type VisData struct {
	Public bool `json:"public"`
	Lan    bool `json:"lan"`
	Steam  bool `json:"steam"`
}

type FactConf struct {
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

	servName := " [" + cfg.Global.GroupName + "] " + cfg.Local.ServerCallsign + "-" + cfg.Local.Name
	path := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/server-settings.json"

	heartbeats := 60
	autosaves := 240
	autosave_interval := 15
	autokick := 30
	resetSchedule := "Manually Reset"
	mapPreset := cfg.Local.MapPreset

	if cfg.Local.MapGenPreset != "" {
		mapPreset = cfg.Local.MapGenPreset
	}
	if cfg.Local.DefaultUPSRate > 0 {
		heartbeats = cfg.Local.DefaultUPSRate
	}
	if cfg.Global.FactorioData.Autosaves > 0 {
		autosaves = cfg.Global.FactorioData.Autosaves
	}
	if cfg.Local.FactorioData.Autosave_interval > 0 {
		autosave_interval = cfg.Local.FactorioData.Autosave_interval
	}
	if cfg.Local.ResetScheduleText != "" {
		resetSchedule = cfg.Local.ResetScheduleText
	}

	//for pos, patreon := range glob.PlayerList {
	//patreon logic
	//}

	conf := FactConf{
		Name:        servName,
		Description: cfg.Global.GroupName + "\n" + cfg.Global.FactorioData.ServerDescription,
		Tags: []string{
			cfg.Global.GroupName,
			cfg.Local.Name,
			mapPreset,
			resetSchedule,
			fmt.Sprintf("%v UPS", heartbeats),
		},
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

	_, err := os.Create(path)

	if err != nil {
		botlog.DoLog("GenerateFactorioConfig: os.Create failure")
		return false
	}

	err = ioutil.WriteFile(path, outbuf.Bytes(), 0644)

	if err != nil {
		botlog.DoLog("GenerateFactorioConfig: WriteFile failure")
		return false
	}

	botlog.DoLog("Server settings json written.")
	return true
}
