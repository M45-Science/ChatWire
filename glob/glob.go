package glob

import (
	"io"
	"os/exec"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Pipe is an WriteCloser interface
var Pipe io.WriteCloser
var Gametime = "gx-x-x-x"
var Sav_timer time.Time
var CharName = ""
var Running = true
var Shutdown = false
var Reboot = false

var DS *discordgo.Session
var GCMD *exec.Cmd

const MaxPlayers = 65535

var PlayerListMax = 0
var PlayerList [MaxPlayers + 1]string
var NumLogins = 0
