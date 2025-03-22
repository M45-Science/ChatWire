package fact

import (
	"ChatWire/cfg"
	"fmt"
	"strings"
	"time"

	"github.com/hako/durafmt"
)

const units = "year:years,week:weeks,day:days,hour:hours,minute:minutes,second:seconds,ms:ms,us:us"
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
			units, err := durafmt.DefaultUnitsCoder.Decode(units)
			if err != nil {
				panic(err)
			}
			LogCMS(cfg.Local.Channel.ChatChannel, "❇️ Scheduled map reset was over "+durafmt.Parse(maxResetWindow).Format(units)+" ago. Skipping.")
		} else {
			Map_reset(false)
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

func DisableNextReset() {
	cfg.Local.Options.NextReset = time.Time{}
	cfg.WriteLCfg()
}

func DisableResetSchedule() {
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
	units, err := durafmt.DefaultUnitsCoder.Decode(units)
	if err != nil {
		panic(err)
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
		buf = buf + fmt.Sprintf("%v month%v", i.Months, Plural(i.Months))
	}
	if i.Weeks > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v week%v", i.Weeks, Plural(i.Weeks))
	}
	if i.Days > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v day%v", i.Days, Plural(i.Days))
	}
	if i.Hours > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v hour%v", i.Hours, Plural(i.Hours))
	}

	return "Every " + buf
}

func Plural(i int) string {
	if i > 1 {
		return "s"
	}

	return ""
}
