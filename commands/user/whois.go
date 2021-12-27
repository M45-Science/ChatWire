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

//Last Seen
type ByLastSeen []glob.PlayerData

func (a ByLastSeen) Len() int           { return len(a) }
func (a ByLastSeen) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByLastSeen) Less(i, j int) bool { return a[i].LastSeen > a[j].LastSeen }

//Created time
type ByNew []glob.PlayerData

func (a ByNew) Len() int           { return len(a) }
func (a ByNew) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByNew) Less(i, j int) bool { return a[i].Creation > a[j].Creation }

func levelToString(level int) string {

	name := "Error"

	if level <= -254 {
		name = "Deleted"
	} else if level == -1 {
		name = "Banned"
	} else if level == 0 {
		name = "New"
	} else if level == 1 {
		name = "Member"
	} else if level == 2 {
		name = "Regular"
	} else if level >= 255 {
		name = "Admin"
	}

	return name
}

// CheckAdmin checks if the user attempting to run an admin command is an admin
func checkadmin(m *discordgo.MessageCreate) bool {
	for _, role := range m.Member.Roles {
		if role == cfg.Global.RoleData.Moderator {
			return true
		}
	}
	for _, admin := range cfg.Global.AdminData.IDs {
		if m.Author.ID == admin {
			return true
		}
	}
	return false
}

func Whois(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {

	layoutUS := "01-02-06 3:04 PM"

	maxresults := constants.WhoisResults
	if checkadmin(m) {
		maxresults = constants.AdminWhoisResults
	}
	var slist []glob.PlayerData
	argnum := len(args)

	//Reconstruct list, to remove empty entries and to reduce lock time
	glob.PlayerListLock.RLock()
	for _, player := range glob.PlayerList {
		slist = append(slist, *player)
	}
	glob.PlayerListLock.RUnlock()

	buf := ""

	if argnum < 1 {
		fact.CMS(m.ChannelID, "**Arguments:** <option>\n\n```options:\nrecent (recently online)\nnew (by time joined)\nregistered (recently registered)\n<factorio/discord name search>```")
		return
	} else if strings.ToLower(args[0]) == "recent" {
		buf = "Recently online:\n"

		sort.Sort(ByLastSeen(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		tnow := time.Now()
		tnow = tnow.Round(time.Second)

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
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateString(p.Name, 20), sclean.TruncateString(disc.GetNameFromID(p.ID, false), 20), lseen, joined, levelToString(p.Level))
				count++
			}
		}

	} else if strings.ToLower(args[0]) == "new" {
		buf = "Recently joined:\n"

		sort.Sort(ByNew(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		tnow := time.Now()
		tnow = tnow.Round(time.Second)

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
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateString(p.Name, 20), sclean.TruncateString(disc.GetNameFromID(p.ID, false), 20), lseen, joined, levelToString(p.Level))
				count++
			}
		}

	} else if strings.ToLower(args[0]) == "registered" {
		buf = "Recently joined and registered:\n"

		sort.Sort(ByNew(slist))

		buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", "Factorio Name", "Discord Name", "Last Seen", "Joined", "Level")

		tnow := time.Now()
		tnow = tnow.Round(time.Second)

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
					buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateString(p.Name, 20), sclean.TruncateString(disc.GetNameFromID(p.ID, false), 20), lseen, joined, levelToString(p.Level))
					count++
				}
			}
		}

	} else {

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
				buf = buf + fmt.Sprintf("`%20s : %20s : %18s : %18s : %7s`\n", sclean.TruncateString(p.Name, 20), sclean.TruncateString(disc.GetNameFromID(p.ID, false), 20), lseen, joined, levelToString(p.Level))
			}
		}
		if buf == "" {
			buf = "No results."
		}
	}

	fact.CMS(m.ChannelID, buf)
}
