package cfg

import (
	"os"
	"testing"
)

func TestSetGlobalDefaultsUsesBinaryPlayerDatabase(t *testing.T) {
	oldGlobal := Global
	t.Cleanup(func() { Global = oldGlobal })

	Global = global{}
	Global.Paths.Folders.ServersRoot = t.TempDir() + string(os.PathSeparator)
	setGlobalDefaults()

	if Global.Paths.DataFiles.DBFile != "playerdb.cwdb" {
		t.Fatalf("DBFile = %q, want playerdb.cwdb", Global.Paths.DataFiles.DBFile)
	}
	if Global.Paths.DataFiles.DBFormat != "binary" {
		t.Fatalf("DBFormat = %q, want binary", Global.Paths.DataFiles.DBFormat)
	}
}

func TestSetGlobalDefaultsPreservesJSONPlayerDatabase(t *testing.T) {
	oldGlobal := Global
	t.Cleanup(func() { Global = oldGlobal })

	Global = global{}
	Global.Paths.Folders.ServersRoot = t.TempDir() + string(os.PathSeparator)
	Global.Paths.DataFiles.DBFile = "players.json"
	Global.Paths.DataFiles.DBFormat = " JSON "
	setGlobalDefaults()

	if Global.Paths.DataFiles.DBFile != "players.json" {
		t.Fatalf("DBFile = %q, want players.json", Global.Paths.DataFiles.DBFile)
	}
	if Global.Paths.DataFiles.DBFormat != "json" {
		t.Fatalf("DBFormat = %q, want json", Global.Paths.DataFiles.DBFormat)
	}
}
