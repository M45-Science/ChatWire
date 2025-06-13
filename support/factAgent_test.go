package support

import (
	"bytes"
	"net"
	"testing"
)

func TestAgentStartSend(t *testing.T) {
	c1, c2 := net.Pipe()
	dialFn = func() (net.Conn, error) { return c1, nil }
	agentConn = nil
	done := make(chan []byte)
	go func() {
		buf := make([]byte, 32)
		n, _ := c2.Read(buf)
		done <- buf[:n]
	}()
	if err := AgentStart("bin", []string{"a", "b"}); err != nil {
		t.Fatalf("AgentStart error: %v", err)
	}
	out := <-done
	exp := []byte{byte(agentCmdStart)}
	exp = append(exp, []byte("bin a b\n")...)
	if !bytes.Equal(out, exp) {
		t.Fatalf("unexpected bytes %v", out)
	}
	c1.Close()
	c2.Close()
}

func TestAgentStopSend(t *testing.T) {
	c1, c2 := net.Pipe()
	dialFn = func() (net.Conn, error) { return c1, nil }
	agentConn = nil
	done := make(chan []byte)
	go func() {
		buf := make([]byte, 2)
		n, _ := c2.Read(buf)
		done <- buf[:n]
	}()
	if err := AgentStop(); err != nil {
		t.Fatalf("AgentStop error: %v", err)
	}
	out := <-done
	if !bytes.Equal(out, []byte{byte(agentCmdStop)}) {
		t.Fatalf("unexpected bytes %v", out)
	}
	c1.Close()
	c2.Close()
}

func TestAgentRunning(t *testing.T) {
	c1, c2 := net.Pipe()
	dialFn = func() (net.Conn, error) { return c1, nil }
	agentConn = nil
	go func() {
		buf := make([]byte, 1)
		c2.Read(buf)
		c2.Write([]byte{1})
	}()
	if !AgentRunning() {
		t.Fatalf("expected running true")
	}
	c1.Close()
	c2.Close()
}

func TestAgentWrite(t *testing.T) {
	c1, c2 := net.Pipe()
	dialFn = func() (net.Conn, error) { return c1, nil }
	agentConn = nil
	done := make(chan []byte)
	go func() {
		buf := make([]byte, 16)
		n, _ := c2.Read(buf)
		done <- buf[:n]
	}()
	if err := AgentWrite("cmd"); err != nil {
		t.Fatalf("AgentWrite error: %v", err)
	}
	out := <-done
	if !bytes.Equal(out, []byte{byte(agentCmdWrite), 'c', 'm', 'd', '\n'}) {
		t.Fatalf("unexpected bytes %v", out)
	}
	c1.Close()
	c2.Close()
}

func TestAgentReadBuffered(t *testing.T) {
	c1, c2 := net.Pipe()
	dialFn = func() (net.Conn, error) { return c1, nil }
	agentConn = nil
	go func() {
		buf := make([]byte, 1)
		c2.Read(buf) // read request
		c2.Write([]byte("line1\nline2\n\x00"))
	}()
	lines, err := AgentReadBuffered()
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Fatalf("unexpected lines %v", lines)
	}
	c1.Close()
	c2.Close()
}
