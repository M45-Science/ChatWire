package user

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/sclean"

	"github.com/bwmarrin/discordgo"
)

/* SORT FUNCTIONS */
/* Last Seen */
type ByLastSeen []glob.PlayerData

func (a ByLastSeen) Len() int           { return len(a) }
func (a ByLastSeen) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLastSeen) Less(i, j int) bool { return a[i].LastSeen > a[j].LastSeen }

/* Created time */
type ByNew []glob.PlayerData

func (a ByNew) Len() int           { return len(a) }
func (a ByNew) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNew) Less(i, j int) bool { return a[i].Creation > a[j].Creation }

/* Check if Discord moderator */
func checkModerator(roles []string) bool {

	if cfg.Global.Discord.Roles.RoleCache.Moderator == "" {
		return false
	}
	for _, r := range roles {
		if r == cfg.Global.Discord.Roles.RoleCache.Moderator {
			return true
		}
	}
	return false
}

/*  Get info on a specific player */
func Whois(s *discordgo.Session, i *discordgo.InteractionCreate) {

	var args []string = strings.Split(i.Message.Content, " ")
	layoutUS := "01-02-06 3:04 PM"

	maxresults := constants.WhoisResults
	if checkModerator(i.Member.Roles) {
		maxresults = constants.AdminWhoisResults
	}
	var slist []glob.PlayerData
	argnum := len(args)

	/* Reconstruct list, to remove empty entries and to reduce lock time */
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		slist = append(slist, *player)
	}
	glob.PlayerListLock.RUnlock()

	buf := ""

	if argnum < 1 {
		fact.CMS(cfg.Local.Channel.ChatChannel, "**Arguments:** <option>\n\n```options:\nrecent (recently online)\nnew (by time joined)\nregistered (recently registered)\n<factorio/discord name search>```")
		return
		/* SHOW RECENTLY SEEN PLAYERS */
	} else if strings.ToLower(args[0]) == "recent" {
		buf = "Recently online:\n"

		sort.Sort(ByLastSeen(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		count := 0
		for _, p := range slist {
			if p.LastSeen > 0 && count < maxresults {
				lseen := ""
				if p.LastSeen == 0 {
					lseen = constants.Unknown
				} else {
					ltime := time.Unix(p.LastSeen, 0)
					lseen = ltime.Format(layoutUS)
				}

				joined := ""
				if p.Creation == 0 {
					joined = constants.Unknown
				} else {
					jtime := time.Unix(p.Creation, 0)
					joined = jtime.Format(layoutUS)
				}
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateStringEllipsis(p.Name, 20), sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID, false), 20), lseen, joined, fact.LevelToString(p.Level))
				count++
			}
		}
		/* SHOW PLAYERS THAT JUST JOINED */
	} else if strings.ToLower(args[0]) == "new" {
		buf = "Recently joined:\n"

		sort.Sort(ByNew(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		count := 0
		for _, p := range slist {
			if p.LastSeen > 0 && count < maxresults {
				lseen := ""
				if p.LastSeen == 0 {
					lseen = constants.Unknown
				} else {
					ltime := time.Unix(p.LastSeen, 0)
					lseen = ltime.Format(layoutUS)
				}

				joined := ""
				if p.Creation == 0 {
					joined = constants.Unknown
				} else {
					jtime := time.Unix(p.Creation, 0)
					joined = jtime.Format(layoutUS)
				}
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateStringEllipsis(p.Name, 20), sclean.TruncateString(disc.GetNameFromID(p.ID, false), 20), lseen, joined, fact.LevelToString(p.Level))
				count++
			}
		}

		/* SHOW PLAYERS THAT REGISTERED */
	} else if strings.ToLower(args[0]) == "registered" {
		buf = "Recently joined and registered:\n"

		sort.Sort(ByNew(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		count := 0
		for _, p := range slist {
			if p.ID != "" && p.Name != "" {
				if p.LastSeen > 0 && count < maxresults {
					lseen := ""
					if p.LastSeen == 0 {
						lseen = constants.Unknown
					} else {
						ltime := time.Unix(p.LastSeen, 0)
						lseen = ltime.Format(layoutUS)
					}

					joined := ""
					if p.Creation == 0 {
						joined = constants.Unknown
					} else {
						jtime := time.Unix(p.Creation, 0)
						joined = jtime.Format(layoutUS)
					}
					buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateStringEllipsis(p.Name, 20), sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID, false), 20), lseen, joined, fact.LevelToString(p.Level))
					count++
				}
			}
		}

	} else {
		/*STANDARD WHOIS SEARCH*/
		count := 0
		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")
		for _, p := range slist {
			if count > maxresults {
				break
			}
			if strings.Contains(strings.ToLower(p.Name), strings.ToLower(args[0])) || strings.Contains(strings.ToLower(disc.GetNameFromID(p.ID, false)), strings.ToLower(args[0])) {

				lseen := ""
				if p.LastSeen == 0 {
					lseen = constants.Unknown
				} else {
					ltime := time.Unix(p.LastSeen, 0)
					lseen = ltime.Format(layoutUS)
				}

				joined := ""
				if p.Creation == 0 {
					joined = constants.Unknown
				} else {
					jtime := time.Unix(p.Creation, 0)
					joined = jtime.Format(layoutUS)
				}
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateStringEllipsis(p.Name, 20), sclean.TruncateStringEllipsis(disc.GetNameFromID(p.ID, false), 20), lseen, joined, fact.LevelToString(p.Level))
				count++
			}
		}
		if buf == "" {
			buf = "No results."
		}
	}

	fact.CMS(cfg.Local.Channel.ChatChannel, buf)
}
