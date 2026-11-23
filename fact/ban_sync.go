package fact

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"ChatWire/glob"
)

type desiredBan struct {
	Name     string `json:"name"`
	Banned   bool   `json:"banned"`
	Reason   string `json:"reason,omitempty"`
	Revision uint64 `json:"revision,string"`
}

type desiredRank struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Revision uint64 `json:"revision,string"`
}

func acknowledgePlayerRank(rank desiredRank) {
	banSync.Lock()
	defer banSync.Unlock()
	if taggedBanSyncActiveLocked() {
		writeSoftModCommandForGeneration("rank-revision", rank, &banSync.generation)
	}
}

type banSyncState struct {
	desired desiredBan
	request string
	sent    time.Time
	applied bool
}

var banSync = struct {
	sync.Mutex
	enabled    bool
	generation uint64
	players    map[string]banSyncState
}{players: make(map[string]banSyncState)}

func rememberBanState(name string, player *glob.PlayerData, force bool) {
	if !player.Banned() && player.BanRevision == 0 && !force {
		return
	}
	desired := desiredBan{Name: name, Banned: player.Banned(), Revision: player.BanRevision}
	if desired.Banned {
		desired.Reason = player.BanReason
	}
	banSync.Lock()
	defer banSync.Unlock()
	if old, exists := banSync.players[name]; exists {
		if old.desired.Revision > desired.Revision || old.desired == desired && !force {
			return
		}
	}
	banSync.players[name] = banSyncState{desired: desired}
}

func taggedBanSyncActiveLocked() bool {
	PipeLock.Lock()
	defer PipeLock.Unlock()
	return banSync.enabled && Pipe != nil && pipeGeneration == banSync.generation
}

func TaggedBanSyncEnabled() bool {
	_, enabled := taggedBanSyncGeneration()
	return enabled
}

func taggedBanSyncGeneration() (uint64, bool) {
	banSync.Lock()
	defer banSync.Unlock()
	return banSync.generation, taggedBanSyncActiveLocked()
}

func TaggedBanSyncGeneration(generation uint64) bool {
	banSync.Lock()
	defer banSync.Unlock()
	return generation == banSync.generation && taggedBanSyncActiveLocked()
}

func EnableTaggedBanSync(generation uint64) {
	banSync.Lock()
	defer banSync.Unlock()
	PipeLock.Lock()
	current := Pipe != nil && generation == pipeGeneration
	PipeLock.Unlock()
	if !current || banSync.enabled && banSync.generation == generation {
		return
	}
	banSync.enabled, banSync.generation = true, generation
	for name, state := range banSync.players {
		banSync.players[name] = banSyncState{desired: state.desired}
	}
}

// ReconcilePlayerBans retries the latest committed state, including unbans.
// A new Factorio generation starts with no acknowledgements.
func ReconcilePlayerBans() {
	banSync.Lock()
	defer banSync.Unlock()
	if !taggedBanSyncActiveLocked() {
		return
	}
	now := time.Now()
	sent := 0
	for name, state := range banSync.players {
		if state.applied || now.Sub(state.sent) < 10*time.Second {
			continue
		}
		state.request = writeSoftModCommandForGeneration("ban-state", state.desired, &banSync.generation)
		if state.request == "" {
			return
		}
		state.sent = now
		banSync.players[name] = state
		sent++
		if sent == 32 {
			return
		}
	}
}

func ConfirmTaggedBan(generation uint64, request, name, revision string, changed bool) {
	name = strings.ToLower(name)
	banSync.Lock()
	defer banSync.Unlock()
	state, exists := banSync.players[name]
	if !exists || generation != banSync.generation || !taggedBanSyncActiveLocked() ||
		request == "" || request != state.request || revision != strconv.FormatUint(state.desired.Revision, 10) {
		return
	}
	state.applied = true
	state.request = ""
	banSync.players[name] = state
	if state.desired.Banned && changed {
		writeFactorioLineForGeneration("/purge "+name, 1024, &generation)
	}
}

// EnforceStoredBan is used on joins as well as during startup reconciliation.
func EnforceStoredBan(name string) bool {
	name = strings.ToLower(name)
	glob.PlayerListLock.RLock()
	player := glob.PlayerList[name]
	if player == nil || !player.Banned() {
		glob.PlayerListLock.RUnlock()
		return false
	}
	saved := *player
	glob.PlayerListLock.RUnlock()
	if TaggedBanSyncEnabled() {
		rememberBanState(name, &saved, true)
		ReconcilePlayerBans()
	} else {
		WriteBan(name, saved.BanReason)
	}
	return true
}
