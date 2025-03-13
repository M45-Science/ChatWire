package fact

import (
	"ChatWire/cfg"
	"ChatWire/cwlog"
	"fmt"
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
	until := cfg.Local.Options.NextReset.Sub(time.Now().UTC())

	for _, time := range warnTimes {
		if until == time {
			warnMapReset()
			return
		}
	}

	if until < 0 {

		if HasResetInterval() {
			AdvanceReset()
		} else {
			cfg.Local.Options.NextReset = time.Time{}
		}

		//Reset was some time ago, skip
		if until < maxResetWindow {
			units, err := durafmt.DefaultUnitsCoder.Decode(units)
			if err != nil {
				panic(err)
			}
			cfg.Local.Options.NextReset = time.Time{}
			cwlog.DoLogCW("Scheduled map reset was over %v ago. Skipping.", durafmt.Parse(maxResetWindow).Format(units))
		} else {

			Map_reset(false)
		}
	}
}

func warnMapReset() {
	buf := "WARNING: Map reset in " + TimeTillReset()
	LogCMS(cfg.Local.Channel.ChatChannel, buf)
	FactChat(AddFactColor("red", buf))
	FactChat(AddFactColor("cyan", buf))
	FactChat(AddFactColor("black", buf))
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
	cfg.Local.Options.NextReset = startDate
	cfg.WriteLCfg()
}

func AdvanceReset() {
	if !HasResetInterval() {
		return
	}
	s := cfg.Local.Options.ResetInterval
	newResetTime := cfg.Local.Options.NextReset.AddDate(0, s.Months, s.Days)
	newResetTime = newResetTime.Add(time.Duration(s.Hours) * time.Hour)
	cfg.Local.Options.NextReset = newResetTime
	SetResetDate()

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
	next := cfg.Local.Options.NextReset.UTC().Sub(time.Now().UTC()).Round(time.Minute)
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
