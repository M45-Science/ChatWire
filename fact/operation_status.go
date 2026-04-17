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
	operationProgressThrottle = 10 * time.Second
	operationReminderInterval = 10 * time.Second
)

type operationStatusState struct {
	token                string
	title                string
	description          string
	startedAt            time.Time
	lastProgressKey      string
	lastProgressUpdateAt time.Time
	announced            bool
	pendingDelayID       uint64
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
		AnnouncePendingOperation(tok, glob.COLOR_CYAN)
	}(token, title, description)
	return token
}

func UpdateOperation(token, title, description string, color int) {
	updateOperation(token, title, description, color, false, false)
}

func AnnouncePendingOperation(token string, color int) {
	if token == "" {
		return
	}

	operationStatusLock.Lock()
	if token != operationStatus.token || operationStatus.announced {
		operationStatusLock.Unlock()
		return
	}
	if time.Since(operationStatus.startedAt) < operationAnnounceDelay {
		operationStatusLock.Unlock()
		return
	}
	title := operationStatus.title
	description := operationStatus.description
	operationStatus.announced = true
	operationStatusLock.Unlock()

	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), title, description, color))
	}
}

func AnnounceOperationNow(token, title, description string, color int) {
	updateOperation(token, title, description, color, false, true)
}

func UpdateOperationProgress(token, title, description string, color int) {
	updateOperation(token, title, description, color, true, false)
}

func UpdateOperationProgressDelayed(token, title, description string, color int, delay time.Duration) {
	UpdateOperationProgressDelayedWithReminder(token, title, description, description, color, delay)
}

func UpdateOperationProgressDelayedWithReminder(token, title, description, reminderDescription string, color int, delay time.Duration) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	reminderDescription = strings.TrimSpace(reminderDescription)
	if token == "" || title == "" || description == "" {
		return
	}
	if reminderDescription == "" {
		reminderDescription = description
	}
	if delay <= 0 {
		UpdateOperationProgress(token, title, description, color)
		return
	}

	operationStatusLock.Lock()
	if token != operationStatus.token {
		operationStatusLock.Unlock()
		return
	}
	operationStatus.pendingDelayID++
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	go func(tok, ttl, firstDesc, repeatDesc string, firstWait time.Duration, id uint64) {
		wait := firstWait
		desc := firstDesc
		for {
			time.Sleep(wait)
			if !emitScheduledOperationProgress(tok, ttl, desc, color, id) {
				return
			}
			desc = repeatDesc
			wait = operationReminderInterval
		}
	}(token, title, description, reminderDescription, delay, delayID)
}

func updateOperation(token, title, description string, color int, throttled bool, force bool) {
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
	operationStatus.pendingDelayID++

	now := time.Now()
	if !force && !operationStatus.announced && now.Sub(operationStatus.startedAt) < operationAnnounceDelay {
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

func emitScheduledOperationProgress(token, title, description string, color int, delayID uint64) bool {
	operationStatusLock.Lock()
	if token != operationStatus.token || delayID != operationStatus.pendingDelayID {
		operationStatusLock.Unlock()
		return false
	}

	now := time.Now()
	progressKey := title + "\n" + description
	if progressKey == operationStatus.lastProgressKey && now.Sub(operationStatus.lastProgressUpdateAt) < operationProgressThrottle {
		operationStatusLock.Unlock()
		return true
	}

	operationStatus.title = title
	operationStatus.description = description
	operationStatus.lastProgressKey = progressKey
	operationStatus.lastProgressUpdateAt = now
	operationStatus.announced = true
	operationStatusLock.Unlock()

	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), title, description, color))
	}
	return true
}

func CompleteOperation(token, title, description string, color int) {
	finalizeOperation(token, title, description, color)
}

func FailOperation(token, title, description string, color int) {
	finalizeOperation(token, title, description, color)
}

func CancelOperation(token string) {
	finalizeOperation(token, "", "", 0)
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
