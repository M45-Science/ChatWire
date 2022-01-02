package disc

import (
	"ChatWire/constants"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

/* Discord data */
var Guild *discordgo.Guild
var Guildname = constants.Unknown
var GuildLock sync.RWMutex
var DS *discordgo.Session

/*To-Discord message buffer*/
var CMSBuffer []CMSBuf
var CMSBufferLock sync.Mutex

/* Channel name data */
var OldChanName = constants.Unknown
var NewChanName = constants.Unknown
var UpdateChannelLock sync.Mutex

/* To-Discord message buffer */
type CMSBuf struct {
	Added   time.Time
	Channel string
	Text    string
}
