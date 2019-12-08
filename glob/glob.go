package glob

import (
	"io"
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

var DS *discordgo.Session
