package support

import (
	"strings"

	"ChatWire/cwlog"
)

func handleCmdMsg(input *handleData) bool {
	/******************
	 * COMMAND REPORTING
	 ******************/
	if strings.HasPrefix(input.line, "[CMD]") {
		cwlog.DoLogAudit(input.line)
		return true
	}
	return false
}

func handleAuditMsg(input *handleData) bool {
	/******************
	 * AUDIT LOGGING
	 ******************/
	if strings.HasPrefix(input.line, "[AUDIT]") {
		cwlog.DoLogAudit(input.line)
		return true
	}
	return false
}
