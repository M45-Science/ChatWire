package controlruntime

import (
	"ChatWire/fact"
	"ChatWire/firewall"
	"ChatWire/glob"
	"ChatWire/webcontrol"
	"net/http"
	"sort"
	"strings"
)

func (rt *Runtime) playerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /internal/v1/players", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("scope") != "all" {
			fact.OnlinePlayersLock.RLock()
			players := append([]glob.OnlinePlayerData{}, glob.OnlinePlayers...)
			fact.OnlinePlayersLock.RUnlock()
			webcontrol.JSON(w, 200, map[string]any{"items": players})
			return
		}
		q := strings.ToLower(r.URL.Query().Get("q"))
		cursor := r.URL.Query().Get("cursor")
		glob.PlayerListLock.RLock()
		names := []string{}
		for name := range glob.PlayerList {
			if name > cursor && strings.Contains(name, q) {
				names = append(names, name)
			}
		}
		sort.Strings(names)
		more := len(names) > 100
		if more {
			names = names[:100]
		}
		out := []map[string]any{}
		for _, name := range names {
			p := glob.PlayerList[name]
			if p != nil {
				out = append(out, map[string]any{"id": name, "name": p.Name, "level": p.Level, "minutes": p.Minutes, "last_seen": p.LastSeen, "ban_reason": p.BanReason})
			}
		}
		glob.PlayerListLock.RUnlock()
		next := ""
		if more {
			next = names[len(names)-1]
		}
		webcontrol.JSON(w, 200, map[string]any{"items": out, "next_cursor": next})
	})
	mux.HandleFunc("GET /internal/v1/players/{player}", func(w http.ResponseWriter, r *http.Request) {
		glob.PlayerListLock.RLock()
		p := glob.PlayerList[strings.ToLower(r.PathValue("player"))]
		var out map[string]any
		if p != nil {
			out = map[string]any{"name": p.Name, "level": p.Level, "minutes": p.Minutes, "last_seen": p.LastSeen, "ban_reason": p.BanReason}
		}
		glob.PlayerListLock.RUnlock()
		if out == nil {
			webcontrol.Error(w, 404, "not_found", "Player not found.")
			return
		}
		webcontrol.JSON(w, 200, out)
	})
	mux.HandleFunc("GET /internal/v1/host/ip-bans", func(w http.ResponseWriter, r *http.Request) {
		if !rt.primary {
			webcontrol.Error(w, 403, "forbidden", "Primary instance required.")
			return
		}
		v, e := firewall.List()
		if e != nil {
			webcontrol.Error(w, 503, "unavailable", "Unable to read firewall rules.")
			return
		}
		webcontrol.JSON(w, 200, v)
	})
	mux.HandleFunc("POST /internal/v1/host/actions/{action}/preview", func(w http.ResponseWriter, r *http.Request) { rt.previewAction(w, r) })
	mux.HandleFunc("POST /internal/v1/host/actions/{action}", func(w http.ResponseWriter, r *http.Request) { rt.action(w, r) })
	mux.HandleFunc("POST /internal/v1/players/{player}/actions/set-level/preview", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("action", "player-level")
		rt.previewAction(w, r)
	})
	mux.HandleFunc("POST /internal/v1/players/{player}/actions/set-level", func(w http.ResponseWriter, r *http.Request) {
		r.SetPathValue("action", "player-level")
		rt.action(w, r)
	})
}
func setPlayerLevel(a webcontrol.Actor, p Params) (any, error) {
	if e := fact.SetPlayerLevelByModerator(p.Player, p.Level, p.Reason, a.Name); e != nil {
		return nil, e
	}
	return "Player level updated.", nil
}
func firewallAction(action, ip string) (any, error) {
	return firewall.Execute(strings.TrimPrefix(action, "ip-"), ip, "web")
}
