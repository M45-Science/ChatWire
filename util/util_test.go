package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrimPrefixIgnoreCase(t *testing.T) {
	got := TrimPrefixIgnoreCase("HelloWorld", "hello")
	if got != "World" {
		t.Fatalf("expected World got %q", got)
	}
	got = TrimPrefixIgnoreCase("ChatWire", "wire")
	if got != "ChatWire" {
		t.Fatalf("no prefix should return input, got %q", got)
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	if !ContainsIgnoreCase("Hello World", "world") {
		t.Fatalf("expected true")
	}
	if ContainsIgnoreCase("abc", "D") {
		t.Fatalf("expected false")
	}
}

func TestStringToBool(t *testing.T) {
	v, err := StringToBool("TRUE")
	if v != true || err {
		t.Fatalf("expected true,false got %v,%v", v, err)
	}
	v, err = StringToBool("no")
	if v != false || err {
		t.Fatalf("expected false,false got %v,%v", v, err)
	}
	v, err = StringToBool("maybe")
	if err != true {
		t.Fatalf("expected conversion error")
	}
	if v != false {
		t.Fatalf("expected value false when error")
	}
}

func TestBoolToOnOff(t *testing.T) {
	if BoolToOnOff(true) != "on" {
		t.Fatalf("true should be on")
	}
	if BoolToOnOff(false) != "off" {
		t.Fatalf("false should be off")
	}
}

func TestClearOldSignals(t *testing.T) {
	dir := t.TempDir()
	files := []string{".start", ".queue", ".stop", ".rebootcw"}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer os.Chdir(oldwd)
	os.Chdir(dir)
	for _, f := range files {
		os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o644)
	}
	ClearOldSignals()
	for _, f := range files {
		if _, err := os.Stat(filepath.Join(dir, f)); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed", f)
		}
	}
}
