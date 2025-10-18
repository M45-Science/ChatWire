package cfg

import (
	"errors"
	"testing"
)

func TestSetLocalDefaultsExecutableError(t *testing.T) {
	origExecLookup := executableLookup
	executableLookup = func() (string, error) {
		return "", errors.New("executable lookup failed")
	}
	t.Cleanup(func() {
		executableLookup = origExecLookup
	})

	origLocal := Local
	t.Cleanup(func() {
		Local = origLocal
	})

	Local = local{
		Name: "existing",
		Port: 1,
		Channel: channel{
			ChatChannel: "existing",
		},
		Options: localOptions{
			SoftModOptions: softmodOptions{
				SoftModPath: "existing",
			},
		},
	}

	setLocalDefaults()

	if Local.Callsign != "cw" {
		t.Fatalf("expected fallback callsign 'cw', got %q", Local.Callsign)
	}
}
