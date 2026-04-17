package fact

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

const (
	operationAnnounceDelay    = 3 * time.Second
	operationProgressThrottle = 15 * time.Second
)

type operationStatusState struct {
	token                string
	title                string
	description          string
	startedAt            time.Time
	lastProgressKey      string
	lastProgressUpdateAt time.Time
	announced            bool
}

var (
	operationStatusLock sync.Mutex
	operationStatus     operationStatusState
)

func BeginOperation(title, description string) string {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || description == "" {
		return ""
	}

	token := fmt.Sprintf("op-%d", time.Now().UnixNano())
	operationStatusLock.Lock()
	operationStatus = operationStatusState{
		token:       token,
		title:       title,
		description: description,
		startedAt:   time.Now(),
	}
	operationStatusLock.Unlock()

	go func(tok, ttl, desc string) {
		time.Sleep(operationAnnounceDelay)
		UpdateOperation(tok, ttl, desc, glob.COLOR_CYAN)
	}(token, title, description)
	return token
}

func UpdateOperation(token, title, description string, color int) {
	updateOperation(token, title, description, color, false)
}

func UpdateOperationProgress(token, title, description string, color int) {
	updateOperation(token, title, description, color, true)
}

func updateOperation(token, title, description string, color int, throttled bool) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if token == "" || title == "" || description == "" {
		return
	}

	operationStatusLock.Lock()
	if token != operationStatus.token {
		operationStatusLock.Unlock()
		return
	}

	now := time.Now()
	if !operationStatus.announced && now.Sub(operationStatus.startedAt) < operationAnnounceDelay {
		operationStatus.title = title
		operationStatus.description = description
		operationStatusLock.Unlock()
		return
	}

	if throttled {
		progressKey := title + "\n" + description
		if progressKey == operationStatus.lastProgressKey && now.Sub(operationStatus.lastProgressUpdateAt) < operationProgressThrottle {
			operationStatusLock.Unlock()
			return
		}
		operationStatus.lastProgressKey = progressKey
		operationStatus.lastProgressUpdateAt = now
	}

	operationStatus.title = title
	operationStatus.description = description
	operationStatus.announced = true
	operationStatusLock.Unlock()

	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), title, description, color))
	}
}

func CompleteOperation(token, title, description string, color int) {
	finalizeOperation(token, title, description, color)
}

func FailOperation(token, title, description string, color int) {
	finalizeOperation(token, title, description, color)
}

func finalizeOperation(token, title, description string, color int) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if token == "" {
		return
	}

	operationStatusLock.Lock()
	if token != operationStatus.token {
		operationStatusLock.Unlock()
		return
	}
	announced := operationStatus.announced || time.Since(operationStatus.startedAt) >= operationAnnounceDelay
	operationStatus = operationStatusState{}
	operationStatusLock.Unlock()

	if !announced || title == "" || description == "" {
		return
	}

	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), title, description, color))
	}
}
