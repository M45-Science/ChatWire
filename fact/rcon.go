package fact

import (
	"ChatWire/cfg"
	"ChatWire/glob"
	"errors"
	"fmt"
	"github.com/M45-Science/rcon"
	"strings"
)

var ErrRCONOutcomeUnknown = errors.New("RCON command outcome is unknown")

func ExecuteRCON(command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", errors.New("Command required.")
	}
	address := fmt.Sprintf("127.0.0.1:%d", cfg.Local.Port+cfg.Global.Options.RconOffset)
	// The RCON client applies bounded read/write deadlines internally.
	conn, e := rcon.Dial(address, glob.RCONPass)
	if e != nil {
		return "", errors.New("RCON unavailable.")
	}
	defer conn.Close()
	id, e := conn.Write("/" + strings.TrimPrefix(command, "/"))
	if e != nil {
		return "", fmt.Errorf("RCON write failed: %w", ErrRCONOutcomeUnknown)
	}
	out, reply, e := conn.Read()
	if e != nil || reply != id {
		return "", fmt.Errorf("RCON response unavailable: %w", ErrRCONOutcomeUnknown)
	}
	if len(out) > 65536 {
		out = out[:65536] + "\n[truncated]"
	}
	return out, nil
}
