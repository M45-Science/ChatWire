package fact

import (
	"bytes"
	"io"
	"sync"
	"time"

	"ChatWire/constants"
)

var (
	/*Factorio stdio links */
	Pipe       io.WriteCloser
	PipeLock   sync.Mutex
	GameBuffer *bytes.Buffer

	/* Factorio patch info */
	FactorioVersion = constants.Unknown

	NewVersion   = constants.Unknown
	NewPatchName = constants.Unknown
	LastSaveName = constants.Unknown

	/* Factorio in-game time data */
	LastGametime   = ""
	PausedTicks    = 0
	Gametime       = constants.Unknown
	GametimeString = constants.Unknown

	MaxTickHistory = 4000

	TickHistory     []TickInt
	TickHistoryLock sync.Mutex

	/* Factorio status */
	FactIsRunning    = false
	FactorioBooted   = false
	FactorioBootedAt time.Time
	DoUpdateFactorio = false

	/* Factorio autostart */
	FactAutoStart = false

	/* Reboot-when-empty */
	QueueReboot     = false
	QueueFactReboot = false

	/*Factorio save game data */
	GameMapName = ""
	GameMapPath = ""

	/* Players online */
	NumPlayers        = 0
	OnlinePlayersLock sync.RWMutex

	/* Slow-connect status */
	SlowConnectLock   sync.Mutex
	SlowConnectTimer  int64 = 0
	SlowConnectEvents       = 0

	/*  Map gen data */
	LastMapSeed int = 0
	LastMapCode     = ""
)

type TickInt struct {
	Day  int
	Hour int
	Min  int
	Sec  int
}
