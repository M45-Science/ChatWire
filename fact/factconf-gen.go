package fact

import (
	"ChatWire/cfg"
	"fmt"
)

type factConf struct {
	name        string
	description string
	tags        []string

	username                  string
	token                     string
	require_user_verivation   bool
	max_heartbeats_per_second int
	allow_commands            string

	autosave_interval       int
	autosave_slots          int
	afk_autokick_interval   int
	auto_pause              bool
	only_admins_can_pause   bool
	autosave_only_on_server bool
}

func GenerateFactorioConfig() {

	heartbeats := 60
	autosaves := 240
	autosave_interval := 15
	autokick := 30
	resetSchedule := "Manually Reset"

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

	conf := factConf{
		name:        "Factorio",
		description: "",
		tags: []string{
			cfg.Global.GroupName,
			cfg.Local.MapGenPreset,
			cfg.Local.Name,
			resetSchedule,
			fmt.Sprintf("%v UPS", heartbeats),
		},

		username:                  cfg.Global.FactorioData.Username,
		token:                     cfg.Global.FactorioData.Token,
		require_user_verivation:   true,
		max_heartbeats_per_second: heartbeats,
		allow_commands:            "admins-only",

		autosave_interval:       autosave_interval,
		autosave_slots:          autosaves,
		afk_autokick_interval:   autokick,
		auto_pause:              cfg.Local.FactorioData.Autopause,
		only_admins_can_pause:   true,
		autosave_only_on_server: true,
	}

	//Write file
}
