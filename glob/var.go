package glob

import (
	"context"
	"os"
	"os/exec"
	"sync"
	"time"

	"ChatWire/constants"

	"github.com/bwmarrin/discordgo"
)

var (
	FactorioLock  sync.Mutex
	UpdatersLock  sync.Mutex
	BootMessage   *discordgo.Message
	UpdateMessage *discordgo.Message

	FactorioCmd     *exec.Cmd
	FactorioContext context.Context
	FactorioCancel  context.CancelFunc

	AlphaValue    map[string]int
	RCONPass      string
	OnlineCommand = constants.OnlineCommand
	OnlinePlayers []OnlinePlayerData

	/* Boot flags */
	DoRegisterCommands   *bool
	DoDeregisterCommands *bool
	LocalTestMode        *bool
	NoAutoLaunch         *bool
	NoDiscord            *bool
	ProxyURL             *string
	MigrateJSONToSQLite  *bool
	MigrateSQLiteToJSON  *bool

	/* Vote map */
	VoteBox     VoteContainerData
	VoteBoxLock sync.Mutex

	/* Server status */
	ServerRunning  bool = true
	Uptime         time.Time
	SoftModVersion = constants.Unknown

	/* Log data */
	GameLogName = ""
	CWLogName   = ""
	GameLogDesc *os.File
	CWLogDesc   *os.File

	/* CW reboot */
	DoRebootCW = false

	/* Increasing relaunch delay */
	RelaunchThrottle = 0

	/* Player database */
	PlayerList          map[string]*PlayerData
	PlayerListLock      sync.RWMutex
	PlayerListWriteLock sync.Mutex

	/* Registration codes */
	PassList         map[string]*PassData
	PasswordListLock sync.RWMutex

	/* Player database status */
	PlayerListUpdated       = false
	PlayerListUpdatedLock   sync.Mutex
	PlayerListDirty         = false
	PlayerListDirtyLock     sync.Mutex
	PlayerListSeenDirty     = false
	PlayerListSeenDirtyLock sync.Mutex

	/* Global config status */
	GlobalCfgUpdated     = false
	GlobalCfgUpdatedLock sync.Mutex

	/* Factorio server watchdog */
	NoResponseCount = 0

	/* Update warning */
	UpdateWarnCounter  = 0
	UpdateGraceMinutes = 5
	UpdateZipAttempts  = 0

	/* Chat spam */
	ChatterLock      sync.Mutex
	ChatterList      map[string]time.Time
	ChatterSpamScore map[string]int

	//Message throttles
	LastSusWarning  time.Time
	LastCrashReport time.Time

	/* Pause command */
	PausedForConnect     bool
	PausedCount          int
	PausedAt             time.Time
	PausedConnectAttempt bool
	PausedFor            string
	PausedLock           sync.Mutex
)
