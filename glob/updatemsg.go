package glob

import "github.com/bwmarrin/discordgo"

// SetUpdateMessage safely stores the update message pointer.
func SetUpdateMessage(m *discordgo.Message) {
	UpdateMessageLock.Lock()
	defer UpdateMessageLock.Unlock()
	UpdateMessage = m
}

// GetUpdateMessage safely retrieves the update message pointer.
func GetUpdateMessage() *discordgo.Message {
	UpdateMessageLock.Lock()
	defer UpdateMessageLock.Unlock()
	m := UpdateMessage
	return m
}

// ResetUpdateMessage clears the stored update message.
func ResetUpdateMessage() {
	SetUpdateMessage(nil)
}
