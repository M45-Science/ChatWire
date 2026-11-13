package support

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"ChatWire/constants"
)

const (
	burstQuiet         = 3 * time.Second
	burstLimit         = 15 * time.Second
	siteRadius         = 128 // tiles from the first reported position, on the same surface
	firstRepeat        = constants.SusWarningInterval * time.Minute
	maxRepeat          = 16 * time.Minute
	reviewMarkMinimum  = 120
	reviewMineMinimum  = 48
	witnessMarkMinimum = 40
	witnessMineMinimum = 16
	siteExpiry         = 30 * time.Minute
)

type activityLocation struct {
	surface string
	x, y    float64
	gps     string
}

func locationFromActivity(line string) activityLocation {
	i := strings.LastIndex(line, "[gps=")
	if i < 0 {
		return activityLocation{}
	}
	raw, _, ok := strings.Cut(line[i+5:], "]")
	if !ok {
		return activityLocation{}
	}
	parts := strings.SplitN(raw, ",", 3)
	if len(parts) < 2 {
		return activityLocation{}
	}
	x, errX := strconv.ParseFloat(parts[0], 64)
	y, errY := strconv.ParseFloat(parts[1], 64)
	if errX != nil || errY != nil || math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
		return activityLocation{}
	}
	surface := "nauvis"
	if len(parts) == 3 && parts[2] != "" {
		surface = parts[2]
	}
	return activityLocation{surface, x, y, "[gps=" + raw + "]"}
}

// A site keeps its cooldown between bursts. The counts describe the open
// burst (or the accumulated follow-up waiting for its cooldown).
type activitySite struct {
	anchor            activityLocation
	last, first, next time.Time
	delay             time.Duration
	armed             bool
	marks, cancels    int
	mined, placed     int
	types             map[string]int
	player            string
	firstGPS          string
	miningContext     string
	normalAlert       bool
	gameNotified      bool
	roleNotified      bool
	early             bool
}

const earlySessionWindow = 15 * time.Minute
const earlyMiningBonus = 36

type activitySession struct {
	bonusUsed     bool
	joined        time.Time
	mined, placed int
}

// Keep the bonus local to this session and qualifying mining events. Repeated
// events do not accumulate the bonus in the persistent player database.
func (s *activitySession) observe(now time.Time, action string) int64 {
	if s == nil || now.Before(s.joined) || now.Sub(s.joined) > earlySessionWindow {
		return 0
	}
	switch action {
	case "placed":
		s.placed++
	case "mined":
		s.mined++
		if !s.bonusUsed && s.mined >= 8 && s.mined >= 3*s.placed {
			return earlyMiningBonus
		}
	}
	return 0
}

func (a *activityAlerts) join(player string, now time.Time) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.sessions == nil {
		a.sessions = make(map[string]*activitySession)
	}
	a.sessions[strings.ToLower(player)] = &activitySession{joined: now}
}

func (a *activityAlerts) leave(player string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, strings.ToLower(player))
}

type activityAlerts struct {
	mu       sync.Mutex
	sessions map[string]*activitySession
	sites    map[string][]*activitySite
}

var mapActivityAlerts = activityAlerts{sites: make(map[string][]*activitySite)}

func (a *activityAlerts) record(now time.Time, player, action, detail string, qualified bool, context string) {
	cancel := action == "undecon"
	mark := action == "decon"
	if !mark && !cancel && action != "mined" && action != "placed" && !qualified {
		return
	}
	entity, _, _ := strings.Cut(detail, " ")
	// A planner rectangle is an additional record for the same selection.
	// Individual marks carry the prototype names and actual positions.
	if (mark || cancel) && (entity == "" || strings.HasPrefix(entity, "[")) {
		return
	}
	loc := locationFromActivity(detail)
	key := strings.ToLower(player)
	a.mu.Lock()
	defer a.mu.Unlock()
	var site *activitySite
	best := math.Inf(1)
	for _, s := range a.sites[key] {
		if loc.surface != s.anchor.surface || now.Sub(s.last) >= siteExpiry {
			continue
		}
		dx, dy := loc.x-s.anchor.x, loc.y-s.anchor.y
		dist := dx*dx + dy*dy
		if dist <= siteRadius*siteRadius && dist < best {
			site = s
			best = dist
		}
	}
	if site == nil {
		if cancel {
			return
		}
		site = &activitySite{anchor: loc, player: player}
		a.sites[key] = append(a.sites[key], site)
	}
	if cancel && site.first.IsZero() {
		return
	}
	if site.first.IsZero() {
		site.first = now
		site.firstGPS = loc.gps
	}
	site.last = now
	if mark {
		if site.types == nil {
			site.types = make(map[string]int)
		}
		site.types[entity]++
		site.marks++
	} else if cancel {
		site.cancels++
	} else if action == "mined" {
		site.mined++
	} else if action == "placed" {
		site.placed++
	}
	if context != "" {
		site.miningContext = context
	}
	site.normalAlert = site.normalAlert || (qualified && context == "")
	if qualified {
		session := a.sessions[key]
		if session != nil && !now.Before(session.joined) && now.Sub(session.joined) <= earlySessionWindow {
			site.early = true
		}
	}
	site.armed = site.armed || qualified
}

