package glob

import (
	"io"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Pipe is an WriteCloser interface
var Version = "0.0.3-2-24-2020-01-56-AM"
var Pipe io.WriteCloser
var OurLogname = ""
var Gametime = "gx-x-x-x"
var CharName = ""
var Sav_timer time.Time
var Running = true
var Shutdown = false
var Reboot = false
var QueueReload = false
var NumPlayers = 0
var RecordPlayers = 0
var RelaunchThrottle = 0

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

var MaxMapTypes = 0
var MapTypes = [...]string{"placeholder", "default", "rich-resources", "marathon", "death-world", "death-world-marathon", "rail-world", "ribbon-world", "island"}
var MapPrevLock sync.Mutex
