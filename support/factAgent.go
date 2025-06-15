package support

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"ChatWire/cwlog"
	"ChatWire/fact"
	"ChatWire/glob"
)

const agentSocket = "factorio-agent.sock"

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

func AgentStart(bin string, args []string) error {
	socketLock.Lock()
	defer socketLock.Unlock()
	conn, err := getConn()
	if err != nil {
		return err
	}
	buf := []byte{byte(agentCmdStart)}
	all := append([]string{bin}, args...)
	if len(all) > 0 {
		buf = append(buf, []byte(strings.Join(all, " ")+"\n")...)
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

// AgentWatch listens for buffered output notifications from the Factorio agent.
// When data is available it reads the buffered lines and sends them on the
// provided channel. The context can be used to stop the goroutine.
func AgentWatch(ctx context.Context, out chan<- []string) error {
	go func() {
		var conn net.Conn
		var r *bufio.Reader
		var err error
		for {
			if ctx.Err() != nil {
				if conn != nil {
					conn.Close()
				}
				return
			}
			if conn == nil {
				conn, err = dialFn()
				if err != nil {
					cwlog.DoLogCW("Agent watch dial error: %v", err)
					time.Sleep(time.Second)
					continue
				}
				r = bufio.NewReader(conn)
			}
			b1, err := r.ReadByte()
			if err != nil {
				cwlog.DoLogCW("Agent watch read error: %v", err)
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			if agentCmd(b1) != agentCmdStop {
				continue
			}
			b2, err := r.ReadByte()
			if err != nil {
				cwlog.DoLogCW("Agent watch read error: %v", err)
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			if b2 != agentNotifyBuffered {
				continue
			}
			if _, err := conn.Write([]byte{byte(agentCmdRead)}); err != nil {
				cwlog.DoLogCW("Agent watch write error: %v", err)
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			data, err := r.ReadBytes(0)
			if err != nil {
				cwlog.DoLogCW("Agent watch read error: %v", err)
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			data = data[:len(data)-1]
			if len(data) == 0 {
				continue
			}
			lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
			select {
			case out <- lines:
			case <-ctx.Done():
				conn.Close()
				return
			}
		}
	}()
	return nil
}

// AttachRunningFactorio checks if the Factorio agent is already running a server
// and, if so, sets up IPC pipes and log reading so ChatWire can interact with it.
// It returns true if a running instance was detected.
func AttachRunningFactorio(ctx context.Context) bool {
	if !AgentRunning() {
		return false
	}

	cwlog.DoLogCW("Detected running Factorio instance via agent. Attaching...")

	if ctx == nil {
		ctx = context.Background()
	}
	glob.FactorioContext, glob.FactorioCancel = context.WithCancel(ctx)

	fact.GameBuffer = new(bytes.Buffer)
	fact.PipeLock.Lock()
	fact.Pipe = NewAgentWriter()
	fact.PipeLock.Unlock()

	bufCh := make(chan []string, 10)
	if err := AgentWatch(glob.FactorioContext, bufCh); err != nil {
		cwlog.DoLogCW("Agent watch error: %v", err)
	}
	go func() {
		for {
			select {
			case lines := <-bufCh:
				for _, l := range lines {
					fact.GameBuffer.WriteString(l + "\n")
				}
			case <-glob.FactorioContext.Done():
				return
			}
		}
	}()

	fact.FactorioBooted = true
	fact.FactorioBootedAt = time.Now()
	fact.SetFactRunning(true, false)
	fact.WriteFact("/sversion")
	fact.WriteFact(glob.OnlineCommand)
	return true
}
