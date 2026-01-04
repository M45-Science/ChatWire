package support

import (
	"strings"

	"ChatWire/glob"
	"ChatWire/sclean"
)

/* Check if an idiot pasted their register code to a chat channel */
func ProtectIdiots(text string) bool {
	idiotID := ""
	checkme := strings.ToLower(text)
	checkme = sclean.AlphaOnly(checkme)

	/* Only run if there are active registration codes */
	if len(glob.PassList) > 0 {
		for i, o := range glob.PassList {
			password := strings.ToLower(o.Code)
			password = sclean.AlphaOnly(password)

			/* Found an active register code */
			if strings.Contains(checkme, password) {
				idiotID = i
				break
			}

			/* Just in case they miss part of it when copying/pasting */
			plen := len(password)
			if plen > 3 {
				trimEnd := password[0 : plen-2]
				trimStart := password[2:plen]

				if strings.Contains(checkme, trimEnd) {
					idiotID = i
					break
				} else if strings.Contains(checkme, trimStart) {
					idiotID = i
					break
				}
			}
		}
		/* We got one, invalidate their code */
		if idiotID != "" {
			delete(glob.PassList, idiotID)
			return true
		}
		return false
	}

	return false
}
