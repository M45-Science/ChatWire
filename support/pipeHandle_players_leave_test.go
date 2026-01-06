package support

import (
	"strings"
	"testing"

	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/glob"
)

type recordWriteCloser struct {
	writes []string
}

func (r *recordWriteCloser) Write(p []byte) (int, error) {
	r.writes = append(r.writes, string(p))
	return len(p), nil
}

func (r *recordWriteCloser) Close() error { return nil }

func TestHandlePlayerLeaveRefreshesOnlinePlayersWithSoftMod(t *testing.T) {
	origPipe := fact.Pipe
	origBooted := fact.FactorioBooted
	origRunning := fact.FactIsRunning
	origSoftMod := glob.SoftModVersion
	origOnlineCmd := glob.OnlineCommand

	t.Cleanup(func() {
		fact.Pipe = origPipe
		fact.FactorioBooted = origBooted
		fact.FactIsRunning = origRunning
		glob.SoftModVersion = origSoftMod
		glob.OnlineCommand = origOnlineCmd
	})

	rec := &recordWriteCloser{}
	fact.Pipe = rec
	fact.FactorioBooted = true
	fact.FactIsRunning = true
	glob.SoftModVersion = "test"
	glob.OnlineCommand = constants.SoftModOnlineCMD

	input := &handleData{
		noDatestamp:        "[LEAVE] alice",
		noDatestampList:    []string{"[LEAVE]", "alice"},
		noDatestampListLen: 2,
	}

	if !handlePlayerLeave(input) {
		t.Fatalf("expected leave to be handled")
	}

	for _, write := range rec.writes {
		if strings.Contains(write, constants.SoftModOnlineCMD) {
			return
		}
	}
	t.Fatalf("expected %q to be written to the Factorio pipe, got %#v", constants.SoftModOnlineCMD, rec.writes)
}

func TestHandleDisconnectRefreshesOnlinePlayersWithSoftMod(t *testing.T) {
	origPipe := fact.Pipe
	origSoftMod := glob.SoftModVersion
	origOnlineCmd := glob.OnlineCommand

	t.Cleanup(func() {
		fact.Pipe = origPipe
		glob.SoftModVersion = origSoftMod
		glob.OnlineCommand = origOnlineCmd
	})

	rec := &recordWriteCloser{}
	fact.Pipe = rec
	glob.SoftModVersion = "test"
	glob.OnlineCommand = constants.SoftModOnlineCMD

	input := &handleData{
		noTimecode: "Info ServerMultiplayerManager abc",
		line:       "Info ServerMultiplayerManager abc removing peer 123",
	}
	handleDisconnect(input)

	for _, write := range rec.writes {
		if strings.Contains(write, constants.SoftModOnlineCMD) {
			return
		}
	}
	t.Fatalf("expected %q to be written to the Factorio pipe, got %#v", constants.SoftModOnlineCMD, rec.writes)
}
