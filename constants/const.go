package constants

import "time"

const (
	Version = "529-03.21.2022-0418"
	Unknown = "Unknown"

	/* ChatWire files */
	CWGlobalConfig   = "../cw-global-config.json"
	CWLocalConfig    = "cw-local-config.json"
	WhitelistName    = "server-whitelist.json"
	ServSettingsName = "server-settings.json"
	ModsQueueFolder  = "mods-queue"
	ModsFolder       = "mods"
	RoleListFile     = "../RoleList.dat"
	VoteRewindFile   = "vote-rewind.dat"

	/* ChatWire settings */
	PauseThresh              = 5  /* Number of repeated time reports before we assume server is paused */
	ErrorDelayShutdown       = 30 /* If we close on error, sleep this long before exiting */
	RestartLimitMinutes      = 5  /* If cw.lock is newer than this, sleep */
	RestartLimitSleepMinutes = 2  /* cw.lock is new, sleep this long then exit. */

	/* Vote Rewind */
	VotesNeededRewind     = 2 /* Number of votes needed to rewind */
	RewindCooldownMinutes = 1 /* Cooldown between rewinds */
	VoteLifetime          = 5 /* How long a vote lasts */
	MaxRewindChanges      = 2 /* Max number of times a player can change their vote */
	MaxVotesPerMap        = 4 /* Max number of votes per map */
	MaxRewindResults      = 20

	/* Max results to return */
	WhoisResults      = 20
	AdminWhoisResults = 20

	/* Maximum time to wait for Factorio update download */
	FactorioUpdateCheckLimit = 10 * time.Minute

	/* Maximum time before giving up on patching */
	FactorioUpdateProcessLimit = 5 * time.Minute

	/* Maximum time before giving up on checking zipfile integrity */
	ZipIntegrityLimit = 1 * time.Minute

	/* Maximum time before giving up on mod updater */
	ModUpdateLimit = 10 * time.Minute
	ModUpdaterPath = "scripts/mod_updater.py"

	/* Maximum time to wait for Factorio to close */
	MaxFactorioCloseWait = 30

	/* How often to check if Factorio server is alive */
	WatchdogInterval = time.Second

	/* Throttle chat, 1.5 seconds per message. */
	CMSRate     = 500 * time.Millisecond  //Time we spend waiting for buffer to fill up once active
	CMSRestTime = 1000 * time.Millisecond //Time to sleep after sending a message
	CMSPollRate = 100 * time.Millisecond  //Time between polls

	/* Used for chat colors in-game */
	NumColors = 17
)

var Colors = [...]struct {
	R float32
	G float32
	B float32
}{
	{1, 0, 0},          /* RED */
	{1, 0.25, 0},       /* RED-ORANGE */
	{1, 0.5, 0},        /* ORANGE */
	{1, 0.66, 0},       /* ORANGE-YELLOW */
	{1, 1, 0},          /* YELLOW */
	{0.66, 1, 0},       /* YELLOW-GREEN */
	{0, 1, 0},          /* GREEN */
	{0, 1, 0.66},       /* GREEN-BLUE */
	{0, 1, 1},          /* CYAN */
	{0, 0.66, 1},       /* CYAN-BLUE */
	{0, 0, 1},          /* BLUE */
	{0.33, 0, 1},       /* BLUE-PURPLE */
	{0.66, 0, 1},       /* PURPLE */
	{0.85, 0, 1},       /* PURPLE-MAGENTA */
	{1, 0, 1},          /* MAGENTA */
	{1, 0, 0.66},       /* MAGENTA-RED */
	{0.66, 0.66, 0.66}, /* GRAY */
}

/* Factorio map preset names */
var MapTypes = []string{"custom", "default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island"}
