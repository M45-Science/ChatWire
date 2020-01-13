package glob

import (
	"io"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Pipe is an WriteCloser interface
var Version = "0.0.2"
var Pipe io.WriteCloser
var Gametime = "gx-x-x-x"
var CharName = ""
var Sav_timer time.Time
var Running = true
var Shutdown = false
var Reboot = false
var QueueReload = false
var NumPlayers = 0
var RecordPlayers = 0

var DS *discordgo.Session

const MaxPlayers = 16777215

var PlayerListMax = 0
var PlayerList [MaxPlayers + 1]string
var NumLogins = 0

var PlayerListWriteLock sync.Mutex
var PlayerListLock sync.Mutex

var RecordPlayersWriteLock sync.Mutex
var RecordPlayersLock sync.Mutex
var Refresh = true

var NoResponseCount = 0
