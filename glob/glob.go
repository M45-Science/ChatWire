package glob

import (
	"io"
	"time"
)

// Pipe is an WriteCloser interface
var Pipe io.WriteCloser
var Gametime = "gx-x-x-x"
var Sav_timer time.Time
var CharName = ""
var Running = true
var Shutdown = false
