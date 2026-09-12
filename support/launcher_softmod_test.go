package support

import "testing"

func TestKeepSaveEntryRequiresRootFolder(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{name: "world/level.dat", want: true},
		{name: "world/level.dat0", want: true},
		{name: "world/settings.json", want: true},
		{name: "world/script.dat", want: true},
		{name: "other/settings.json", want: false},
		{name: "world/nested/settings.json", want: false},
		{name: "world/control.lua", want: false},
	}

	for _, tc := range tests {
		if got := keepSaveEntry(tc.name, "world"); got != tc.want {
			t.Errorf("keepSaveEntry(%q) = %v, want %v", tc.name, got, tc.want)
		}
	}
}
