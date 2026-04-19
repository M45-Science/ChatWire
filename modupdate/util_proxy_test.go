package modupdate

import (
	"testing"

	"ChatWire/glob"
)

func TestShouldThrottleModPortalRequestWithoutProxy(t *testing.T) {
	oldProxy := glob.ProxyURL
	defer func() {
		glob.ProxyURL = oldProxy
	}()

	glob.ProxyURL = nil
	if !shouldThrottleModPortalRequest() {
		t.Fatal("expected throttling when proxy url is unset")
	}

	empty := ""
	glob.ProxyURL = &empty
	if !shouldThrottleModPortalRequest() {
		t.Fatal("expected throttling when proxy url is empty")
	}
}

func TestShouldThrottleModPortalRequestWithProxy(t *testing.T) {
	oldProxy := glob.ProxyURL
	defer func() {
		glob.ProxyURL = oldProxy
	}()

	proxy := "http://proxy.example"
	glob.ProxyURL = &proxy
	if shouldThrottleModPortalRequest() {
		t.Fatal("expected throttling to be disabled in proxy mode")
	}
}
