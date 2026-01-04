package support

/* Main threads/loops */
func MainLoops() {
	startGameWatchdog()
	startBanWatcher()
	startCMSBuffer()
	startPasscodeCleanup()
	startPanelTokenCleanup()
	startPlayerListSaveLoop()
	startPlayerSeenSaveLoop()
	startDBFileWatcher()
	startGlobalConfigWatcher()
	startLocalConfigWatcher()
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
