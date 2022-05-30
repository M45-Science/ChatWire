package fact

import (
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron"

	"ChatWire/cfg"
	"ChatWire/cwlog"
)

var CronVar *cron.Cron
var ScheduleDescription string

func SetupSchedule() {
	if cfg.Local.Options.Schedule != "" {
		CronVar = cron.NewWithLocation(time.UTC)
		err := interpSchedule(cfg.Local.Options.Schedule)

		if err {
			cwlog.DoLogCW("Error setting up schedule.")
		} else {
			cwlog.DoLogCW("Schedule set up: " + cfg.Local.Options.Schedule)
			CronVar.Start()
		}
	} else {
		cwlog.DoLogCW("No schedule set, skipping.")
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

func interpSchedule(desc string) bool {
	var warn15, warn5, warn1, reset string

	if desc == "" {
		return false
	} else if strings.EqualFold(desc, "three months") {
		warn15 = "0 45 15 */3 *"
		warn5 = "0 55 15 */3 *"
		warn1 = "0 59 15 */3 *"
		reset = "0 0 16 */3 * *"
	} else if strings.EqualFold(desc, "two months") {
		warn15 = "0 45 15 */2 *"
		warn5 = "0 55 15 */2 *"
		warn1 = "0 59 16 */2 *"
		reset = "0 0 16 1 */2 *"
	} else if strings.EqualFold(desc, "monthly") {
		warn15 = "0 45 15 1 * *"
		warn5 = "0 55 15 1 * *"
		warn1 = "0 59 15 1 * *"
		reset = "0 0 16 1 * *"
	} else if strings.EqualFold(desc, "1st and 15th") {
		warn15 = "0 45 15 1,15 * *"
		warn5 = "0 55 15 1,15 * *"
		warn1 = "0 59 15 1,15 * *"
		reset = "0 0 16 1,15 * *"
	} else if strings.EqualFold(desc, "fridays") {
		warn15 = "0 45 15 * * FRI"
		warn5 = "0 55 15 * * FRI"
		warn1 = "0 59 15 * * FRI"
		reset = "0 0 16 * * FRI"
	} else if strings.EqualFold(desc, "every other day") {
		warn15 = "0, 45, 15 */2 * *"
		warn5 = "0, 55, 15 */2 * *"
		warn1 = "0, 59, 15 */2 * *"
		reset = "0, 0, 16 */2 * *"
	} else if strings.EqualFold(desc, "every day") {
		warn15 = "0, 45, 15 * * *"
		warn5 = "0, 55, 15 * * *"
		warn1 = "0, 59, 15 * * *"
		reset = "0, 0, 16 * * *"
	}

	err1 := CronVar.AddFunc(warn15, func() { doWarn(15) })
	err2 := CronVar.AddFunc(warn5, func() { doWarn(5) })
	err3 := CronVar.AddFunc(warn1, func() { doWarn(1) })
	err4 := CronVar.AddFunc(reset, func() { Map_reset("", false) })

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		return false
	} else {
		UpdateScheduleDesc()
		return true
	}

}

func UpdateScheduleDesc() bool {
	if len(CronVar.Entries()) == 3 {
		nextReset := CronVar.Entries()[3].Next
		ScheduleDescription = nextReset.Round(time.Hour).String()
		GenerateFactorioConfig()
		//support.ConfigSoftMod()
		return true
	} else {
		return false
	}
}
