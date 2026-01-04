package fact

import (
	"io"
	"sync"
	"time"

	"ChatWire/constants"
)

var (
	/*Factorio stdio links */
	Pipe       io.WriteCloser
	PipeLock   sync.Mutex
	GameLineCh chan string

	/* Factorio patch info */
	FactorioVersion = constants.Unknown

	NewVersion   = constants.Unknown
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
	SlowConnectLock  sync.Mutex
	SlowConnectTimer int64
)

type TickInt struct {
	Day  int
	Hour int
	Min  int
	Sec  int
}
