package glob

import (
	"os"
	"sync"
	"time"
)

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

	//Temporary storage for tallying votes
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

/* Vote Rewind */
var VoteBox RewindVoteContainerData
var VoteBoxLock sync.Mutex

/* Server status */
var ServerRunning bool = true
var Uptime time.Time

/* Log data */
var GameLogName = ""
var BotLogName = ""
var GameLogDesc *os.File
var BotLogDesc *os.File

/* Bot-reboot */
var DoRebootBot = false
var DoRebootBotLock sync.RWMutex

/* Increasing relaunch delay */
var RelaunchThrottle = 0
var RelaunchThrottleLock sync.RWMutex

/* Player database */
var PlayerList map[string]*PlayerData
var PlayerListLock sync.RWMutex
var PlayerListWriteLock sync.Mutex

/* Registration codes */
var PassList map[string]*PassData
var PasswordListLock sync.RWMutex

/* Login count */
var NumLogins = 0
var NumLoginsLock sync.RWMutex

/* Player database status */
var PlayerListUpdated = false
var PlayerListUpdatedLock sync.Mutex
var PlayerListDirty = false
var PlayerListDirtyLock sync.Mutex
var PlayerListSeenDirty = false
var PlayerListSeenDirtyLock sync.Mutex

/* Max players online record */
var RecordPlayers = 0
var RecordPlayersWriteLock sync.Mutex
var RecordPlayersLock sync.RWMutex

/* Factorio server watchdog */
var NoResponseCount = 0
var NoResponseCountLock sync.RWMutex

/* Random color */
var LastColor = 0

/* Member-count status */
var LastTotalStat = ""
var LastMemberStat = ""
var LastRegularStat = ""

/* Update warning */
var UpdateWarnCounterLock sync.Mutex
var UpdateWarnCounter = 0
var UpdateGraceMinutes = 10
