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

var ServerRunning bool = true
var Uptime time.Time

var Guild *discordgo.Guild
var Guildname = constants.Unknown
var GuildLock sync.RWMutex

type CMSBuf struct {
	Added   time.Time
	Channel string
	Text    string
}

var CMSBuffer []CMSBuf
var CMSBufferLock sync.Mutex

var Pipe io.WriteCloser
var PipeLock sync.Mutex

var DS *discordgo.Session

var FactorioVersion = constants.Unknown
var NewVersion = constants.Unknown
var NewPatchName = constants.Unknown
var LastSaveName = constants.Unknown

var OldChanName = constants.Unknown
var NewChanName = constants.Unknown
var GameLogName = ""
var BotLogName = ""
var GameLogDesc *os.File
var BotLogDesc *os.File
var GameBuffer *bytes.Buffer

var GameMapName = ""
var GameMapPath = ""
var GameMapLock sync.Mutex

var ModLoadMessage *discordgo.Message
var ModLoadString = constants.Unknown
var ModLoadLock sync.RWMutex

var LastGametime = ""
var PausedTicks = 0
var PausedTicksLock sync.RWMutex

var Gametime = constants.Unknown
var GametimeString = constants.Unknown
var GametimeLock sync.RWMutex

var FactIsRunning = false
var FactIsRunningLock sync.RWMutex

var FactAutoStart = false
var FactAutoStartLock sync.RWMutex

var QueueReload = false
var QueueReloadLock sync.RWMutex

var DoRebootBot = false
var DoRebootBotLock sync.RWMutex

var RecordPlayers = 0
var NumPlayers = 0
var NumPlayersLock sync.RWMutex

var RelaunchThrottle = 0
var RelaunchThrottleLock sync.RWMutex

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

type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

var PassList map[string]*PassData

var NumLogins = 0
var NumLoginsLock sync.RWMutex
var UpdateChannelLock sync.Mutex

var FactorioBooted = false
var FactorioBootedAt time.Time
var FactorioBootedLock sync.RWMutex

var PlayerListUpdated = false
var PlayerListUpdatedLock sync.Mutex

var PlayerListDirty = false
var PlayerListDirtyLock sync.Mutex

var PlayerListSeenDirty = false
var PlayerListSeenDirtyLock sync.Mutex

var RecordPlayersWriteLock sync.Mutex
var RecordPlayersLock sync.RWMutex

var PasswordListLock sync.RWMutex

var NoResponseCount = 0
var NoResponseCountLock sync.RWMutex

var MaxMapTypes = 0
var LastMapSeed uint64 = 0
var LastMapCode = ""
var FactorioLaunchLock sync.Mutex

var UpdateFactorioLock sync.Mutex
var DoUpdateFactorio = false
var DoUpdateFactorioLock sync.Mutex

var ManMinutes = 0
var ManMinutesLock sync.Mutex

var LastColor = 0

var ConnectPauseLock sync.Mutex
var ConnectPauseTimer int64 = 0
var ConnectPauseCount = 0

var LastTotalStat = ""
var LastMemberStat = ""
var LastRegularStat = ""

var UpdateWarnCounterLock sync.Mutex
var UpdateWarnCounter = 0
var UpdateGraceMinutes = 10
