package fact

import (
	"ChatWire/botlog"
	"ChatWire/cfg"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"ChatWire/constants"
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
	Only_admins_can_pause   bool `json:"only_admins_can_pause"`
	Autosave_only_on_server bool `json:"autosave_only_on_server"`
}

/* Generate a server-settings.json file for Factorio */
func GenerateFactorioConfig() bool {

	tempPath := constants.ServSettingsName + ".tmp"
	finalPath := cfg.Global.PathData.FactorioServersRoot + cfg.Global.PathData.FactorioHomePrefix + cfg.Local.ServerCallsign + "/" + constants.ServSettingsName

	servName := "\u0080 [" + cfg.Global.GroupName + "] " + strings.ToUpper(cfg.Local.ServerCallsign) + "-" + cfg.Local.Name

	//Setup some defaults
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
	if cfg.Local.FactorioData.AutoSaveMinutes > 0 {
		autosave_interval = cfg.Local.FactorioData.AutoSaveMinutes
	}
	if cfg.Local.FactorioData.AFKKickMinutes > 0 {
		autokick = cfg.Local.FactorioData.AFKKickMinutes
	}

	//Add some settings to descrLines, such as cheats, no blueprint, etc.

	var descrLines []string

	descrLines = strings.Split(cfg.Global.FactorioData.ServerDescription, "\n")
	if cfg.Local.SoftModOptions.DoWhitelist {
		descrLines = append(descrLines, "MEMBERS-ONLY")
	}
	if cfg.Local.SoftModOptions.FriendlyFire {
		descrLines = append(descrLines, "FRIENDLY FIRE")
	}
	if cfg.Local.MapGenPreset != "" {
		descrLines = append(descrLines, "Map gen: "+cfg.Local.MapGenPreset)
	} else if cfg.Local.MapPreset != "" {
		descrLines = append(descrLines, "Map preset: "+cfg.Local.MapPreset)
	}
	if cfg.Local.ResetScheduleText != "" {
		descrLines = append(descrLines, "MAP RESETS: "+cfg.Local.ResetScheduleText)
	}
	if cfg.Local.EnableCheats {
		descrLines = append(descrLines, "SANDBOX")
	}
	if cfg.Local.DisableBlueprints {
		descrLines = append(descrLines, "NO BLUEPRINTS")
	}
	if cfg.Local.SlowConnect.SlowConnect {
		descrLines = append(descrLines, "slow-connect on")
	}
	if !cfg.Local.FactorioData.AutoPause {
		descrLines = append(descrLines, "AUTO-PAUSE OFF")
	}
	if cfg.Global.AuthServerBans {
		descrLines = append(descrLines, "AUTH-SERVER BANS ON")
	}
	if cfg.Global.FactorioData.Username != "" {
		descrLines = append(descrLines, "Owner: "+cfg.Global.FactorioData.Username)
	}
	descrLines = append(descrLines, fmt.Sprintf("%v:%v", cfg.Global.Domain, cfg.Local.Port))

	var tags []string
	tags = append(tags, cfg.Global.GroupName)

	cfg.RoleListLock.Lock()
	conf := FactConf{
		Comment:     "Auto-generated! DO NOT MODIFY! Changes will be overwritten!",
		Name:        servName,
		Description: strings.Join(descrLines, "\n") + "\n[color=purple]Patreons: " + strings.Join(cfg.RoleList.Patreons, ", ") + "[/color]\n[color=cyan]Nitro Boosters: " + strings.Join(cfg.RoleList.NitroBooster, ", ") + "[/color]\n[color=red]Moderators: " + strings.Join(cfg.RoleList.Moderators, ", ") + "[/color]\n",
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
		Auto_pause:              cfg.Local.FactorioData.AutoPause,
		Only_admins_can_pause:   true,
		Autosave_only_on_server: true,
	}
	cfg.RoleListLock.Unlock()

	c := "/config set"
	if IsFactorioBooted() {
		WriteFact(c + " name " + servName)
		//WriteFact(c + " description " + strings.Join(descrLines, " ")) //No way to do newline =/
		//		WriteFact(c + " max-players " + "0")
		//		WriteFact(c + " visibility-public " + "true")
		//		WriteFact(c + " visibility-steam " + "true")
		//		WriteFact(c + " visibility-lan " + "false")
		//		WriteFact(c + " require-user-verification " + "true")
		//		WriteFact(c + " allow-commands " + "admins-only")
		WriteFact(c + " autosave-interval " + strconv.Itoa(autosave_interval))
		WriteFact(c + " afk-auto-kick " + strconv.Itoa(autokick))
		//		WriteFact(c + " only-admins-can-pause " + "true")
		//		WriteFact(c + " autosave-only-on-server " + "true")
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
