package moderator

import "testing"

func TestShortFactorioVersion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   string
		wantOK bool
	}{
		{name: "full semver", input: "1.2.3", want: "1.2", wantOK: true},
		{name: "release candidate", input: "1.2.3-rc1", want: "1.2", wantOK: true},
		{name: "too short", input: "1.2", want: "", wantOK: false},
		{name: "unknown", input: "Unknown", want: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := shortFactorioVersion(tt.input)
			if ok != tt.wantOK {
				t.Fatalf("ok mismatch: got %v want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("version mismatch: got %q want %q", got, tt.want)
			}
		})
	}
}
