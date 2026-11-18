package factUpdater

import (
	"ChatWire/glob"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDirectAttachmentWithProxyConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.(http.Flusher).Flush() // Chunked response with no Content-Length.
		w.Write([]byte("save"))
	}))
	defer server.Close()
	oldProxy := glob.ProxyURL
	t.Cleanup(func() { glob.ProxyURL = oldProxy })
	proxy := server.URL
	glob.ProxyURL = &proxy

	body, name, err := HttpGet(true, server.URL+"/map.zip?signature=test", true)
	if err != nil || name != "map.zip" || string(body) != "save" {
		t.Fatalf("direct attachment got name %q, body %q, error %v", name, body, err)
	}
	if _, _, err := HttpGet(false, "https://example.com/download", true); err == nil {
		t.Fatal("proxy response without a length was accepted")
	}
}
