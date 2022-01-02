package glob

/* Player database */
type PlayerData struct {
	Name     string
	Level    int
	ID       string
	Creation int64
	LastSeen int64
}

/* Registrarion codes */
type PassData struct {
	Code   string
	DiscID string
	Time   int64
}

type RoleListData struct {
	Version      string
	Patreons     []string
	NitroBooster []string
	Moderators   []string
}
