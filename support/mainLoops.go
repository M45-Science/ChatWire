package support

/* Main threads/loops */
func MainLoops() {
	startGameWatchdog()
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
