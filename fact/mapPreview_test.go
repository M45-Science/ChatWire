package fact

import (
	"encoding/base64"
	"io"
	"strings"
	"testing"

	"ChatWire/cfg"
	"ChatWire/disc"
)

func TestMapPreviewURLPreservesExchange(t *testing.T) {
	url, err := MapPreviewURL(" \n" + sampleMapExchangeString + "\r\n")
	if err != nil {
		t.Fatal(err)
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(url, "https://mapgen.m45sci.xyz/#m="))
	if err != nil {
		t.Fatal(err)
	}
	restored := ">>>" + base64.StdEncoding.EncodeToString(data) + "<<<"
	if restored != sampleMapExchangeString {
		t.Fatal("share link changed the map exchange data")
	}
	if _, err := ParseMapExchangeString(restored); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []string{"", ">>><<<", ">>>??<<<", "YWJj", strings.Repeat("a", 512001)} {
		if _, err := MapPreviewURL(invalid); err == nil {
			t.Fatalf("accepted invalid exchange %q", invalid[:min(len(invalid), 20)])
		}
	}
}

func TestAnnounceMapPreviewQueuesServerChat(t *testing.T) {
	drainMapResetCMS()
	t.Cleanup(drainMapResetCMS)
	old := cfg.Local.Channel.ChatChannel
	cfg.Local.Channel.ChatChannel = "preview-test-channel"
	t.Cleanup(func() { cfg.Local.Channel.ChatChannel = old })
	if err := AnnounceMapPreview(sampleMapExchangeString); err != nil {
		t.Fatal(err)
	}
	want, _ := MapPreviewURL(sampleMapExchangeString)
	select {
	case message := <-disc.CMSChan:
		if message.Channel != "preview-test-channel" || message.Text != "" || message.Embed == nil {
			t.Fatalf("wrong chat announcement: %+v", message)
		}
		if message.Embed.Title != "New map" || message.Embed.Description != "[Preview map]("+want+")" {
			t.Fatalf("wrong preview embed: %+v", message.Embed)
		}
	default:
		t.Fatal("no chat announcement queued")
	}
}

func TestLongMapPreviewLinkIsAttachedIntact(t *testing.T) {
	url := "https://mapgen.m45sci.xyz/#m=" + strings.Repeat("a", 2500)
	message := mapPreviewMessage(url)
	if len(message.Files) != 1 {
		t.Fatal("long link was not attached")
	}
	data, err := io.ReadAll(message.Files[0].Reader)
	if err != nil || string(data) != url+"\n" {
		t.Fatalf("attachment truncated the link: %v", err)
	}
}
