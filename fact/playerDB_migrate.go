package fact

import (
	"os"
	"path/filepath"
	"strings"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

func migrateDB(sourceExt, destExt string) {
	base := strings.TrimSuffix(cfg.Global.Paths.DataFiles.DBFile, filepath.Ext(cfg.Global.Paths.DataFiles.DBFile))
	source := base + sourceExt
	dest := base + destExt
	srcPath := cfg.Global.Paths.Folders.ServersRoot + source
	if _, err := os.Stat(srcPath); err != nil {
		cwlog.DoLogCW("Source DB missing: %v", srcPath)
		return
	}
	orig := cfg.Global.Paths.DataFiles.DBFile
	cfg.Global.Paths.DataFiles.DBFile = source
	glob.PlayerList = make(map[string]*glob.PlayerData)
	LoadPlayers(false, false, false)
	cfg.Global.Paths.DataFiles.DBFile = dest
	WritePlayers()
	cfg.Global.Paths.DataFiles.DBFile = orig
	cwlog.DoLogCW("Migrated %v -> %v", source, dest)
}

func MigrateJSONToSQLite() {
	migrateDB(".json", ".sqlite")
}

func MigrateSQLiteToJSON() {
	migrateDB(".sqlite", ".json")
}
