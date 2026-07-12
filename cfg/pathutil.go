package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ChatWire/constants"
	"ChatWire/util"
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

// GetMapGeneratorCacheFolder returns the local per-server cache folder for a shared generator.
func GetMapGeneratorCacheFolder(name string) string {
	safeName := cacheableMapGeneratorName(name)
	if safeName == "" {
		return ""
	}
	return filepath.Join(GetLocalMapGeneratorFolder(), constants.MapGeneratorCacheDir, safeName)
}

// GetCachedMapGeneratorFiles returns local fallback JSON paths for a shared generator.
func GetCachedMapGeneratorFiles(name string) (string, string) {
	dir := GetMapGeneratorCacheFolder(name)
	if dir == "" {
		return "", ""
	}
	return filepath.Join(dir, constants.MapGenSettingsName), filepath.Join(dir, constants.MapSettingsName)
}

// CacheMapGenerator stores a local fallback copy of a complete shared generator.
func CacheMapGenerator(name string) (string, string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, "none") || strings.EqualFold(name, constants.CustomMapGeneratorName) {
		return "", "", nil
	}

	if cacheableMapGeneratorName(name) == "" {
		return "", "", fmt.Errorf("map generator %q cannot be cached", name)
	}

	genSrc, setSrc := GetMapGeneratorFiles(name)
	if !fileExists(genSrc) || !fileExists(setSrc) {
		return "", "", fmt.Errorf("map generator %q is unavailable", name)
	}

	genDst, setDst := GetCachedMapGeneratorFiles(name)
	if genDst == "" || setDst == "" {
		return "", "", fmt.Errorf("map generator %q cannot be cached", name)
	}

	if err := os.MkdirAll(filepath.Dir(genDst), os.ModePerm); err != nil {
		return "", "", err
	}
	if err := copyMapGeneratorFile(genSrc, genDst); err != nil {
		return "", "", err
	}
	if err := copyMapGeneratorFile(setSrc, setDst); err != nil {
		return "", "", err
	}
	return genDst, setDst, nil
}

func cacheableMapGeneratorName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `\\/`) {
		return ""
	}

	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || clean != filepath.Base(clean) {
		return ""
	}
	return clean
}

func copyMapGeneratorFile(src string, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return util.WriteBytesAtomic(dst, data, 0644)
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

	folderGenPath := filepath.Join(dir, name, constants.MapGenSettingsName)
	folderSetPath := filepath.Join(dir, name, constants.MapSettingsName)
	if fileExists(folderGenPath) && fileExists(folderSetPath) {
		return folderGenPath, folderSetPath
	}

	return filepath.Join(dir, name+"-gen.json"), filepath.Join(dir, name+"-set.json")
}

func fileExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}
