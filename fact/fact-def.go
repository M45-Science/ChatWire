package fact

import (
	"ChatWire/constants"
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	/*Factorio stdio links */
	Pipe       io.WriteCloser
	PipeLock   sync.Mutex
	GameBuffer *bytes.Buffer

	/* Factorio patch info */
	FactorioVersion = constants.Unknown
	NewVersion      = constants.Unknown
	NewPatchName    = constants.Unknown
	LastSaveName    = constants.Unknown

	/* Factorio game mod data*/
	ModLoadMessage *discordgo.Message
	ModLoadString  = constants.Unknown
	ModLoadLock    sync.RWMutex

	/* Factorio in-game time data */
	LastGametime    = ""
	PausedTicks     = 0
	PausedTicksLock sync.RWMutex
	Gametime        = constants.Unknown
	GametimeString  = constants.Unknown
	GametimeLock    sync.RWMutex

	MaxTickHistory = 4000

	TickHistory     []TickInt
	TickHistoryLock sync.Mutex

	/* Factorio status */
	FactIsRunning        = false
	FactIsRunningLock    sync.RWMutex
	FactorioBooted       = false
	FactorioBootedAt     time.Time
	FactorioBootedLock   sync.RWMutex
	FactorioLaunchLock   sync.Mutex
	UpdateFactorioLock   sync.Mutex
	DoUpdateFactorio     = false
	DoUpdateFactorioLock sync.Mutex

	/* Locker detect */
	LockerDetectStart time.Time
	LockerStart       bool
	LockerLock        sync.Mutex
	LastLockerName    string

	/* Factorio autostart */
	FactAutoStart     = false
	FactAutoStartLock sync.RWMutex

	/* Reboot-when-empty */
	QueueReload     = false
	QueueReloadLock sync.RWMutex

	/*Factorio save game data */
	GameMapName = ""
	GameMapPath = ""
	GameMapLock sync.Mutex

	/* Players online */
	NumPlayers     = 0
	NumPlayersLock sync.RWMutex

	/* Slow-connect status */
	ConnectPauseLock  sync.Mutex
	ConnectPauseTimer int64 = 0
	ConnectPauseCount       = 0

	/* Number of man-minutes */
	ManMinutes     = 0
	ManMinutesLock sync.Mutex

	/*  Map gen data */
	LastMapSeed uint64 = 0
	LastMapCode        = ""
)

type TickInt struct {
	Day  int
	Hour int
	Min  int
	Sec  int
}
