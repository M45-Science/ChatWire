package glob

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"math"
	"os"
	"sync"
	"time"

	"ChatWire/constants"

	"github.com/bwmarrin/discordgo"
)

var FactorioLock sync.Mutex

func Ptr[T any](v T) *T {
	return &v
}

/* Most of these are here, because go is unable to handle import cycles */
/* I wish they would address this issue, because many people avoid packages due to this */

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

var (
	FactorioContext context.Context
	FactorioCancel  context.CancelFunc

	AlphaValue    map[string]int
	RCONPass      string
	OnlineCommand = constants.OnlineCommand
	OnlinePlayers []OnlinePlayerData

	/* Boot flags */
	DoRegisterCommands   *bool
	DoDeregisterCommands *bool
	LocalTestMode        *bool
	NoAutoLaunch         *bool

	/* Vote map */
	VoteBox     VoteContainerData
	VoteBoxLock sync.Mutex

	/* Server status */
	ServerRunning  bool = true
	Uptime         time.Time
	SoftModVersion = constants.Unknown

	/* Log data */
	GameLogName = ""
	CWLogName   = ""
	GameLogDesc *os.File
	CWLogDesc   *os.File

	/* CW reboot */
	DoRebootCW = false

	/* Increasing relaunch delay */
	RelaunchThrottle = 0

	/* Player database */
	PlayerList          map[string]*PlayerData
	PlayerListLock      sync.RWMutex
	PlayerListWriteLock sync.Mutex

	/* Registration codes */
	PassList         map[string]*PassData
	PasswordListLock sync.RWMutex

	/* Player database status */
	PlayerListUpdated       = false
	PlayerListUpdatedLock   sync.Mutex
	PlayerListDirty         = false
	PlayerListDirtyLock     sync.Mutex
	PlayerListSeenDirty     = false
	PlayerListSeenDirtyLock sync.Mutex

	/* Factorio server watchdog */
	NoResponseCount = 0

	/* Update warning */
	UpdateWarnCounter  = 0
	UpdateGraceMinutes = 5
	UpdateZipAttempts  = 0

	/* Chat spam */
	ChatterLock      sync.Mutex
	ChatterList      map[string]time.Time
	ChatterSpamScore map[string]int

	//Message throttles
	LastSusWarning  time.Time
	LastCrashReport time.Time

	/* Pause command */
	PausedForConnect     bool
	PausedCount          int
	PausedAt             time.Time
	PausedConnectAttempt bool
	PausedFor            string
	PausedLock           sync.Mutex
)

/* Used for map names */
func RandomBase64String(l int) string {
	buff := make([]byte, int(math.Ceil(float64(l)/float64(1.33333333333))))
	_, _ = rand.Read(buff)

	str := base64.RawURLEncoding.EncodeToString(buff)
	/* strip 1 extra character we get from odd length results */
	return str[:l]
}
