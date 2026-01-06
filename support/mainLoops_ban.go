package support

import "ChatWire/banlist"

func startBanWatcher() {
	/********************************
	 * Watch ban file for changes
	 ********************************/
	go banlist.WatchBanFile()
}
