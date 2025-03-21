package fact

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

	Username                           string `json:"username"`
	Token                              string `json:"token"`
	Require_user_verification          bool   `json:"require_user_verification"`
	Max_upload_slots                   int    `json:"max_upload_slots"`
	Max_upload_in_kilobytes_per_second int    `json:"max_upload_in_kilobytes_per_second"`
	Max_heartbeats_per_second          int    `json:"max_heartbeats_per_second"`
	Allow_commands                     string `json:"allow_commands"`

	Autosave_interval               int  `json:"autosave_interval"`
	Autosave_slots                  int  `json:"autosave_slots"`
	Afk_autokick_interval           int  `json:"afk_autokick_interval"`
	Auto_pause                      bool `json:"auto_pause"`
	Auto_pause_when_players_connect bool `json:"auto_pause_when_players_connect"`
	Only_admins_can_pause           bool `json:"only_admins_can_pause_the_game"`
	Autosave_only_on_server         bool `json:"autosave_only_on_server"`
	Non_blocking_saving             bool `json:"non_blocking_saving"`
}
