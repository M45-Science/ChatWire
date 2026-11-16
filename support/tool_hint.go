package support

import (
	"encoding/json"
	"time"

	"ChatWire/cwlog"
	"ChatWire/fact"
)

func handleToolHint(data json.RawMessage) {
	var request struct {
		Name string `json:"name"`
		ID   int64  `json:"id"`
	}
	if json.Unmarshal(data, &request) != nil || request.ID < 1 {
		return
	}
	allowed, err := fact.ClaimToolHint(request.Name, time.Now())
	if err != nil {
		cwlog.DoLogCW("Tool hint cooldown failed for %s: %v", request.Name, err)
	}
	fact.WriteSoftModCommand("tool-hint-result", map[string]any{
		"name": request.Name, "id": request.ID, "allowed": allowed && err == nil,
	})
}
