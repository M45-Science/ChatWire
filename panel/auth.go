package panel

import (
	"time"

	"ChatWire/constants"
	"ChatWire/glob"
)

// GenerateToken creates a temporary token for web access.
func GenerateToken(id string) string {
	now := time.Now().Unix()
	token := glob.RandomBase64String(128)
	orig := now
	glob.PanelTokenLock.Lock()
	for k, v := range glob.PanelTokens {
		if v.DiscID == id {
			if v.Orig < orig {
				orig = v.Orig
			}
			delete(glob.PanelTokens, k)
		}
	}
	if now-orig > constants.PanelTokenLimitSec {
		orig = now
	}
	glob.PanelTokens[token] = &glob.PanelTokenData{Token: token, DiscID: id, Time: now, Orig: orig, IP: ""}
	glob.PanelTokenLock.Unlock()
	return token
}

func tokenValid(tok string) bool {
	glob.PanelTokenLock.RLock()
	data, ok := glob.PanelTokens[tok]
	glob.PanelTokenLock.RUnlock()
	if !ok {
		return false
	}
	now := time.Now().Unix()
	if now-data.Time > constants.PassExpireSec {
		return false
	}
	if now-data.Orig > constants.PanelTokenLimitSec {
		return false
	}
	return true
}
