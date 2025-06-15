package main

import (
	"bytes"
	"net"
	"strings"
	"testing"
	"time"
)

func TestStartFactorioErrors(t *testing.T) {
	procCmd = nil
	if err := startFactorio(nil); err == nil {
		t.Fatalf("expected error for nil args")
	}
	procCmd = commandFn("/bin/sh", "-c", "true")
	if err := startFactorio([]string{"/bin/sh"}); err == nil {
		t.Fatalf("expected running error")
	}
	procCmd = nil
}

func TestStartFactorioOk(t *testing.T) {
	procCmd = nil
	if err := startFactorio([]string{"/bin/sh", "-c", "exit 0"}); err != nil {
		t.Fatalf("start error: %v", err)
	}
	// wait for process to exit
	for i := 0; i < 10 && procCmd != nil; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if procCmd != nil {
		t.Fatalf("process did not exit")
	}
}

func TestWriteFactorioNil(t *testing.T) {
	procIn = nil
	writeFactorio("test") // should not panic
}

func TestHandleConnUnknown(t *testing.T) {
	c1, c2 := net.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		handleConn(c1)
	}()
	c2.Write([]byte{0xff})
	c2.Close()
	<-done
}

func TestReadStdoutLongLine(t *testing.T) {
	longLine := strings.Repeat("a", 2*1024*1024) + "\nshort\n"
	outBuf = nil
	readStdout(bytes.NewBufferString(longLine))
	if len(outBuf) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(outBuf))
	}
	if outBuf[0] != strings.Repeat("a", 2*1024*1024) {
		t.Fatalf("long line mismatch")
	}
	if outBuf[1] != "short" {
		t.Fatalf("short line mismatch: %q", outBuf[1])
	}
}
