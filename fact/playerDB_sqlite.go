package fact

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"strconv"
	"strings"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
)

func openSQLite() (*sql.DB, error) {
	dbPath := cfg.Global.Paths.Folders.ServersRoot + cfg.Global.Paths.DataFiles.DBFile
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	stmt := `CREATE TABLE IF NOT EXISTS players(
        name TEXT PRIMARY KEY,
        level INTEGER,
        id TEXT,
        ban_reason TEXT,
        creation INTEGER,
        last_seen INTEGER,
        minutes INTEGER,
        sus_score INTEGER
    );`
	if _, err := db.Exec(stmt); err != nil {
		return nil, err
	}
	return db, nil
}

func loadPlayersSQLite(bootMode, minimize, clearBans bool) {
	db, err := openSQLite()
	if err != nil {
		cwlog.DoLogCW("SQLite open: %v", err)
		return
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, level, id, ban_reason, creation, last_seen, minutes, sus_score FROM players")
	if err != nil {
		cwlog.DoLogCW("SQLite query: %v", err)
		return
	}
	defer rows.Close()

	glob.PlayerListLock.Lock()
	defer glob.PlayerListLock.Unlock()
	var removed int
	banCount := 0
	doBan := true
	for rows.Next() {
		var name, id, reason string
		var level int
		var creation, seen, minutes, sus int64
		if err := rows.Scan(&name, &level, &id, &reason, &creation, &seen, &minutes, &sus); err != nil {
			continue
		}
		if clearBans && level < 0 {
			removed++
			continue
		}
		if minimize {
			if level == 0 || level == -255 {
				removed++
				continue
			}
			if level > 0 {
				sus = 0
				reason = ""
			}
			if id == "0" {
				id = ""
			}
			if _, err := strconv.ParseUint(id, 10, 64); err != nil {
				reason = id
				id = ""
			}
		}
		if banCount > 5 {
			doBan = false
		}
		didBan := false
		if level == 2 && minutes > constants.VeteranThresh {
			level = 3
		}
		if bootMode {
			didBan = addPlayer(strings.ToLower(name), level, id, creation, seen, reason, sus, minutes, false)
		} else {
			didBan = addPlayer(strings.ToLower(name), level, id, creation, seen, reason, sus, minutes, doBan)
		}
		if didBan {
			banCount++
		}
	}
	if removed > 0 {
		cwlog.DoLogCW("Removed: %v entries.\n", removed)
	}
}

func writePlayersSQLite() {
	db, err := openSQLite()
	if err != nil {
		cwlog.DoLogCW("SQLite open: %v", err)
		return
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		cwlog.DoLogCW("SQLite begin: %v", err)
		return
	}
	if _, err := tx.Exec("DELETE FROM players"); err != nil {
		cwlog.DoLogCW("SQLite clear: %v", err)
		tx.Rollback()
		return
	}
	glob.PlayerListLock.RLock()
	defer glob.PlayerListLock.RUnlock()
	stmt, err := tx.Prepare("INSERT INTO players(name, level, id, ban_reason, creation, last_seen, minutes, sus_score) VALUES (?,?,?,?,?,?,?,?)")
	if err != nil {
		cwlog.DoLogCW("SQLite prepare: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()
	for name, pd := range glob.PlayerList {
		_, err := stmt.Exec(name, pd.Level, pd.ID, pd.BanReason, pd.Creation, pd.LastSeen, pd.Minutes, pd.SusScore)
		if err != nil {
			cwlog.DoLogCW("SQLite insert: %v", err)
		}
	}
	if err := tx.Commit(); err != nil {
		cwlog.DoLogCW("SQLite commit: %v", err)
	}
}
