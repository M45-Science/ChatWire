package support

import (
	"strings"

	"ChatWire/cwlog"
)

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
