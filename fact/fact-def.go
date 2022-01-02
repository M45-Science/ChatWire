package fact

import (
	"ChatWire/constants"
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

/*Factorio stdio links */
var Pipe io.WriteCloser
var PipeLock sync.Mutex
var GameBuffer *bytes.Buffer

/* Factorio patch info */
var FactorioVersion = constants.Unknown
var NewVersion = constants.Unknown
var NewPatchName = constants.Unknown
var LastSaveName = constants.Unknown

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

/*Factorio save game data */
var GameMapName = ""
var GameMapPath = ""
var GameMapLock sync.Mutex

/* Players online */
var NumPlayers = 0
var NumPlayersLock sync.RWMutex

/* Slow-connect status */
var ConnectPauseLock sync.Mutex
var ConnectPauseTimer int64 = 0
var ConnectPauseCount = 0

/* Number of man-minutes */
var ManMinutes = 0
var ManMinutesLock sync.Mutex

/*  Map gen data */
var LastMapSeed uint64 = 0
var LastMapCode = ""
