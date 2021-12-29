package glob

import (
	"bytes"
	"io"
	"os"
	"sync"
	"time"

	"ChatWire/constants"

	"github.com/bwmarrin/discordgo"
)

/* Server status */
var ServerRunning bool = true
var Uptime time.Time

/* Discord data */
var Guild *discordgo.Guild
var Guildname = constants.Unknown
var GuildLock sync.RWMutex
var DS *discordgo.Session

/* To-Discord message buffer */
type CMSBuf struct {
	Added   time.Time
	Channel string
	Text    string
}

/* Discord role member-lists */
var PatreonList []string
var PatreonSent bool
var PatreonLock sync.Mutex

var Nitrolist []string
var NitroSent bool
var NitroLock sync.Mutex

var ModeratorList []string
var ModeratorSent bool
var ModeratorLock sync.Mutex

/*To-Discord message buffer*/
var CMSBuffer []CMSBuf
var CMSBufferLock sync.Mutex

/*F actorio stdio links */
var Pipe io.WriteCloser
var PipeLock sync.Mutex
var GameBuffer *bytes.Buffer

/* Factorio patch info */
var FactorioVersion = constants.Unknown
var NewVersion = constants.Unknown
var NewPatchName = constants.Unknown
var LastSaveName = constants.Unknown

/* Channel name data */
var OldChanName = constants.Unknown
var NewChanName = constants.Unknown
var UpdateChannelLock sync.Mutex

/* Log data */
var GameLogName = ""
var BotLogName = ""
var GameLogDesc *os.File
var BotLogDesc *os.File

/*Factorio save game data */
var GameMapName = ""
var GameMapPath = ""
var GameMapLock sync.Mutex

/* Factorio game mod data*/
var ModLoadMessage *discordgo.Message
var ModLoadString = constants.Unknown
var ModLoadLock sync.RWMutex

/* Factorio in-game time data */
var LastGametime = ""
var PausedTicks = 0
var PausedTicksLock sync.RWMutex
var Gametime = constants.Unknown
var GametimeString = constants.Unknown
var GametimeLock sync.RWMutex

/* Factorio status */
var FactIsRunning = false
var FactIsRunningLock sync.RWMutex
var FactorioBooted = false
var FactorioBootedAt time.Time
var FactorioBootedLock sync.RWMutex
var FactorioLaunchLock sync.Mutex
var UpdateFactorioLock sync.Mutex
var DoUpdateFactorio = false
var DoUpdateFactorioLock sync.Mutex

/* Factorio autostart */
var FactAutoStart = false
var FactAutoStartLock sync.RWMutex

/* Reboot-when-empty */
var QueueReload = false
var QueueReloadLock sync.RWMutex

/* Bot-reboot */
var DoRebootBot = false
var DoRebootBotLock sync.RWMutex

/* Players online */
var RecordPlayers = 0
var NumPlayers = 0
var NumPlayersLock sync.RWMutex

/* Increasing relaunch delay */
var RelaunchThrottle = 0
var RelaunchThrottleLock sync.RWMutex

/* Player database */
type PlayerData struct {
	Name     string
	Level    int
	ID       string
	Creation int64
	LastSeen int64
}

var PlayerList map[string]*PlayerData
var PlayerListLock sync.RWMutex
var PlayerListWriteLock sync.Mutex

/* Registrarion codes */
type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

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
var RecordPlayersWriteLock sync.Mutex
var RecordPlayersLock sync.RWMutex

/* Factorio server watchdog */
var NoResponseCount = 0
var NoResponseCountLock sync.RWMutex

/*  Map gen data */
var MaxMapTypes = 0
var LastMapSeed uint64 = 0
var LastMapCode = ""

/* Number of man-minutes */
var ManMinutes = 0
var ManMinutesLock sync.Mutex

/* Random color */
var LastColor = 0

/* Slow-connect status */
var ConnectPauseLock sync.Mutex
var ConnectPauseTimer int64 = 0
var ConnectPauseCount = 0

/* Member-count status */
var LastTotalStat = ""
var LastMemberStat = ""
var LastRegularStat = ""

/* Update warning */
var UpdateWarnCounterLock sync.Mutex
var UpdateWarnCounter = 0
var UpdateGraceMinutes = 10
