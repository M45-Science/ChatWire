package fact

import (
	"fmt"

	"ChatWire/constants"
)

func StatusChatWireOnline() string {
	return constants.ProgName + " " + constants.Version + " is now online."
}

func StatusChatWireShuttingDown() string {
	return constants.ProgName + " shutting down."
}

func StatusFactorioOnline(version string) string {
	if version == "" || version == constants.Unknown {
		return "🟢 Factorio is online."
	}
	return "🟢 Factorio " + version + " is online."
}

func StatusFactorioOffline() string {
	return "🔴 Factorio is now offline."
}

func StatusStartingFactorio() string {
	return "Starting Factorio."
}

func StatusStoppingFactorio() string {
	return "Stopping Factorio."
}

func StatusRestartingFactorio() string {
	return "Restarting Factorio."
}

func StatusRestartingChatWire() string {
	return "Restarting ChatWire."
}

func StatusChangingMap(saveName string) string {
	if saveName != "" {
		return fmt.Sprintf("Changing map to %s.", saveName)
	}
	return "Changing map."
}

func StatusResettingMap() string {
	return "Resetting map."
}

func StatusLoadingMods() string {
	return "Factorio is loading mods."
}

func StatusLoadingModsStill() string {
	return "Factorio is continuing to load mods."
}

func StatusLoadingMap(saveName string) string {
	if saveName != "" {
		return fmt.Sprintf("Factorio is loading map %s.", saveName)
	}
	return "Factorio is loading the map."
}

func StatusLoadingMapStill(saveName string) string {
	if saveName != "" {
		return fmt.Sprintf("Factorio is still loading map %s.", saveName)
	}
	return "Factorio is still loading the map."
}

func StatusSavingMap() string {
	return "Factorio is saving the map."
}

func StatusSavingMapStill() string {
	return "Factorio is still saving the map."
}

func StatusBringingServerOnline() string {
	return "Factorio is bringing the server online."
}

func StatusBringingServerOnlineStill() string {
	return "Factorio is still bringing the server online."
}
