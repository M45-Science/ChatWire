package fact

import (
	"ChatWire/cfg"
	"fmt"
	"time"

	"github.com/hako/durafmt"
)

const units = "year:years,week:weeks,day:days,hour:hours,minute:minutes,second:seconds,ms:ms,us:us"

func AdvanceReset() {
	if !HasResetInterval() {
		return
	}
	s := cfg.Local.Options.ResetInterval
	newResetTime := cfg.Local.Options.NextReset.AddDate(s.Years, s.Months, s.Days)
	newResetTime.Add(time.Duration(s.Hours) * time.Hour)
	cfg.Local.Options.NextReset = newResetTime
}

func HasResetInterval() bool {
	s := cfg.Local.Options.ResetInterval
	if s.Years == 0 && s.Months == 0 && s.Days == 0 && s.Hours == 0 {
		return false
	}
	return true
}

func HasResetTime() bool {
	return cfg.Local.Options.NextReset.IsZero()
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
	next := time.Until(cfg.Local.Options.NextReset)

	units, err := durafmt.DefaultUnitsCoder.Decode(units)
	if err != nil {
		panic(err)
	}

	dura := durafmt.Parse(next)
	return dura.LimitToUnit("minute").LimitFirstN(2).Format(units)
}

func FormatResetTime() string {
	if !HasResetTime() {
		return "No reset is scheduled"
	}
	return cfg.Local.Options.NextReset.Format("Jan 02, 03:04PM UTC")
}

func FormatResetInterval() string {
	if !HasResetInterval() {
		return "No reset interval is set"
	}
	buf := ""

	first := true

	i := cfg.Local.Options.ResetInterval
	if i.Years > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v year", i.Years, Plural(i.Years))
	}
	if i.Months > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v month", i.Months, Plural(i.Months))
	}
	if i.Weeks > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v week", i.Weeks, Plural(i.Weeks))
	}
	if i.Days > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v day", i.Days, Plural(i.Days))
	}
	if i.Hours > 0 {
		if !first {
			buf = buf + ", "
		}
		first = false
		buf = buf + fmt.Sprintf("%v hour", i.Hours, Plural(i.Hours))
	}

	return buf
}

func Plural(i int) string {
	if i > 1 {
		return "s"
	}

	return ""
}
