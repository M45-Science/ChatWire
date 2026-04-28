package support

import (
	"strings"
	"testing"
)

func TestSyncModDownloadName(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{
			name: "relative download path",
			line: "0.000 Info HttpSharedState.cpp:1: Downloading /download/some-mod/file.zip",
			want: "some-mod",
			ok:   true,
		},
		{
			name: "full download url",
			line: "0.000 Info HttpSharedState.cpp:1: Downloading https://mods.factorio.com/download/some-mod/file.zip",
			want: "some-mod",
			ok:   true,
		},
		{
			name: "missing url",
			line: "0.000 Info HttpSharedState.cpp:1: Downloading",
		},
		{
			name: "not download line",
			line: "0.000 Loading mod base 2.0.72",
		},
		{
			name: "download path without mod name",
			line: "0.000 Info HttpSharedState.cpp:1: Downloading /download/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := syncModDownloadName(strings.Fields(tt.line))
			if ok != tt.ok {
				t.Fatalf("ok mismatch: got %v want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("name mismatch: got %q want %q", got, tt.want)
			}
		})
	}
}
