package cfg

import "time"

/* GLOBAL CONFIG */
type global struct {
	GroupName     string `form:"RO" web:"Group Name"`
	PrimaryServer string `form:"RO" web:"Primary Server"`
	Discord       discord
	Factorio      factData

	Paths   dataPaths `form:"-"`
	Options globalOptions
}

type discord struct {
	Token       string `form:"hidden"`
	Guild       string `form:"RO"`
	Application string `form:"hidden"`

	ReportChannel   string `form:"RO" web:"Report Channel ID"`
	AnnounceChannel string `form:"RO" web:"Announce Channel ID"`
	Roles           roles
	Comment         string `form:"-"`
	SusPingRole     string `web:"Suspicious Ping RoleID"`
}

type roles struct {
	Comment   string `form:"-"`
	Admin     string `form:"RO"`
	Moderator string `form:"RO"`
	Regular   string
	Veteran   string
	Member    string
	New       string

	Patreon   string
	Supporter string
	Nitro     string

	RoleCache roleCache `form:"-"`
}

type roleCache struct {
	Comment   string `form:"-"`
	Admin     string
	Moderator string
	Regular   string
	Veteran   string
	Member    string
	New       string

	Patreon   string
	Supporter string
	Nitro     string
}

type factData struct {
	Username string `form:"RO"`
	Token    string `form:"hidden"`
}

type dataPaths struct {
	ChatWirePrefix string
	Folders        folderPaths
	Binaries       binaryPaths
	URLs           urlPaths
	DataFiles      dataFiles
}

type folderPaths struct {
	FactorioDir   string
	ServersRoot   string
	Saves         string
	Mods          string
	MapGenerators string
	MapArchives   string
	ModPack       string
}

type binaryPaths struct {
	FactBinary string
}

type urlPaths struct {
	Domain      string `form:"RO" web:"Domain Name"`
	PathPrefix  string `form:"RO"`
	LogPath     string `form:"RO"`
	LogsPathWeb string `form:"RO"`
	ArchivePath string `form:"RO"`
	ModPackPath string `form:"RO"`
}

type dataFiles struct {
	DBFile string `form:"-"`
	Bans   string `form:"-"`
}

type globalOptions struct {
	Description        string `web:"Factorio Description"`
	Comment            string `form:"-"`
	ResetPingRole      string `form:"-"`
	UseAuthserver      bool   `form:"RO" web:"Use Authserver Bans"`
	AutosaveMax        int    `form:"RO" web:"Max Autosaves"`
	RconOffset         int    `form:"RO" web:"Rcon Port Offset"`
	ShutupSusWarn      bool   `web:"Suspicious Warning Mute"`
	DisableSpamProtect bool   `form:"RO" web:"Disable Chat Spam AutoBan"`
	FCLWarnOnly        bool   `web:"FCL-Ban Warn-Only"`
	FCLWarnRegulars    bool   `web:"FCL-Ban Warning For Regulars"`
	NonBlockSave       bool
}

/* LOCAL CONFIG */
type local struct {
	Callsign       string `form:"RO"`
	Name           string
	Port           int    `form:"RO"`
	RCONPass       string `json:"-"`
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
	NewMap         bool   `json:"-"`
	Scenario       string `web:"Scenario Name"`
	MapGenerator   string `web:"Map Generator Name"`
	MapPreset      string `web:"Map Preset Name"`
	Seed           int    `web:"Map Seed"`
	AFKMin         int    `web:"AFK Minutes"`
	AutosaveMin    int    `web:"Autosave Minutes"`
	AutoPause      bool   `web:"Pause When Empty"`
	AdminOnlyPause bool   `web:"Admin-Only Pause"`
	Heartbeats     int    `web:"Heartbeats per second"`
}

type channel struct {
	Comment     string `form:"-"`
	ChatChannel string `web:"Channel ID"`
}

type ResetInterval struct {
	Months, Weeks, Days, Hours int
}

type localOptions struct {
	LocalDescription string `web:"Factorio Description"`

	ResetInterval ResetInterval `web:"Reset interval"`
	NextReset     time.Time     `web:"Next map reset"`
	ResetHour     int           `web:"Map Reset Hour"`
	SkipReset     bool          `web:"Skip Map Reset"`

	ResetPingRole  string `form:"-"`
	PlayHourEnable bool   `web:"Limit Open Hours"`
	PlayStartHour  int    `web:"Open Hour"`
	PlayEndHour    int    `web:"Close Hour"`

	AutoStart       bool    `web:"Auto Boot/Reboot Factorio"`
	AutoUpdate      bool    `web:"Auto Factorio Updates"`
	ExpUpdates      bool    `web:"Factorio Experimental Updates"`
	HideAutosaves   bool    `web:"Hide Autosaves"`
	HideResearch    bool    `web:"Hide Research on Discord"`
	RegularsOnly    bool    `web:"Regulars, Veterans only"`
	MembersOnly     bool    `web:"Members, Regulars, Veterans only"`
	CustomWhitelist bool    `web:"Private whitelist"`
	ModUpdate       bool    `web:"Auto update Factorio Mods"`
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
