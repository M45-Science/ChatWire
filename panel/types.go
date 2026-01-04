package panel

type panelCmd struct {
	Cmd   string
	Label string
	Icon  string
}

type panelSave struct {
	Name string
	Age  string
}

type panelCommand struct {
	Name        string
	Description string
}

type panelCmdGroup struct {
	Name string
	Cmds []panelCmd
}

type panelData struct {
	ServerName    string
	Callsign      string
	CWVersion     string
	Factorio      string
	SoftMod       string
	Players       int
	Gametime      string
	SaveName      string
	UPS           string
	CWUp          string
	FactUp        string
	NextReset     string
	TimeTill      string
	ResetInterval string
	Total         int
	Mods          int
	Banned        int
	Mem           int
	Reg           int
	Vet           int
	AccessLevel   int
	ModNames      []string
	PlayHours     string
	HoursEnabled  bool
	Paused        bool
	FactRunning   bool
	MapSchedule   bool
	Token         string
	CmdGroups     []panelCmdGroup
	Saves         []panelSave
	Commands      []panelCommand
	Info          string
	LocalCfg      string
	GlobalCfg     string
	LocalJSON     string
}

var modCmdGroups = []panelCmdGroup{
	{
		Name: "ChatWire",
		Cmds: []panelCmd{
			{Cmd: "reboot-chatwire", Label: "Reboot ChatWire", Icon: "restart_alt"},
			{Cmd: "queue-reboot", Label: "Queue ChatWire Reboot", Icon: "schedule"},
			{Cmd: "force-reboot", Label: "Force ChatWire Reboot", Icon: "restart_alt"},
			{Cmd: "queue-fact-reboot", Label: "Queue Factorio Reboot", Icon: "schedule"},
			{Cmd: "reload-config", Label: "Reload Config", Icon: "refresh"},
		},
	},
	{
		Name: "Factorio",
		Cmds: []panelCmd{
			{Cmd: "start-factorio", Label: "Start Factorio", Icon: "play_arrow"},
			{Cmd: "stop-factorio", Label: "Stop Factorio", Icon: "stop"},
			{Cmd: "install-factorio", Label: "Install Factorio", Icon: "download"},
			{Cmd: "update-factorio", Label: "Update Factorio", Icon: "update"},
			{Cmd: "new-map", Label: "New Map", Icon: "create_new_folder"},
			{Cmd: "archive-map", Label: "Archive Map", Icon: "archive"},
			{Cmd: "map-reset", Label: "Reset Map", Icon: "map"},
		},
	},
	{
		Name: "Mods",
		Cmds: []panelCmd{
			{Cmd: "update-mods", Label: "Update Mods", Icon: "system_update_alt"},
			{Cmd: "sync-mods", Label: "Sync Mods", Icon: "sync"},
		},
	},
}
