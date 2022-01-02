package glob

import (
	"os"
	"sync"
	"time"
)

/* Server status */
var ServerRunning bool = true
var Uptime time.Time

/* Discord role member-lists */
var RoleListLock sync.Mutex
var RoleList RoleListData

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

var PlayerList map[string]*PlayerData
var PlayerListLock sync.RWMutex
var PlayerListWriteLock sync.Mutex

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
