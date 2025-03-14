package glob

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

/* Player database */
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

/* Registrarion codes */
type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

/* Votes Container */
type VoteContainerData struct {
	Version string
	Votes   []MapVoteData

	/* Temporary storage for tallying votes */
	Tally         []VoteTallyData `json:"-"`
	LastMapChange time.Time       `json:"-"`
	NumChanges    int

	Dirty bool `json:"-"`
}

/* Votes */
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

/* Temporary storage for tallying votes */
type VoteTallyData struct {
	Selection string
	Count     int
}

/* From softmod /online command */
type OnlinePlayerData struct {
	Name       string
	ScoreTicks int
	TimeTicks  int
	Level      int
	AFK        string
}

type AppCmdData struct {
	Name, Description        string
	Options                  []OptionData
	Type                     discordgo.ApplicationCommandType
	DefaultMemberPermissions *int64
}

type CommandData struct {
	Function      func(cmd *CommandData, i *discordgo.InteractionCreate)
	ModeratorOnly bool
	AdminOnly     bool

	PrimaryOnly bool
	Global      bool
	Disabled    bool
	AppCmd      AppCmdData
}

type OptionData struct {
	Name, Description string
	Type              discordgo.ApplicationCommandOptionType
	Required          bool
	MinValue          *float64
	MaxValue          *float64

	Choices []ChoiceData
}

type ChoiceData struct {
	Name     string
	Value    interface{}
	Function func(cmd *CommandData, i *discordgo.InteractionCreate)
}
