package cfg

import "ChatWire/constants"

// GetFactorioFolder returns the path to the Factorio installation for the current server.
func GetFactorioFolder() string {
	return Global.Paths.Folders.ServersRoot +
		Global.Paths.ChatWirePrefix +
		Local.Callsign + "/" +
		Global.Paths.Folders.FactorioDir + "/"
}

// GetModsFolder returns the path to the mod directory.
func GetModsFolder() string {
	return Global.Paths.Folders.ServersRoot +
		Global.Paths.ChatWirePrefix +
		Local.Callsign + "/" +
		Global.Paths.Folders.FactorioDir + "/" +
		constants.ModsFolder + "/"
}

// GetSavesFolder returns the path to the saves directory.
func GetSavesFolder() string {
	return Global.Paths.Folders.ServersRoot +
		Global.Paths.ChatWirePrefix +
		Local.Callsign + "/" +
		Global.Paths.Folders.FactorioDir + "/" +
		Global.Paths.Folders.Saves
}
