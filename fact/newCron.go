package fact

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hako/durafmt"
)

const unitsSpec = "year:years,week:weeks,day:days,hour:hours,minute:minutes,second:seconds,ms:ms,us:us"

var (
	unitsOnce      sync.Once
	cachedUnits    durafmt.Units
	cachedUnitsErr error
)

func loadResetUnits() (durafmt.Units, error) {
	unitsOnce.Do(func() {
		cachedUnits, cachedUnitsErr = durafmt.DefaultUnitsCoder.Decode(unitsSpec)
	})

	return cachedUnits, cachedUnitsErr
}

const maxResetWindow = time.Hour * 3

func CheckMapReset() {
	if !HasResetTime() {
		return
	}

	var warnTimes = []time.Duration{
		time.Minute * 30, time.Minute * 10, time.Minute * 5, time.Minute, time.Second * 30, time.Second * 10}
	until := cfg.Local.Options.NextReset.Sub(time.Now().UTC().Round(time.Second)).Round(time.Second)

	for _, t := range warnTimes {
		if until == t {
			if NumPlayers == 0 {
				if until < (time.Minute * 5) {
					continue
				}
			}
			warnMapReset()
			break
		}
	}

	if until < 0 {

		if HasResetInterval() {
			AdvanceReset()
		} else {
			cfg.Local.Options.NextReset = time.Time{}
		}

		//Reset was some time ago, skip
		if until < -maxResetWindow {
			units, err := loadResetUnits()
			if err != nil {
				cwlog.DoLogCW(fmt.Sprintf("failed to load reset duration units: %v", err))
				LogCMS(cfg.Local.Channel.ChatChannel, "❇️ Scheduled map reset was over "+maxResetWindow.String()+" ago. Skipping.")
			} else {
				LogCMS(cfg.Local.Channel.ChatChannel, "❇️ Scheduled map reset was over "+durafmt.Parse(maxResetWindow).Format(units)+" ago. Skipping.")
			}
		} else {
			if err := Map_reset(false); err != nil {
				cwlog.DoLogCW(fmt.Sprintf("Scheduled map reset failed: %v", err))
			}
		}
	}
}

func warnMapReset() {
	buf := "Map reset in " + TimeTillReset()
	buf = strings.ToUpper(buf)
	LogCMS(cfg.Local.Channel.ChatChannel, "⚠️ **"+buf+"**")

	if NumPlayers > 0 {
		warn := "*** NOTICE: "
		FactChat(warn + AddFactColor("red", buf))
		FactChat(warn + AddFactColor("cyan", buf))
		FactChat(warn + AddFactColor("black", buf))
	}
}

func SetResetDate() {
	if !HasResetInterval() {
		return
	}
	n := cfg.Local.Options.ResetInterval

	var offset = time.Now().UTC()
	if cfg.Local.Options.ResetHour > 0 {
		base := time.Now().UTC()
		offset = time.Date(base.Year(), base.Month(), base.Day(), cfg.Local.Options.ResetHour, 0, 0, 0, time.UTC)
	}
	startDate := offset.AddDate(0, n.Months, n.Days)
	startDate = startDate.Add(time.Duration(n.Weeks) * time.Hour * 24 * 7)
	startDate = startDate.Add(time.Duration(n.Hours) * time.Hour)
	cfg.Local.Options.NextReset = startDate.Round(time.Second)
	cfg.WriteLCfg()
}

func AdvanceReset() {
	if !HasResetInterval() {
		return
	}
	s := cfg.Local.Options.ResetInterval
	partA := cfg.Local.Options.NextReset.AddDate(0, s.Months, s.Days)
	partB := partA.Add(time.Duration(s.Weeks) * time.Hour * 24 * 7)
	partC := partB.Add(time.Duration(s.Hours) * time.Hour)
	cfg.Local.Options.NextReset = partC.Round(time.Second)
	// Persist the updated NextReset so a process restart doesn’t lose the skip/advance
	cfg.WriteLCfg()
}

func HasResetInterval() bool {
	s := cfg.Local.Options.ResetInterval
	if s.Months == 0 && s.Weeks == 0 && s.Days == 0 && s.Hours == 0 {
		return false
	}
	return true
}

func HasResetTime() bool {
	return cfg.Local.Options.NextReset.Unix() > 0
}

func disableNextReset() {
	cfg.Local.Options.NextReset = time.Time{}
	cfg.WriteLCfg()
}

func disableResetSchedule() {
	cfg.Local.Options.ResetInterval = cfg.ResetInterval{}
	cfg.WriteLCfg()
}

func TimeTillReset() string {
	if !HasResetTime() {
		return "No reset is scheduled"
	}
	next := cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC())

	if next > time.Minute {
		next = next.Round(time.Minute)
	} else {
		next = next.Round(time.Second)
	}
	units, err := loadResetUnits()
	if err != nil {
		cwlog.DoLogCW(fmt.Sprintf("failed to load reset duration units: %v", err))
		return next.String()
	}

	dura := durafmt.Parse(next)
	return dura.LimitFirstN(2).Format(units)
}

func FormatResetTime() string {
	if !HasResetTime() {
		return "No reset is scheduled"
	}
	return cfg.Local.Options.NextReset.UTC().Format("Jan 02, 03:04PM UTC")
}

func FormatResetInterval() string {
	if !HasResetInterval() {
		return "No reset interval is set"
	}
	buf := ""

	first := true

	i := cfg.Local.Options.ResetInterval
	if i.Months > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v month%v", i.Months, plural(i.Months))
	}
	if i.Weeks > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v week%v", i.Weeks, plural(i.Weeks))
	}
	if i.Days > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v day%v", i.Days, plural(i.Days))
	}
	if i.Hours > 0 {
		if !first {
			buf = buf + ", "
		}
		buf = buf + fmt.Sprintf("%v hour%v", i.Hours, plural(i.Hours))
	}

	return "Every " + buf
}

func plural(i int) string {
	if i > 1 {
		return "s"
	}

	return ""
}
