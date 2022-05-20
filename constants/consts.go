package constants

import "time"

const (
	Version = "534-05.05.2022-0556p"
	Unknown = "Unknown"

	/* ChatWire files */
	CWGlobalConfig      = "../cw-global-config.json"
	CWLocalConfig       = "cw-local-config.json"
	WhitelistName       = "server-whitelist.json"
	ServSettingsName    = "server-settings.json"
	ModsQueueFolder     = "mods-queue"
	ModsFolder          = "mods"
	RoleListFile        = "../RoleList.dat"
	VoteRewindFile      = "vote-rewind.dat"
	MembersPrefix       = "M"
	ArchiveFolderSuffix = " maps"

	/* ChatWire settings */
	PauseThresh              = 5  /* Number of repeated time reports before we assume server is paused */
	ErrorDelayShutdown       = 30 /* If we close on error, sleep this long before exiting */
	RestartLimitMinutes      = 5  /* If cw.lock is newer than this, sleep */
	RestartLimitSleepMinutes = 2  /* cw.lock is new, sleep this long then exit. */

	/* Vote Rewind */
	VotesNeededRewind     = 2      /* Number of votes needed to rewind */
	RewindCooldownMinutes = 1      /* Cooldown between rewinds */
	VoteLifetime          = 60 * 3 /* How long a vote lasts */
	MaxRewindChanges      = 2      /* Max number of times a player can change their vote */
	MaxVotesPerMap        = 4      /* Max number of votes per map */
	MaxRewindResults      = 20

	/* Max results to return */
	WhoisResults      = 20

	/* Maximum time to wait for Factorio update download */
	FactorioUpdateCheckLimit = 15 * time.Minute

	/* Maximum time before giving up on patching */
	FactorioUpdateProcessLimit = 10 * time.Minute

	/* Maximum time before giving up on checking zipfile integrity */
	ZipIntegrityLimit = 2 * time.Minute

	/* Maximum time before giving up on mod updater */
	ModUpdateLimit = 10 * time.Minute
	ModUpdaterPath = "scripts/mod_updater.py"

	/* Maximum time to wait for Factorio to close */
	MaxFactorioCloseWait = 45 * 10 //Loop sleep is 1/10 of a second

	/* How often to check if Factorio server is alive */
	WatchdogInterval = time.Second

	/* Throttle chat, 1.5 seconds per message. */
	CMSRate                  = 500 * time.Millisecond  //Time we spend waiting for buffer to fill up once active
	CMSRestTime              = 2000 * time.Millisecond //Time to sleep after sending a message
	CMSPollRate              = 100 * time.Millisecond  //Time between polls
	MaxDiscordAttempts       = 100
	ApplicationCommmandSleep = time.Millisecond * 100
)

/* Factorio map preset names */
var MapTypes = []string{"default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island"}
