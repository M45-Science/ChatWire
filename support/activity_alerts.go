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
	burstQuiet  = 3 * time.Second
	burstLimit  = 15 * time.Second
	siteRadius  = 128 // tiles from the first reported position, on the same surface
	firstRepeat = constants.SusWarningInterval * time.Minute
	maxRepeat   = 16 * time.Minute
	siteExpiry  = 30 * time.Minute
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
	types             map[string]int
	player            string
	firstGPS          string
}

type activityAlerts struct {
	mu    sync.Mutex
	sites map[string][]*activitySite
}

var mapActivityAlerts = activityAlerts{sites: make(map[string][]*activitySite)}

func (a *activityAlerts) record(now time.Time, player, action, detail string, qualified bool) {
	cancel := action == "undecon"
	mark := action == "decon"
	if !mark && !cancel && !qualified {
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
	}
	site.armed = site.armed || qualified
}

func (s *activitySite) summary() string {
	where := s.firstGPS
	if where == "" {
		where = "location unavailable"
	}
	if s.marks == 0 {
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

type activityAlert struct{ player, message string }

func (a *activityAlerts) due(now time.Time) []activityAlert {
	a.mu.Lock()
	defer a.mu.Unlock()
	var alerts []activityAlert
	for player, sites := range a.sites {
		keep := sites[:0]
		for _, s := range sites {
			if !s.first.IsZero() && (now.Sub(s.last) >= burstQuiet || now.Sub(s.first) >= burstLimit) && (!s.armed || !now.Before(s.next)) {
				if s.armed {
					alerts = append(alerts, activityAlert{s.player, s.summary()})
					if s.delay == 0 {
						s.delay = firstRepeat
					} else {
						s.delay = min(2*s.delay, maxRepeat)
					}
					s.next = now.Add(s.delay)
				}
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
