package fact

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/glob"
)

const (
	operationAnnounceDelay    = 3 * time.Second
	operationProgressThrottle = 10 * time.Second
	operationOptionalTTL      = 60 * time.Second
)

type operationStatusState struct {
	token                string
	title                string
	description          string
	startedAt            time.Time
	lastProgressKey      string
	lastProgressUpdateAt time.Time
	announced            bool
	suppressAnnounce     bool
	pendingDelayID       uint64
	optionalMessages     []operationMessageRef
}

type operationMessageRef struct {
	channelID string
	messageID string
}

var (
	operationStatusLock    sync.Mutex
	operationStatus        operationStatusState
	operationDeleteMessage = func(channelID, messageID string) error {
		if disc.DS == nil {
			return nil
		}
		return disc.DS.ChannelMessageDelete(channelID, messageID)
	}
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

func SuppressPendingOperationAnnouncement(token string) {
	if token == "" {
		return
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if token != operationStatus.token {
		return
	}
	operationStatus.suppressAnnounce = true
}

func UpdateOperation(token, title, description string, color int) {
	updateOperation(token, title, description, color, false, false)
}

func AnnouncePendingOperation(token string, color int) {
	if token == "" {
		return
	}

	operationStatusLock.Lock()
	if token != operationStatus.token || operationStatus.announced || operationStatus.suppressAnnounce {
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
		CMS(cfg.Local.Channel.ChatChannel, description)
		glob.ResetUpdateMessage()
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
	if token == "" || title == "" || description == "" {
		return
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
	firstWait := nextOptionalProgressDelay(operationStatus.startedAt, operationStatus.announced, delay)
	operationStatus.pendingDelayID++
	delayID := operationStatus.pendingDelayID
	operationStatusLock.Unlock()

	go func(tok, ttl, firstDesc string, firstWait time.Duration, id uint64) {
		time.Sleep(firstWait)
		_ = emitScheduledOperationProgress(tok, ttl, firstDesc, color, id)
	}(token, title, description, firstWait, delayID)
}

func nextOptionalProgressDelay(startedAt time.Time, announced bool, delay time.Duration) time.Duration {
	if delay <= 0 {
		return 0
	}
	if announced {
		return delay
	}

	elapsed := time.Since(startedAt)
	if elapsed < 0 {
		elapsed = 0
	}
	if elapsed < operationAnnounceDelay {
		delay += operationAnnounceDelay - elapsed
	}
	return delay
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

	optionalMessages := clearTrackedOptionalMessagesLocked()
	operationStatus.title = title
	operationStatus.description = description
	operationStatus.announced = true
	operationStatusLock.Unlock()

	deleteOperationMessages(optionalMessages)
	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		CMS(cfg.Local.Channel.ChatChannel, description)
		glob.ResetUpdateMessage()
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
		CMS(cfg.Local.Channel.ChatChannel, description)
		glob.ResetUpdateMessage()
	}
	return true
}

func trackOptionalOperationMessage(msg *discordgo.Message) (operationMessageRef, bool) {
	if msg == nil || msg.ChannelID == "" || msg.ID == "" {
		return operationMessageRef{}, false
	}
	ref := operationMessageRef{
		channelID: msg.ChannelID,
		messageID: msg.ID,
	}
	operationStatusLock.Lock()
	operationStatus.optionalMessages = append(operationStatus.optionalMessages, ref)
	operationStatusLock.Unlock()
	return ref, true
}

func scheduleOptionalOperationCleanup(ref operationMessageRef) {
	if ref.channelID == "" || ref.messageID == "" {
		return
	}

	go func(ref operationMessageRef) {
		time.Sleep(operationOptionalTTL)

		operationStatusLock.Lock()
		found := false
		for i, msg := range operationStatus.optionalMessages {
			if msg == ref {
				operationStatus.optionalMessages = append(operationStatus.optionalMessages[:i], operationStatus.optionalMessages[i+1:]...)
				found = true
				break
			}
		}
		operationStatusLock.Unlock()
		if !found {
			return
		}

		if err := operationDeleteMessage(ref.channelID, ref.messageID); err != nil {
			cwlog.DoLogCW("operation status cleanup: unable to delete message %s: %v", ref.messageID, err)
			return
		}
		resetUpdateMessageIfMatches(ref)
	}(ref)
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

func CancelOperationDelayedProgress(token string) {
	if token == "" {
		return
	}

	operationStatusLock.Lock()
	defer operationStatusLock.Unlock()
	if token != operationStatus.token {
		return
	}
	operationStatus.pendingDelayID++
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
	optionalMessages := operationStatus.optionalMessages
	operationStatus = operationStatusState{}
	operationStatusLock.Unlock()

	deleteOperationMessages(optionalMessages)
	if !announced || title == "" || description == "" {
		return
	}

	cwlog.DoLogCW("%s: %s", title, description)
	if cfg.Local.Channel.ChatChannel != "" {
		CMS(cfg.Local.Channel.ChatChannel, description)
		glob.ResetUpdateMessage()
	}
}

func clearTrackedOptionalMessagesLocked() []operationMessageRef {
	if len(operationStatus.optionalMessages) == 0 {
		return nil
	}
	msgs := append([]operationMessageRef(nil), operationStatus.optionalMessages...)
	operationStatus.optionalMessages = nil
	return msgs
}

func deleteOperationMessages(msgs []operationMessageRef) {
	for _, msg := range msgs {
		if msg.channelID == "" || msg.messageID == "" {
			continue
		}
		if err := operationDeleteMessage(msg.channelID, msg.messageID); err != nil {
			cwlog.DoLogCW("operation status cleanup: unable to delete message %s: %v", msg.messageID, err)
			continue
		}
		resetUpdateMessageIfMatches(msg)
	}
}

func resetUpdateMessageIfMatches(ref operationMessageRef) {
	current := glob.GetUpdateMessage()
	if current != nil && current.ChannelID == ref.channelID && current.ID == ref.messageID {
		glob.ResetUpdateMessage()
	}
}
