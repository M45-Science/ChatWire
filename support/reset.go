package support

import (
	"strings"

	"ChatWire/fact"
)

var lastDur string

func UpdateDuration() {
	if fact.HasResetTime() {
		buf := "/resetdur " + fact.TimeTillReset()
		if fact.HasResetInterval() {
			buf = buf + " (" + fact.FormatResetInterval() + ")"
		}
		/* Don't write it, if nothing has changed */
		if !strings.EqualFold(buf, lastDur) {
			fact.WriteFact(buf)
		}

		lastDur = buf
	} else {
		buf := "/resetdur"

		/* Don't write it, if nothing has changed */
		if !strings.EqualFold(buf, lastDur) {
			fact.WriteFact(buf)
		}

		lastDur = buf
	}
}

func UpdateInterval() {
	/* Config reset-interval */
	if fact.HasResetInterval() {
		fact.WriteFact("/resetint " + fact.FormatResetTime())
	} else {
		fact.WriteFact("/resetint")
	}
}
