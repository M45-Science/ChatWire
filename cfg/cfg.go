package cfg

var Local config
var Global gconfig

type config struct {
	ServerCallsign string

	ChannelData ChannelDataStruct

	MapPreset     string
	AutoStart     bool
	AutoUpdate    bool
	DoWhitelist   bool
	UpdateFactExp bool

	MapGenPreset string

	RestrictMode   bool
	FriendlyFire   bool
	CleanMapOnBoot bool
	ShowStats      bool

	SlowConnect  bool
	DefaultSpeed float32
	ConnectSpeed float32

	ResetScheduleText string
}

type gconfig struct {
	DiscordData    DiscordDataStruct
	AdminData      AdminIDsStruct
	RoleData       RoleDataStruct
	PathData       PathDataStruct
	MapPreviewData MapPreviewDataStruct

	FactorioLaunchParams []string
	DiscordCommandPrefix string
}

//Global
//bor = based on root
//boh = based on home
//ap = absolute path
type PathDataStruct struct {
	FactorioServersRoot string //root of factorio server
	FactorioHomePrefix  string //per-server
	FactorioBinary      string

	RecordPlayersFilename string //boh
	SaveFilePath          string //boh

	ScriptInserterPath string //bor
	DBFileName         string //bor
	LogCompScriptPath  string //bor
	FactUpdaterPath    string //bor
	FactUpdateCache    string //bor
	MapGenPath         string //bor

	MapPreviewPath  string //ap
	MapArchivePath  string //ap
	ImageMagickPath string //ap
	ShellPath       string //ap
	ZipBinaryPath   string //ap
	MapPreviewURL   string
}

type DiscordDataStruct struct {
	Token   string
	GuildID string

	StatTotalChannelID    string
	StatMemberChannelID   string
	StatRegularsChannelID string

	ReportChannelID   string
	AnnounceChannelID string
}

type AdminIDsStruct struct {
	Admins []AdminStruct
}

type AdminStruct struct {
	Name string
	ID   string
}

type RoleDataStruct struct {
	Admins   string
	Regulars string
	Members  string
}

type MapPreviewDataStruct struct {
	Args       string
	Res        string
	Scale      string
	JPGQuality string
	JPGScale   string
}

//Local
type ChannelDataStruct struct {
	Name   string
	Pos    int
	ChatID string
	LogID  string
}
