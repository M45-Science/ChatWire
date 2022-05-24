package disc

import (
	"sync"

	"github.com/bwmarrin/discordgo"

	"ChatWire/constants"
)

var (
	/* Discord data */
	Guild     *discordgo.Guild
	Guildname = constants.Unknown
	DS        *discordgo.Session

	/*To-Discord message buffer*/
	CMSBuffer     []CMSBuf
	CMSBufferLock sync.Mutex

	/* Channel name data */
	OldChanName       = constants.Unknown
	NewChanName       = constants.Unknown
	UpdateChannelLock sync.Mutex
)

/* To-Discord message buffer */
type CMSBuf struct {
	Channel string
	Text    string
}

/* Cache of Players with specific Discord roles*/
type RoleListData struct {
	Version      string
	Patreons     []string
	NitroBooster []string
	Moderators   []string
}
