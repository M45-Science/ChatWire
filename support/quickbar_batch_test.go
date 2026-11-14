package support

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ChatWire/fact"
)

func quickbarBatchFixture(t *testing.T) (string, *testWriteCloser, string) {
	t.Helper()
	w := &testWriteCloser{}
	fact.SetFactorioPipe(w, 0)
	t.Cleanup(func() { fact.SetFactorioPipe(nil, 0) })
	exchange, err := fact.EncodeSoftModPayload([]byte("M45-QB2=rail,"))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(t.TempDir(), "quickbar-presets.json"), w, exchange
}

func queuePreset(b *quickbarSaveBatch, path, who, action string, slot int, exchange string, label *string, id int64) {
	b.enqueue(path, quickbarRequest{Name: who, ID: id, Action: action, Slot: slot, Exchange: exchange, Label: label}, quickbarReply{Name: who, ID: id, Allowed: true, Labels: make([]string, 16)})
}

func TestQuickbarBatchCombinesSlotsPlayersAndLatestChanges(t *testing.T) {
	path, w, exchange := quickbarBatchFixture(t)
	var batch quickbarSaveBatch
	latest, _ := fact.EncodeSoftModPayload([]byte("M45-QB2=copper-plate,"))
	name := "Rails"
	for id := int64(1); id <= 60; id++ {
		queuePreset(&batch, path, "Alice", "store", 2, exchange, nil, id)
	}
	queuePreset(&batch, path, "ALICE", "rename", 2, "", &name, 61)
	queuePreset(&batch, path, "Alice", "store", 2, latest, nil, 62)
	queuePreset(&batch, path, "Alice", "store", 16, exchange, nil, 63)
	queuePreset(&batch, path, "Bob", "store", 1, exchange, nil, 64)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("queued requests wrote the file before the batch flush")
	}
	if len(batch.pending[path]) != 3 {
		t.Fatal("repeat clicks grew the queue")
	}
	_, pending := capturedChatWireRequest(t, w.String())
	if pending["pending"] != true {
		t.Fatal("save was acknowledged before persistence", pending)
	}
	w.Reset()
	batch.flush()
	db, err := readQuickbarDatabase(path)
	if err != nil || db["alice"][1].Exchange != latest || db["alice"][1].Label != name || db["alice"][15].Exchange != exchange || db["bob"][0].Exchange != exchange {
		t.Fatalf("batch lost updates: %+v %v", db, err)
	}
	if len(batch.pending) != 0 || strings.Count(w.String(), "/chatwire ") != 3 {
		t.Fatal("flush did not complete one reply per merged slot")
	}
	_, saved := capturedChatWireRequest(t, w.String())
	if saved["pending"] != nil || saved["error"] != nil {
		t.Fatal("successful flush did not confirm saving", saved)
	}
	// An idle flush must not touch the file.
	old := time.Unix(1, 0)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	w.Reset()
	batch.flush()
	info, err := os.Stat(path)
	if err != nil || !info.ModTime().Equal(old) || w.Len() != 0 {
		t.Fatal("idle batch wrote the file or sent a reply")
	}
}

func TestQuickbarBatchesMergeOtherInstancesAndKeepLoadAvailable(t *testing.T) {
	path, _, exchange := quickbarBatchFixture(t)
	var first, second quickbarSaveBatch
	queuePreset(&first, path, "Alice", "store", 1, exchange, nil, 1)
	queuePreset(&second, path, "Bob", "store", 1, exchange, nil, 2)
	first.flush()
	// Loading stays immediate while another instance has pending changes.
	reply := quickbarReply{Labels: make([]string, 2)}
	if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "load", Slot: 1}, &reply); err != nil || reply.Exchange != exchange {
		t.Fatal("batching blocked loading", reply, err)
	}
	second.flush()
	db, err := readQuickbarDatabase(path)
	if err != nil || db["alice"][0].Exchange != exchange || db["bob"][0].Exchange != exchange {
		t.Fatal("another instance's update was lost", db, err)
	}
}

func TestQuickbarFailedBatchReportsFailureAndCanRetry(t *testing.T) {
	path, w, exchange := quickbarBatchFixture(t)
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	var batch quickbarSaveBatch
	queuePreset(&batch, path, "Alice", "store", 1, exchange, nil, 1)
	w.Reset()
	batch.flush()
	_, failed := capturedChatWireRequest(t, w.String())
	if failed["error"] == nil || failed["pending"] != nil {
		t.Fatal("failed batch claimed success", failed)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "corrupt" {
		t.Fatal("overwrote a corrupt database")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	queuePreset(&batch, path, "Alice", "store", 1, exchange, nil, 2)
	w.Reset()
	batch.flush()
	_, saved := capturedChatWireRequest(t, w.String())
	if saved["error"] != nil {
		t.Fatal("retry failed", saved)
	}
}

func TestQuickbarBatchTickAndShutdownFlush(t *testing.T) {
	path, _, exchange := quickbarBatchFixture(t)
	var batch quickbarSaveBatch
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ticks := make(chan time.Time)
	done := make(chan struct{})
	go func() { batch.run(ctx, ticks); close(done) }()
	queuePreset(&batch, path, "Alice", "store", 1, exchange, nil, 1)
	ticks <- time.Now()
	// Taking mu waits for any current flush to finish. If the tick has only
	// been received, poll until its batch has been drained.
	deadline := time.Now().Add(2 * time.Second)
	for {
		batch.mu.Lock()
		pending := len(batch.pending)
		batch.mu.Unlock()
		if pending == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("tick did not flush")
		}
		time.Sleep(time.Millisecond)
	}
	queuePreset(&batch, path, "Bob", "store", 1, exchange, nil, 2)
	cancel()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown did not flush")
	}
	db, err := readQuickbarDatabase(path)
	if err != nil || db["alice"][0].Exchange != exchange || db["bob"][0].Exchange != exchange {
		t.Fatal("tick or shutdown lost a save", db, err)
	}
	if quickbarSaveInterval != 5*time.Second {
		t.Fatal("unexpected batch interval")
	}
}
