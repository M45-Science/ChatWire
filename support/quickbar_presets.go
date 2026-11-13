package support

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/util"
)

const (
	regularQuickbarSlots = 3
	veteranQuickbarSlots = 5
	maxQuickbarSlots     = 10
)

type quickbarPreset struct {
	Label    string `json:"label,omitempty"`
	Exchange string `json:"exchange"`
}

type quickbarRequest struct {
	Name      string  `json:"name"`
	ID        int64   `json:"id"`
	Action    string  `json:"action"`
	Slot      int     `json:"slot"`
	Admin     bool    `json:"admin"` // Set by SoftMod from the player who triggered the game event.
	FirstJoin bool    `json:"first_join"`
	Label     *string `json:"label,omitempty"`
	Exchange  string  `json:"exchange,omitempty"`
}

type quickbarReply struct {
	Name     string   `json:"name"`
	ID       int64    `json:"id"`
	Allowed  bool     `json:"allowed"`
	Labels   []string `json:"labels"`
	Exchange string   `json:"exchange,omitempty"`
	Error    string   `json:"error,omitempty"`
}

func quickbarSlotLimit(req quickbarRequest) int {
	glob.PlayerListLock.RLock()
	player := glob.PlayerList[strings.ToLower(req.Name)]
	var level int
	var discordID string
	if player != nil {
		level, discordID = player.Level, player.ID
	}
	glob.PlayerListLock.RUnlock()
	if level < 0 {
		return 0
	}
	if req.Admin || disc.IsPatreon(discordID) {
		return maxQuickbarSlots
	}
	if level == 3 {
		return veteranQuickbarSlots
	}
	if level == 2 {
		return regularQuickbarSlots
	}
	return 0
}

func handleQuickbarPreset(data json.RawMessage) {
	var req quickbarRequest
	if json.Unmarshal(data, &req) != nil || req.ID < 1 || req.Name == "" || len(req.Name) > constants.MaxNameLength {
		return
	}
	limit := quickbarSlotLimit(req)
	reply := quickbarReply{Name: req.Name, ID: req.ID, Allowed: limit > 0, Labels: make([]string, limit)}
	if reply.Allowed {
		path := filepath.Join(cfg.Global.Paths.Folders.ServersRoot, "quickbar-presets.json")
		if err := readWriteQuickbar(path, req, &reply); err != nil {
			cwlog.DoLogCW("Quickbar preset %s failed for %s: %v", req.Action, req.Name, err)
			reply.Error = "ChatWire could not access this preset. Please try again."
		}
	}
	fact.WriteSoftModCommand("quickbar-result", reply)
}

func readWriteQuickbar(path string, req quickbarRequest, reply *quickbarReply) error {
	switch req.Action {
	case "access":
	case "load", "store", "rename":
		if req.Slot < 1 || req.Slot > len(reply.Labels) {
			return errors.New("invalid preset slot")
		}
	default:
		return errors.New("unknown preset action")
	}
	if req.Slot == 1 {
		if req.Action == "rename" {
			reply.Error = "Default has a fixed name."
			return nil
		}
		req.Label = nil
	}
	if req.Action == "store" {
		if len(req.Exchange) == 0 || len(req.Exchange) > 10240 {
			return errors.New("invalid preset size")
		}
		decoded, err := fact.DecodeSoftModPayload(req.Exchange)
		if err != nil || len(decoded) > 10240 || !strings.HasPrefix(string(decoded), "M45-QB2=") {
			return errors.New("invalid quickbar exchange string")
		}
	}
	if req.Action == "rename" || req.Action == "store" {
		if req.Action == "rename" && req.Label == nil {
			return errors.New("missing preset name")
		}
		if req.Label != nil && utf8.RuneCountInString(*req.Label) > 32 {
			reply.Error = "Preset names can have up to 32 characters."
			return nil
		}
		// Every server locks the same file before reading and replacing a slot.
		lock, err := util.AcquireFileLock(path+".lock", cfg.Local.Callsign, 2*time.Second)
		if err != nil {
			return err
		}
		defer lock.Release()
	}
	var db map[string][]quickbarPreset
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		db = make(map[string][]quickbarPreset)
	case err != nil:
		return err
	default:
		if err := json.Unmarshal(data, &db); err != nil {
			return err
		}
		if db == nil {
			return errors.New("invalid quickbar database")
		}
	}
	key := strings.ToLower(req.Name)
	slots := db[key]
	switch req.Action {
	case "store", "rename":
		for len(slots) < req.Slot {
			slots = append(slots, quickbarPreset{})
		}
		if req.Action == "store" {
			slots[req.Slot-1].Exchange = req.Exchange
		}
		if req.Label != nil {
			slots[req.Slot-1].Label = strings.TrimSpace(*req.Label)
		}
		db[key] = slots
		if err := util.WriteJSONAtomic(path, db, 0600); err != nil {
			return err
		}
	case "load":
		if req.Slot <= len(slots) {
			reply.Exchange = slots[req.Slot-1].Exchange
		}
	case "access":
		if req.FirstJoin && len(slots) > 0 {
			reply.Exchange = slots[0].Exchange
		}
	}
	for i, preset := range slots[:min(len(slots), len(reply.Labels))] {
		if i > 0 {
			reply.Labels[i] = preset.Label
		}
	}
	return nil
}
