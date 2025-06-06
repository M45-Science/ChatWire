package constants

import "time"

const (
	ProgName           = "ChatWire"
	Version            = "2684-06.06.2025-0253p"
	CWEpoch            = 1653239822390688174 //Never change this
	SeenDivisor        = 60
	SeenEpoch          = 1546326000
	Unknown            = "Unknown"
	MaxWhitelist       = 1500 //Max whitelist items
	MaxNameLength      = 64
	MaxBanReasonLength = 1024
	PassExpireSec      = 5 * 60

	/* ChatWire files */
	CWGlobalConfig      = "../cw-global-config.json"
	CWLocalConfig       = "cw-local-config.json"
	WhitelistName       = "server-whitelist.json"
	AdminlistName       = "server-adminlist.json"
	ServSettingsName    = "server-settings.json"
	ModSettingsName     = "mod-settings.dat"
	ModListName         = "mod-list.json"
	ModVersionsName     = "mod-versions.json"
	ModsQueueFolder     = "mods-queue"
	ModsFolder          = "mods"
	OldModsDir          = "old"
	RoleListFile        = "../RoleList.dat"
	VoteFile            = "votes.dat"
	ArchiveFolderSuffix = " maps"
	TempSaveName        = "softmod.tmp"
	BootUpdateDelayMin  = 2
	SusWarningInterval  = 2
	CrashReportInterval = 300
	MaxSaveBackups      = 10
	ModPackLifeMins     = 180
	ModPackCooldownMin  = 5
	MaxModPacks         = 4
	ErrMsgDelay         = time.Second * 3
	// Default offset added to Factorio server port for RCON
	RconPortOffset = 10000

	// Minimum bytes required in level.dat0 to consider a save valid
	LevelDatMinSize = 50 * 1024

	/* Spam auto-ban settings */
	SpamScoreLimit   = 30
	SpamScoreWarning = 15

	SpamSlowThres = time.Second * 2
	SpamFastThres = time.Millisecond * 500

	SpamCoolThres  = time.Second * 6
	SpamResetThres = time.Second * 15

	/* Player suspect settings */
	SusWarningThresh = 40
	VeteranThresh    = (60 * 24 * 3) //3 days

	/* Online commands */
	OnlineCommand    = "/p o c"
	SoftModOnlineCMD = "/online"

	/* ChatWire settings */
	PauseThresh              = 5  /* Number of repeated time reports before we assume server is paused */
	ErrorDelayShutdown       = 30 /* If we close on error, sleep this long before exiting */
	RestartLimitMinutes      = 5  /* If cw.lock is newer than this, sleep */
	RestartLimitSleepMinutes = 2  /* cw.lock is new, sleep this long then exit. */

	/* Vote Map */
	VotesNeeded       = 2  /* Number of votes needed */
	MapCooldownMins   = 1  /* Cooldown */
	VoteCooldown      = 5  /* Vote cooldown */
	VoteExpire        = 6  /* Vote expires after this number of hours */
	MaxVoteChanges    = 2  /* Max number of times a player can change their vote */
	MaxVotesPerMap    = 10 /* Max number of votes per map */
	MaxMapResults     = 25 /* 25 is Discord max 6/2022, we add one to list for 'new' */
	MaxFullMapResults = 500

	/* Max results to return */
	WhoisResults = 15

	/* Maximum time to wait for Factorio to close */
	MaxFactorioCloseWait = 45 * 10 //Loop sleep is 1/10 of a second

	/* How often to check if Factorio server is alive */
	WatchdogInterval = time.Second

	/* Throttle Discord chat */
	CMSRate            = 500 * time.Millisecond  //Time we spend waiting for buffer to fill up once active
	CMSRestTime        = 3500 * time.Millisecond //Time to sleep after sending a message
	CMSPollRate        = 100 * time.Millisecond  //Time between polls
	MaxDiscordAttempts = 90

	/* Discord limits */
	MaxDiscordMsgLen = 2000

	/* Auto port assignment */
	AlphaStartValue = 10000

	/* Mod history settings */
	ModHistoryKeyStart = 10000
	ModHistoryMaxKey   = 99999
	MaxModHistory      = 250
	ModHistoryPageSize = 25

	/* Moderator uploads */
	MaxModSettingsSize = 1024 * 1024 //1MB
	MaxModListSize     = 1024 * 1024 //1MB
)

/* Factorio map preset names */
var MapTypes = []string{"default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island", "lakes"}
