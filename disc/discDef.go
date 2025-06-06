package disc

import (
	"sync"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
)

var (
	// Guild is the Discord guild ChatWire is connected to.
	Guild *discordgo.Guild
	// Guildname is the name of the connected guild.
	Guildname = constants.Unknown
	// DS is the active Discord session.
	DS *discordgo.Session

	// CMSBuffer holds queued messages destined for Discord.
	CMSBuffer     []CMSBuf
	CMSBufferLock sync.Mutex

	// OldChanName caches the previous channel name during updates.
	OldChanName = constants.Unknown
	// NewChanName caches the new channel name during updates.
	NewChanName       = constants.Unknown
	UpdateChannelLock sync.Mutex
)

// CMSBuf represents a queued Discord message.
type CMSBuf struct {
	Channel string
	Text    string
}

// RoleListData caches lists of Discord users for specific roles.
type RoleListData struct {
	Version      string
	Patreons     []string
	Supporters   []string
	NitroBooster []string
	Moderators   []string
}
