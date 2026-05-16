package cfg

import (
	"os"
	"path/filepath"
	"strings"

	"ChatWire/constants"
)

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

// GetSharedMapGeneratorFolder returns the parent-level folder for reusable named generators.
func GetSharedMapGeneratorFolder() string {
	return filepath.Join(Global.Paths.Folders.ServersRoot, Global.Paths.Folders.MapGenerators)
}

// GetLocalMapGeneratorFolder returns this ChatWire instance's local generator folder.
func GetLocalMapGeneratorFolder() string {
	folder := Global.Paths.Folders.MapGenerators
	if folder == "" {
		folder = constants.DefaultMapGeneratorsDir
	}

	folder = filepath.Base(filepath.Clean(folder))
	if folder == "." || folder == string(os.PathSeparator) {
		folder = constants.DefaultMapGeneratorsDir
	}

	localPath := filepath.Join(".", folder)
	if absPath, err := filepath.Abs(localPath); err == nil {
		return absPath
	}
	return localPath
}

// GetMapGeneratorFolder returns the folder where a generator's JSON files live.
func GetMapGeneratorFolder(name string) string {
	if strings.EqualFold(name, constants.CustomMapGeneratorName) {
		return GetLocalMapGeneratorFolder()
	}
	return GetSharedMapGeneratorFolder()
}

// GetMapGeneratorFiles returns Factorio's map-gen and map-settings JSON paths for a generator.
func GetMapGeneratorFiles(name string) (string, string) {
	dir := GetMapGeneratorFolder(name)
	return filepath.Join(dir, name+"-gen.json"), filepath.Join(dir, name+"-set.json")
}
