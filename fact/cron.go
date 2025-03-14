package fact

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/robfig/cron"

	"ChatWire/cfg"
	"ChatWire/cwlog"
)

var (
	CronVar       *cron.Cron
	NextResetTime time.Time
	NextReset     string
	TillReset     string
	NextResetUnix int64
)

func SetupSchedule() (err bool) {
	if cfg.Local.Options.Schedule != "" && cfg.Local.Options.Schedule != "no-reset" {
		if CronVar != nil {
			cwlog.DoLogCW("SetupSchedule: CronVar is not nil, removing old schedule")
			CronVar.Stop()
			CronVar = nil
		}
		CronVar = cron.NewWithLocation(time.UTC)
		err := InterpSchedule(cfg.Local.Options.Schedule, false)

		if err {
			cwlog.DoLogCW("SetupSchedule: Error setting up schedule.")
			return true
		} else {
			cwlog.DoLogCW("Schedule set up: " + cfg.Local.Options.Schedule)
			CronVar.Start()

			var maxTime time.Duration = 0
			for _, entry := range CronVar.Entries() {
				until := time.Until(entry.Next)
				if until > maxTime {
					maxTime = until
					NextResetTime = entry.Next
				}
			}

			UpdateScheduleDesc()
			return false
		}
	} else {
		cwlog.DoLogCW("SetupSchedule: No schedule set, skipping.")
		return true
	}
}

func doWarn(mins int) {
	if FactorioBooted && FactIsRunning && !cfg.Local.Options.SkipReset {
		buf := fmt.Sprintf("WARNING: MAP WILL RESET IN %v MINUTES!", mins)
		FactChat(AddFactColor("red", buf))
		FactChat(AddFactColor("cyan", buf))
		FactChat(AddFactColor("black", buf))

		CMS(cfg.Local.Channel.ChatChannel, buf)
	}
}

