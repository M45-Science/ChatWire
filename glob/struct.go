package glob

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// PlayerData stores persistent information about a single player.
type PlayerData struct {
	Name      string `json:"-"`
	Level     int    `json:"l,omitempty"`
	ID        string `json:"i,omitempty"`
	BanReason string `json:"b,omitempty"`
	Creation  int64  `json:"c,omitempty"`
	LastSeen  int64  `json:"s,omitempty"`
	Minutes   int64  `json:"m,omitempty"`
	SusScore  int64  `json:"u,omitempty"`

	/* Not on disk */
	AlreadyBanned bool  `json:"-"`
	SpamScore     int64 `json:"-"`
}

// PassData represents a temporary registration code for Discord linkage.
type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

// PanelTokenData represents a temporary web panel token.
type PanelTokenData struct {
	Token  string
	DiscID string
	Time   int64
}

// VoteContainerData holds map vote data for the current campaign.
type VoteContainerData struct {
	Version string
	Votes   []MapVoteData

	/* Temporary storage for tallying votes */
	Tally         []VoteTallyData `json:"-"`
	LastMapChange time.Time       `json:"-"`
	NumChanges    int

	Dirty bool `json:"-"`
}

// MapVoteData stores an individual vote for a map reset.
type MapVoteData struct {
	Name   string
	DiscID string

	Moderator bool
	Supporter bool
	Veteran   bool

	TotalVotes int

	Selection  string
	NumChanges int

	Time    time.Time
	Voided  bool
	Expired bool
}

// VoteTallyData temporarily accumulates vote counts during tallies.
type VoteTallyData struct {
	Selection string
	Count     int
}

// OnlinePlayerData describes a player entry from the /online softmod command.
type OnlinePlayerData struct {
	Name       string
	ScoreTicks int
	TimeTicks  int
	Level      int
	AFK        string
}

// AppCmdData describes a Discord application command.
type AppCmdData struct {
	Name, Description        string
	Options                  []OptionData
	Type                     discordgo.ApplicationCommandType
	DefaultMemberPermissions *int64
}

// CommandData configures a ChatWire slash command handler.
type CommandData struct {
	Function      func(cmd *CommandData, i *discordgo.InteractionCreate)
	ModeratorOnly bool
	AdminOnly     bool

	PrimaryOnly bool
	Global      bool
	Disabled    bool
	AppCmd      AppCmdData
}

// OptionData defines a single option for a command.
type OptionData struct {
	Name, Description string
	Type              discordgo.ApplicationCommandOptionType
	Required          bool
	MinValue          *float64
	MaxValue          *float64

	Choices []ChoiceData
}

// ChoiceData specifies a selectable option value.
type ChoiceData struct {
	Name     string
	Value    interface{}
	Function func(cmd *CommandData, i *discordgo.InteractionCreate)
}
