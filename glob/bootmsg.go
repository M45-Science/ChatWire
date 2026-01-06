package glob

import "github.com/bwmarrin/discordgo"

// SetBootMessage safely stores the boot message pointer.
func SetBootMessage(m *discordgo.Message) {
	BootMessageLock.Lock()
	defer BootMessageLock.Unlock()
	BootMessage = m
}

// GetBootMessage safely retrieves the boot message pointer.
func GetBootMessage() *discordgo.Message {
	BootMessageLock.RLock()
	defer BootMessageLock.RUnlock()
	m := BootMessage
	return m
}
