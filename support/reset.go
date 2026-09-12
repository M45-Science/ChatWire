package support

import "ChatWire/fact"

var lastDur string

func UpdateDuration() {
	buf := resetDurationText()
	if buf != lastDur {
		fact.WriteSoftModCommand("config", map[string]any{"reset_duration": buf})
		lastDur = buf
	}
}

func UpdateInterval() {
	fact.WriteSoftModCommand("config", map[string]any{"reset_date": resetDateText()})
}

func resetDurationText() string {
	if !fact.HasResetTime() {
		return ""
	}
	buf := fact.TimeTillReset()
	if fact.HasResetInterval() {
		buf += " (" + fact.FormatResetInterval() + ")"
	}
	return buf
}

func resetDateText() string {
	if !fact.HasResetInterval() {
		return ""
	}
	return fact.FormatResetTime()
}
