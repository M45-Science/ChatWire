package glob

import (
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Pipe is an WriteCloser interface
var Pipe io.WriteCloser
var Gametime = "gx-x-x-x"
var CharName = ""
var Sav_timer time.Time
var Running = true
var Shutdown = false
var Reboot = false
var QueueReload = false

var DS *discordgo.Session
var GCMD *exec.Cmd

const MaxPlayers = 16777215

var PlayerListMax = 0
var PlayerList [MaxPlayers + 1]string
var NumLogins = 0

var PlayerListWriteLock sync.Mutex
var PlayerListLock sync.Mutex
