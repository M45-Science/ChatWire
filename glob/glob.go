package glob

import (
	"os"
	"sync"
	"time"

	"ChatWire/constants"
)

/* Most of these are here, because go is unable to handle import cycles */
/* I wish they would address this issue, because many people avoid packages due to this */

/* Player database */
type PlayerData struct {
	Name     string
	Level    int
	ID       string
	Creation int64
	LastSeen int64
}

/* Registrarion codes */
type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

/* Rewind Votes Container */
type RewindVoteContainerData struct {
	Version string
	Votes   []RewindVoteData

	/* Temporary storage for tallying votes */
	Tally          []VoteTallyData `json:"-"`
	LastRewindTime time.Time       `json:"-"`
	NumRewind      int

	Dirty bool `json:"-"`
}

/* Rewind Votes */
type RewindVoteData struct {
	Name       string
	DiscID     string
	TotalVotes int

	AutosaveNum int
	NumChanges  int

	Time    time.Time
	Voided  bool
	Expired bool
}

/* Temporary storage for tallying votes */
type VoteTallyData struct {
	Autosave int
	Count    int
}

type OnlinePlayerData struct {
	Name       string
	ScoreTicks int
	TimeTicks  int
	Level      int
}

var (
	OnlineCommand = constants.OnlineCommand
	OnlinePlayers []OnlinePlayerData

	/* Slash command registration */
	DoRegisterCommands   *bool
	DoDeregisterCommands *bool

	/* Vote Rewind */
	VoteBox     RewindVoteContainerData
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
	DoRebootCW     = false
	DoRebootCWLock sync.RWMutex

	/* Increasing relaunch delay */
	RelaunchThrottle     = 0
	RelaunchThrottleLock sync.RWMutex

	/* Player database */
	PlayerList          map[string]*PlayerData
	PlayerListLock      sync.RWMutex
	PlayerListWriteLock sync.Mutex

	/* Registration codes */
	PassList         map[string]*PassData
	PasswordListLock sync.RWMutex

	/* Login count */
	NumLogins     = 0
	NumLoginsLock sync.RWMutex

	/* Player database status */
	PlayerListUpdated       = false
	PlayerListUpdatedLock   sync.Mutex
	PlayerListDirty         = false
	PlayerListDirtyLock     sync.Mutex
	PlayerListSeenDirty     = false
	PlayerListSeenDirtyLock sync.Mutex

	/* Max players online record */
	RecordPlayers          = 0
	RecordPlayersWriteLock sync.Mutex
	RecordPlayersLock      sync.RWMutex

	/* Factorio server watchdog */
	NoResponseCount     = 0
	NoResponseCountLock sync.RWMutex

	/* Update warning */
	UpdateWarnCounterLock sync.Mutex
	UpdateWarnCounter     = 0
	UpdateGraceMinutes    = 15

	ChatterLock      sync.Mutex
	ChatterList      map[string]time.Time
	ChatterSpamScore map[string]int

	PlayerSusLock  sync.Mutex
	PlayerSus      map[string]int
	LastSusWarning time.Time
)