func (s *activitySite) summary() string {
	where := s.firstGPS
	if where == "" {
		where = "location unavailable"
	}
	if s.marks == 0 {
		if !s.normalAlert {
			return fmt.Sprintf("Early-session mining review: %s near %s.", s.player, where)
		}
		return fmt.Sprintf("Possible suspicious activity: %s near %s (mining activity).", s.player, where)
	}
	names := make([]string, 0, len(s.types))
	for name := range s.types {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if s.types[names[i]] == s.types[names[j]] {
			return names[i] < names[j]
		}
		return s.types[names[i]] > s.types[names[j]]
	})
	var items []string
	for _, name := range names[:min(5, len(names))] {
		items = append(items, fmt.Sprintf("%d × %s", s.types[name], name))
	}
	if len(names) > 5 {
		items = append(items, fmt.Sprintf("%d other types", len(names)-5))
	}
	msg := fmt.Sprintf("Deconstruction activity: %s near %s. %d marks: %s.", s.player, where, s.marks, strings.Join(items, ", "))
	if s.cancels > 0 {
		msg += fmt.Sprintf(" %d cancellation events.", s.cancels)
	}
	return msg
}

type activityAlert struct {
	player, message string
	reportOnly      bool
	pingRole        bool
	gameMessage     string
}

func (a *activityAlerts) due(now time.Time) []activityAlert {
	a.mu.Lock()
	defer a.mu.Unlock()
	var alerts []activityAlert
	for key, s := range a.sessions {
		if now.Sub(s.joined) > earlySessionWindow {
			delete(a.sessions, key)
		}
	}
	for player, sites := range a.sites {
		keep := sites[:0]
		for _, s := range sites {
			if !s.first.IsZero() && (now.Sub(s.last) >= burstQuiet || now.Sub(s.first) >= burstLimit) && (!s.armed || !now.Before(s.next)) {
				if s.armed {
					msg := s.summary()
					if s.miningContext != "" {
						msg += " " + s.miningContext
					}
					netMarks := s.marks - s.cancels
					miningImbalance := s.mined >= 3*s.placed
					pingEvidence := s.miningContext != "" || netMarks >= reviewMarkMinimum || (s.mined >= reviewMineMinimum && miningImbalance)
					alert := activityAlert{player: s.player, message: msg, reportOnly: !s.normalAlert}
					if s.normalAlert && !s.gameNotified && s.firstGPS != "" {
						markMinimum, mineMinimum := 4*witnessMarkMinimum, 4*witnessMineMinimum
						if s.early {
							markMinimum, mineMinimum = witnessMarkMinimum, witnessMineMinimum
						}
						var what string
						switch {
						case netMarks >= markMinimum:
							what = fmt.Sprintf("%d deconstruction marks (%d cancellations)", s.marks, s.cancels)
						case s.mined >= mineMinimum && miningImbalance:
							what = fmt.Sprintf("%d structures mined, %d placed", s.mined, s.placed)
						}
						if what != "" {
							alert.gameMessage = fmt.Sprintf("Activity check near %s: %s by %s. If you see damage, use /report with details. Eligible players can vote with /banish.", s.firstGPS, what, s.player)
							s.gameNotified = true
						}
					}
					// A witness warning also merits a moderator ping if this incident has not paged yet.
					if !s.roleNotified && (pingEvidence || alert.gameMessage != "") {
						alert.pingRole = true
						s.roleNotified = true
					}
					alerts = append(alerts, alert)
					if s.delay == 0 {
						s.delay = firstRepeat
					} else {
						s.delay = min(2*s.delay, maxRepeat)
					}
					s.next = now.Add(s.delay)
				}
				s.mined, s.placed = 0, 0
				s.miningContext = ""
				s.early = false
				s.normalAlert = false
				s.first = time.Time{}
				s.marks, s.cancels, s.types, s.armed = 0, 0, nil, false
			}
			if !s.first.IsZero() || s.delay > 0 && now.Sub(s.last) < siteExpiry {
				keep = append(keep, s)
			}
		}
		if len(keep) == 0 {
			delete(a.sites, player)
		} else {
			a.sites[player] = keep
		}
	}
	return alerts
}