func InterpSchedule(desc string, test bool) (err bool) {
	var warn30, warn15, warn5, warn1, reset string

	date := 1
	dateb := 15
	day := "FRI"

	hour := 16
	minute := 0

	if cfg.Local.Options.ResetHour > 0 {
		hour = cfg.Local.Options.ResetHour
	}

	if cfg.Local.Options.ResetDate > 0 {
		date = cfg.Local.Options.ResetDate
		dateb = dateb + cfg.Local.Options.ResetDate

		//Wrap date around for two-week
		if dateb > 28 {
			dateb = (28 - dateb)
		}
		if dateb <= 0 {
			dateb = int(math.Abs(float64(dateb)))
		}
		if dateb > 28 {
			dateb = 28
		}

		if date <= 0 {
			date = 1
		}
		if date > 28 {
			date = 28
		}
	}
	if cfg.Local.Options.ResetDay != "" {
		day = cfg.Local.Options.ResetDay
	}

	if strings.EqualFold(desc, "three-months") {
		warn30 = fmt.Sprintf("0 %v %v %v */3 *", minute+30, hour-1, date)
		warn15 = fmt.Sprintf("0 %v %v %v */3 *", minute+45, hour-1, date)
		warn5 = fmt.Sprintf("0 %v %v %v */3 *", minute+55, hour-1, date)
		warn1 = fmt.Sprintf("0 %v %v %v */3 *", minute+59, hour-1, date)
		reset = fmt.Sprintf("0 %v %v %v */3 *", minute, hour, date)
	} else if strings.EqualFold(desc, "two-months") {
		warn30 = fmt.Sprintf("0 %v %v %v */2 *", minute+30, hour-1, date)
		warn15 = fmt.Sprintf("0 %v %v %v */2 *", minute+45, hour-1, date)
		warn5 = fmt.Sprintf("0 %v %v %v */2 *", minute+55, hour-1, date)
		warn1 = fmt.Sprintf("0 %v %v %v */2 *", minute+59, hour-1, date)
		reset = fmt.Sprintf("0 %v %v %v */2 *", minute, hour, date)
	} else if strings.EqualFold(desc, "monthly") {
		warn30 = fmt.Sprintf("0 %v %v %v * *", minute+30, hour-1, date)
		warn15 = fmt.Sprintf("0 %v %v %v * *", minute+45, hour-1, date)
		warn5 = fmt.Sprintf("0 %v %v %v * *", minute+55, hour-1, date)
		warn1 = fmt.Sprintf("0 %v %v %v * *", minute+59, hour-1, date)
		reset = fmt.Sprintf("0 %v %v %v * *", minute, hour, date)
	} else if strings.EqualFold(desc, "twice-monthly") {
		warn30 = fmt.Sprintf("0 %v %v %v,%v * *", minute+30, hour-1, date, dateb)
		warn15 = fmt.Sprintf("0 %v %v %v,%v * *", minute+45, hour-1, date, dateb)
		warn5 = fmt.Sprintf("0 %v %v %v,%v * *", minute+55, hour-1, date, dateb)
		warn1 = fmt.Sprintf("0 %v %v %v,%v * *", minute+59, hour-1, date, dateb)
		reset = fmt.Sprintf("0 %v %v %v,%v * *", minute, hour, date, dateb)
	} else if strings.EqualFold(desc, "fridays") || strings.EqualFold(desc, "day-of-week") {
		warn30 = fmt.Sprintf("0 %v %v * * %v", minute+30, hour-1, day)
		warn15 = fmt.Sprintf("0 %v %v * * %v", minute+45, hour-1, day)
		warn5 = fmt.Sprintf("0 %v %v * * %v", minute+55, hour-1, day)
		warn1 = fmt.Sprintf("0 %v %v * * %v", minute+59, hour-1, day)
		reset = fmt.Sprintf("0 %v %v * * %v", minute, hour, day)
	} else if strings.EqualFold(desc, "odd-dates") {
		warn30 = fmt.Sprintf("0 %v %v */2 * *", minute+30, hour-1)
		warn15 = fmt.Sprintf("0 %v %v */2 * *", minute+45, hour-1)
		warn5 = fmt.Sprintf("0 %v %v */2 * *", minute+55, hour-1)
		warn1 = fmt.Sprintf("0 %v %v */2 * *", minute+59, hour-1)
		reset = fmt.Sprintf("0 %v %v */2 * *", minute, hour)
	} else if strings.EqualFold(desc, "third-dates") {
		warn30 = fmt.Sprintf("0 %v %v */3 * *", minute+30, hour-1)
		warn15 = fmt.Sprintf("0 %v %v */3 * *", minute+45, hour-1)
		warn5 = fmt.Sprintf("0 %v %v */3 * *", minute+55, hour-1)
		warn1 = fmt.Sprintf("0 %v %v */3 * *", minute+59, hour-1)
		reset = fmt.Sprintf("0 %v %v */3 * *", minute, hour)
	} else if strings.EqualFold(desc, "daily") {
		warn30 = fmt.Sprintf("0 %v %v * * *", minute+30, hour-1)
		warn15 = fmt.Sprintf("0 %v %v * * *", minute+45, hour-1)
		warn5 = fmt.Sprintf("0 %v %v * * *", minute+55, hour-1)
		warn1 = fmt.Sprintf("0 %v %v * * *", minute+59, hour-1)
		reset = fmt.Sprintf("0 %v %v * * *", minute, hour)
	} else if strings.EqualFold(desc, "no-reset") {
		//
	} else {
		cwlog.DoLogCW("interpSchedule: Invalid schedule preset: " + desc)
		return true
	}

	if !test && reset != "" {
		err5 := CronVar.AddFunc(warn30, func() { doWarn(30) })
		err4 := CronVar.AddFunc(warn15, func() { doWarn(15) })
		err3 := CronVar.AddFunc(warn5, func() { doWarn(5) })
		err2 := CronVar.AddFunc(warn1, func() { doWarn(1) })
		err1 := CronVar.AddFunc(reset, func() { go Map_reset(false) })

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil {
			cwlog.DoLogCW("interpSchedule: Error adding function: " + err1.Error() + err2.Error() + err3.Error() + err4.Error() + err5.Error())
			return true
		} else {
			return false
		}
	} else if reset == "" {

		/* Disable cron if set to no-reset */
		if CronVar != nil {
			CronVar.Stop()
			CronVar = nil
			TillReset = ""
			NextReset = ""
			WriteFact("/resetint")
			WriteFact("/resetdur")
		} else {
			NextResetUnix = -1
			TillReset = ""
			NextReset = ""
			WriteFact("/resetint")
			WriteFact("/resetdur")
		}
	}

	return false
}

func UpdateScheduleDesc() (err bool) {

	if cfg.Local.Options.Schedule != "" && cfg.Local.Options.Schedule != "no-reset" && time.Until(NextResetTime) > 0 {

		units, err := durafmt.DefaultUnitsCoder.Decode("y:y,week:weeks,day:days,hour:hours,minute:minutes,sec:secs,ms:ms,us:us")
		if err != nil {
			panic(err)
		}

		n := NextResetTime
		NextReset = n.Format("Mon Jan 02 15:04 MST")
		NextResetUnix = n.Unix()
		TillReset = durafmt.Parse(time.Until(n).Round(time.Minute)).LimitFirstN(2).Format(units)

		if cfg.Local.Options.SkipReset {
			TillReset = "(SKIP) " + TillReset
			NextReset = "(SKIP)" + NextReset
		}

		return false
	}

	//Fallback
	TillReset = ""
	NextReset = ""
	NextResetUnix = -1

	cwlog.DoLogCW("UpdateScheduleDesc: No schedule set, skipping.")
	return true
}
