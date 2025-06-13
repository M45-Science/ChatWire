package support

import (
	"bufio"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var agentSocket = func() string {
	ex, err := os.Executable()
	if err != nil {
		return filepath.Join("agent", "factorio-agent.sock")
	}
	return filepath.Join(filepath.Dir(ex), "agent", "factorio-agent.sock")
}()

type agentCmd byte

const (
	agentCmdStart agentCmd = iota + 1
	agentCmdStop
	agentCmdRunning
	agentCmdWrite
	agentCmdRead
)

const agentNotifyBuffered byte = 0x06
const agentRespOK byte = byte(agentCmdStart)

var (
	agentConn  net.Conn
	connLock   sync.Mutex
	socketLock sync.Mutex
	dialFn     = func() (net.Conn, error) { return net.Dial("unix", agentSocket) }
)

func getConn() (net.Conn, error) {
	connLock.Lock()
	defer connLock.Unlock()
	if agentConn != nil {
		return agentConn, nil
	}
	c, err := dialFn()
	if err != nil {
		return nil, err
	}
	agentConn = c
	return agentConn, nil
}

// agentWriter adapts the agent connection to an io.WriteCloser so that
// existing code writing to fact.Pipe can send commands through the agent.
type agentWriter struct{}

func (agentWriter) Write(p []byte) (int, error) {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return 0, err
	}
	_, err = conn.Write(append([]byte{byte(agentCmdWrite)}, p...))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func (agentWriter) Close() error { return nil }

// NewAgentWriter returns an io.WriteCloser that sends input to the Factorio agent.
func NewAgentWriter() io.WriteCloser { return agentWriter{} }

func AgentStart(args []string) error {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return err
	}
	buf := []byte{byte(agentCmdStart)}
	if len(args) > 0 {
		buf = append(buf, []byte(strings.Join(args, " ")+"\n")...)
	} else {
		buf = append(buf, '\n')
	}
	_, err = conn.Write(buf)
	return err
}

func AgentStop() error {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte{byte(agentCmdStop)})
	return err
}

func AgentRunning() bool {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return false
	}
	if _, err = conn.Write([]byte{byte(agentCmdRunning)}); err != nil {
		return false
	}
	resp := make([]byte, 1)
	if _, err = conn.Read(resp); err != nil {
		return false
	}
	return resp[0] == 1
}

func AgentWrite(line string) error {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return err
	}
	_, err = conn.Write(append([]byte{byte(agentCmdWrite)}, []byte(line+"\n")...))
	return err
}

func AgentReadBuffered() ([]string, error) {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return nil, err
	}
	if _, err = conn.Write([]byte{byte(agentCmdRead)}); err != nil {
		return nil, err
	}
	r := bufio.NewReader(conn)
	data, err := r.ReadBytes(0)
	if err != nil {
		return nil, err
	}
	data = data[:len(data)-1]
	if len(data) == 0 {
		return nil, nil
	}
	out := strings.Split(string(data), "\n")
	if out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out, nil
}
