package support

// StartFactorioLoops starts the loops required to manage and observe Factorio.
// They run even when Discord integration is disabled.
func StartFactorioLoops() {
	startGameWatchdog()
	go HandleChat()
}

/* Main threads/loops */
func MainLoops() {
	startBanWatcher()
	startCMSBuffer()
	startPasscodeCleanup()
	startPlayerListSaveLoop()
	startPlayerSeenSaveLoop()
	startDBFileWatcher()
	startGuildSyncLoop()
	startRoleRefreshLoop()
	startQueuedRebootLoop()
	startUpdateNudgeLoop()
	startLogFileWatchLoop()
	startFactorioUpdateLoop()
	startChannelNameLoop()
	startPauseExpiryLoop()
	startOnlinePollLoop()
	startModUpdateLoop()
	startPlayerTimeLoop()
	startResetDurationLoop()
	startMapResetLoop()
	startModPackCleanupLoop()

	go checkHours()
}
