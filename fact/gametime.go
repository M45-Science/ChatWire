package fact

import (
	"fmt"
	"math"
	"sync"
	"time"

	"ChatWire/constants"
)

const (
	defaultGameUPS   = 60.0
	maximumGameUPS   = 600.0
	upsMeasureWindow = time.Minute
)

type gameTimeSample struct {
	seconds float64
	at      time.Time
}

var gameTimeClock = struct {
	sync.RWMutex
	initialized    bool
	sampleSeconds  float64
	sampledAt      time.Time
	ups            float64
	lastRunningUPS float64
	paused         bool
	samples        []gameTimeSample
}{
	ups:            defaultGameUPS,
	lastRunningUPS: defaultGameUPS,
}

// RecordGameTime records an authoritative /time response and updates the
// observed UPS used to interpolate timestamps between responses. It reports
// whether the authoritative game time was unchanged from the prior response.
func RecordGameTime(day, hour, minute, second int, raw string, initialUPS float64, sampledAt time.Time) bool {
	if sampledAt.IsZero() {
		sampledAt = time.Now()
	}
	initialUPS = validGameUPS(initialUPS)
	totalSeconds := float64(day*86400 + hour*3600 + minute*60 + second)
	return recordGameSeconds(totalSeconds, raw, initialUPS, sampledAt)
}

func recordGameSeconds(totalSeconds float64, raw string, initialUPS float64, sampledAt time.Time) bool {

	gameTimeClock.Lock()
	defer gameTimeClock.Unlock()

	unchanged := gameTimeClock.initialized && totalSeconds == gameTimeClock.sampleSeconds
	previousSeconds := gameTimeClock.sampleSeconds
	wasPaused := gameTimeClock.paused

	LastGametime = Gametime
	Gametime = formatGameTime(int64(math.Floor(totalSeconds)))
	GametimeString = raw

	if !gameTimeClock.initialized || totalSeconds < previousSeconds {
		gameTimeClock.ups = initialUPS
		gameTimeClock.lastRunningUPS = initialUPS
		gameTimeClock.paused = false
		gameTimeClock.samples = []gameTimeSample{{seconds: totalSeconds, at: sampledAt}}
	} else if wasPaused && totalSeconds > previousSeconds {
		// Do not include time spent paused in the measured UPS. Resume with the
		// most recent running rate until another advancing sample refines it.
		gameTimeClock.paused = false
		gameTimeClock.ups = validGameUPS(gameTimeClock.lastRunningUPS)
		gameTimeClock.samples = []gameTimeSample{{seconds: totalSeconds, at: sampledAt}}
	} else {
		gameTimeClock.samples = append(gameTimeClock.samples, gameTimeSample{seconds: totalSeconds, at: sampledAt})
		trimGameTimeSamplesLocked(sampledAt)
		if totalSeconds > previousSeconds {
			updateObservedUPSLocked()
		}
	}

	gameTimeClock.initialized = true
	gameTimeClock.sampleSeconds = totalSeconds
	gameTimeClock.sampledAt = sampledAt
	return unchanged
}

// RecordGameTick records the exact simulation tick returned by the SoftMod.
// game.speed is a multiplier, so its nominal UPS is speed * 60.
func RecordGameTick(tick uint64, speed float64, paused bool, sampledAt time.Time) {
	recordGameSeconds(float64(tick)/60.0, fmt.Sprintf("tick %d", tick), speed*60, sampledAt)
	SetGameTimePaused(paused)
	if paused {
		PausedTicks = constants.PauseThresh + 1
	} else {
		PausedTicks = 0
	}
}

// SetGameTimePaused stops or resumes interpolation. Authoritative samples still
// replace the displayed time while interpolation is stopped.
func SetGameTimePaused(paused bool) {
	gameTimeClock.Lock()
	defer gameTimeClock.Unlock()
	if gameTimeClock.paused == paused {
		return
	}
	gameTimeClock.paused = paused
	if gameTimeClock.initialized {
		gameTimeClock.samples = []gameTimeSample{{seconds: gameTimeClock.sampleSeconds, at: gameTimeClock.sampledAt}}
	}
}

// CurrentGametime returns the authoritative game time advanced at the measured
// UPS rate. It performs no background work and is corrected by every /time
// response.
func CurrentGametime() string {
	return currentGametimeAt(time.Now())
}

func currentGametimeAt(now time.Time) string {
	gameTimeClock.RLock()
	defer gameTimeClock.RUnlock()
	if !gameTimeClock.initialized {
		return Gametime
	}

	seconds := float64(gameTimeClock.sampleSeconds)
	if !gameTimeClock.paused && now.After(gameTimeClock.sampledAt) {
		seconds += now.Sub(gameTimeClock.sampledAt).Seconds() * gameTimeClock.ups / 60.0
	}
	return formatGameTime(int64(math.Floor(seconds)))
}

func CurrentGametimeString() string {
	gameTimeClock.RLock()
	defer gameTimeClock.RUnlock()
	return GametimeString
}

func ResetGametime(initialUPS float64) {
	initialUPS = validGameUPS(initialUPS)
	gameTimeClock.Lock()
	defer gameTimeClock.Unlock()
	LastGametime = ""
	Gametime = constants.Unknown
	GametimeString = constants.Unknown
	gameTimeClock.initialized = false
	gameTimeClock.sampleSeconds = 0
	gameTimeClock.sampledAt = time.Time{}
	gameTimeClock.ups = initialUPS
	gameTimeClock.lastRunningUPS = initialUPS
	gameTimeClock.paused = false
	gameTimeClock.samples = nil
}

func trimGameTimeSamplesLocked(now time.Time) {
	for len(gameTimeClock.samples) > 2 && now.Sub(gameTimeClock.samples[1].at) >= upsMeasureWindow {
		gameTimeClock.samples = gameTimeClock.samples[1:]
	}
}

func updateObservedUPSLocked() {
	if len(gameTimeClock.samples) < 2 {
		return
	}
	first := gameTimeClock.samples[0]
	last := gameTimeClock.samples[len(gameTimeClock.samples)-1]
	elapsed := last.at.Sub(first.at).Seconds()
	advanced := last.seconds - first.seconds
	if elapsed <= 0 || advanced <= 0 {
		return
	}
	ups := validGameUPS(advanced / elapsed * 60.0)
	gameTimeClock.ups = ups
	gameTimeClock.lastRunningUPS = ups
}

func validGameUPS(ups float64) float64 {
	if math.IsNaN(ups) || math.IsInf(ups, 0) || ups <= 0 {
		return defaultGameUPS
	}
	return min(ups, maximumGameUPS)
}

func formatGameTime(totalSeconds int64) string {
	if totalSeconds < 0 {
		totalSeconds = 0
	}
	day := totalSeconds / 86400
	hour := totalSeconds % 86400 / 3600
	minute := totalSeconds % 3600 / 60
	second := totalSeconds % 60
	switch {
	case day > 0:
		return fmt.Sprintf("%.2d-%.2d-%.2d-%.2d", day, hour, minute, second)
	case hour > 0:
		return fmt.Sprintf("%.2d-%.2d-%.2d", hour, minute, second)
	case minute > 0:
		return fmt.Sprintf("%.2d-%.2d", minute, second)
	default:
		return fmt.Sprintf("%.2d", second)
	}
}
