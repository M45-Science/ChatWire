package support

import (
	"os"
	"time"

	"ChatWire/cwlog"
	"ChatWire/glob"
)

func startLogFileWatchLoop() {
	/***********************************
	 * Fix lost connection to log files
	 ***********************************/
	go func() {
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()

		for glob.ServerRunning {
			<-ticker.C

			var err error
			if _, err = os.Stat(glob.CWLogName); err != nil {

				glob.CWLogDesc.Close()
				glob.CWLogDesc = nil
				cwlog.StartCWLog()
				cwlog.DoLogCW("CWLog file was deleted, recreated.")
			}

			if _, err = os.Stat(glob.AuditLogName); err != nil {

				glob.AuditLogDesc.Close()
				glob.AuditLogDesc = nil
				cwlog.StartAuditLog()
				cwlog.DoLogAudit("Audit log file was deleted, recreated.")
			}

			if _, err = os.Stat(glob.GameLogName); err != nil {
				glob.GameLogDesc.Close()
				glob.GameLogDesc = nil
				cwlog.StartGameLog()
				cwlog.DoLogGame("GameLog file was deleted, recreated.")
			}
		}
	}()
}
