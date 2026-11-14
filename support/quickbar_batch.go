package support

import (
	"context"
	"strings"
	"sync"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/util"
)

type quickbarSaveKey struct {
	player string
	slot   int
}

type queuedQuickbarSave struct {
	request    quickbarRequest
	reply      quickbarReply
	generation uint64
}

// Keep one merged update per player/slot rather than an unbounded click queue.
// Holding mu through a flush serializes disk writes with new queued updates.
type quickbarSaveBatch struct {
	mu      sync.Mutex
	pending map[string]map[quickbarSaveKey]queuedQuickbarSave
}

var quickbarSaves quickbarSaveBatch

func (b *quickbarSaveBatch) enqueue(path string, req quickbarRequest, reply quickbarReply) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pending == nil {
		b.pending = make(map[string]map[quickbarSaveKey]queuedQuickbarSave)
	}
	if b.pending[path] == nil {
		b.pending[path] = make(map[quickbarSaveKey]queuedQuickbarSave)
	}
	key := quickbarSaveKey{strings.ToLower(req.Name), req.Slot}
	if previous, ok := b.pending[path][key]; ok {
		// A rename must retain a queued toolbar; a store without a label must
		// retain a queued rename. The latest change to each field wins.
		if req.Action == "rename" && previous.request.Action == "store" {
			req.Action, req.Exchange = "store", previous.request.Exchange
		}
		if req.Label == nil {
			req.Label = previous.request.Label
		}
	}
	_, generation := fact.GameOutputCurrent()
	b.pending[path][key] = queuedQuickbarSave{request: req, reply: reply, generation: generation}
	// Send this before releasing mu so a flush can never overtake "Saving".
	reply.Pending = true
	fact.WriteSoftModCommand("quickbar-result", reply)
}

func (b *quickbarSaveBatch) flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for path, updates := range b.pending {
		db, err := writeQuickbarBatch(path, updates)
		if err != nil {
			cwlog.DoLogCW("Quickbar preset batch failed: %v", err)
		}
		for _, update := range updates {
			// Persist the request even if its Factorio process has stopped, but
			// never acknowledge an old request to a replacement map/process.
			if update.generation != 0 && !fact.IsCurrentFactorioGeneration(update.generation) {
				continue
			}
			reply := update.reply
			if err != nil {
				reply.Error = "ChatWire could not save this preset. Please try again."
			} else {
				fillQuickbarReply(db, update.request, &reply)
			}
			fact.WriteSoftModCommand("quickbar-result", reply)
		}
		delete(b.pending, path)
	}
}

func writeQuickbarBatch(path string, updates map[quickbarSaveKey]queuedQuickbarSave) (map[string][]quickbarPreset, error) {
	// Read the latest disk data under the lock, merging only this instance's
	// dirty slots so another server's updates are preserved.
	lock, err := util.AcquireFileLock(path+".lock", cfg.Local.Callsign, 2*time.Second)
	if err != nil {
		return nil, err
	}
	defer lock.Release()
	db, err := readQuickbarDatabase(path)
	if err != nil {
		return nil, err
	}
	for _, update := range updates {
		applyQuickbarPreset(db, update.request)
	}
	if err := util.WriteJSONAtomic(path, db, 0600); err != nil {
		return nil, err
	}
	return db, nil
}

// FlushQuickbarPresets persists queued requests before an orderly exit.
func FlushQuickbarPresets() { quickbarSaves.flush() }

func (b *quickbarSaveBatch) run(ctx context.Context, ticks <-chan time.Time) {
	for {
		select {
		case <-ctx.Done():
			b.flush()
			return
		case <-ticks:
			b.flush()
		}
	}
}

func startQuickbarSaveLoop() {
	ctx := glob.RuntimeContext()
	go func() {
		ticker := time.NewTicker(quickbarSaveInterval)
		defer ticker.Stop()
		quickbarSaves.run(ctx, ticker.C)
	}()
}
