package fact

import (
	"fmt"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/robfig/cron"

	"ChatWire/cfg"
	"ChatWire/cwlog"
)

var CronVar *cron.Cron
var NextReset string
var TillReset string
var NextResetUnix int64

func SetupSchedule() (err bool) {
	if cfg.Local.Options.Schedule != "" {
		if CronVar != nil {
			cwlog.DoLogCW("SetupSchedule: CronVar is not nil, removing old schedule")
			CronVar.Stop()
		}
		CronVar = cron.NewWithLocation(time.UTC)
		err := InterpSchedule(cfg.Local.Options.Schedule, false)

		if err {
			cwlog.DoLogCW("SetupSchedule: Error setting up schedule.")
			return true
		} else {
			cwlog.DoLogCW("Schedule set up: " + cfg.Local.Options.Schedule)
			CronVar.Start()
			UpdateScheduleDesc()
			return false
		}
	} else {
		cwlog.DoLogCW("SetupSchedule: No schedule set, skipping.")
		return true
	}
}

func doWarn(mins int) {
	if FactorioBooted && FactIsRunning {
		buf := fmt.Sprintf("WARNING: MAP WILL RESET IN %v MINUTES!", mins)
		FactChat(AddFactColor("red", buf))
		FactChat(AddFactColor("green", buf))
		FactChat(AddFactColor("blue", buf))
		FactChat(AddFactColor("white", buf))
		FactChat(AddFactColor("black", buf))

		CMS(cfg.Local.Channel.ChatChannel, buf)
	}
}

func InterpSchedule(desc string, test bool) (err bool) {
	var warn15, warn5, warn1, reset string

	if strings.EqualFold(desc, "three-months") {
		warn15 = "0 45 15 1 */3 *"
		warn5 = "0 55 15 1 */3 *"
		warn1 = "0 59 15 1 */3 *"
		reset = "0 0 16 1 */3 * *"
	} else if strings.EqualFold(desc, "two-months") {
		warn15 = "0 45 15 1 */2 *"
		warn5 = "0 55 15 1 */2 *"
		warn1 = "0 59 16 1 */2 *"
		reset = "0 0 16 1 */2 *"
	} else if strings.EqualFold(desc, "monthly") {
		warn15 = "0 45 15 1 * *"
		warn5 = "0 55 15 1 * *"
		warn1 = "0 59 15 1 * *"
		reset = "0 0 16 1 * *"
	} else if strings.EqualFold(desc, "twice-monthly") {
		warn15 = "0 45 15 1,15 * *"
		warn5 = "0 55 15 1,15 * *"
		warn1 = "0 59 15 1,15 * *"
		reset = "0 0 16 1,15 * *"
	} else if strings.EqualFold(desc, "fridays") {
		warn15 = "0 45 15 * * FRI"
		warn5 = "0 55 15 * * FRI"
		warn1 = "0 59 15 * * FRI"
		reset = "0 0 16 * * FRI"
	} else if strings.EqualFold(desc, "odd-dates") {
		warn15 = "0, 45, 15 */2 * *"
		warn5 = "0, 55, 15 */2 * *"
		warn1 = "0, 59, 15 */2 * *"
		reset = "0, 0, 16 */2 * *"
	} else if strings.EqualFold(desc, "daily") {
		warn15 = "0, 45, 15 * * *"
		warn5 = "0, 55, 15 * * *"
		warn1 = "0, 59, 15 * * *"
		reset = "0, 0, 16 * * *"
	} else if strings.EqualFold(desc, "no-reset") {
		//
	} else {
		cwlog.DoLogCW("interpSchedule: Invalid schedule preset: " + desc)
		return true
	}

	if !test && reset != "" {
		err1 := CronVar.AddFunc(warn15, func() { doWarn(15) })
		err2 := CronVar.AddFunc(warn5, func() { doWarn(5) })
		err3 := CronVar.AddFunc(warn1, func() { doWarn(1) })
		err4 := CronVar.AddFunc(reset, func() { Map_reset("", false) })

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			cwlog.DoLogCW("interpSchedule: Error adding function: " + err1.Error() + err2.Error() + err3.Error() + err4.Error())
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

		}
		return true
	}

	return false
}

func UpdateScheduleDesc() (err bool) {

	if cfg.Local.Options.Schedule != "" && CronVar != nil {
		e := CronVar.Entries()
		a := len(e)
		if a > 3 {

			units, err := durafmt.DefaultUnitsCoder.Decode("year:years,week:weeks,day:days,hour:hours,minute:minutes,second:seconds,millisecond:milliseconds,microsecond:microseconds")
			if err != nil {
				panic(err)
			}

			n := e[a-1].Next
			NextReset = n.Format("Monday, January 2 15:04 MST")
			NextResetUnix = n.Unix()
			TillReset = durafmt.Parse(time.Until(n).Round(time.Minute)).LimitFirstN(2).Format(units) + " from now"

			return false
		} else {
			cwlog.DoLogCW("UpdateScheduleDesc: No schedule set, skipping.")
			return true
		}
	}

	return false
}
