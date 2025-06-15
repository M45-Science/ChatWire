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

	"ChatWire/cfg"
	"ChatWire/cwlog"
	"ChatWire/disc"
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
	agentConn    net.Conn
	connLock     sync.Mutex
	socketLock   sync.Mutex
	dialFn       = func() (net.Conn, error) { return net.Dial("unix", agentSocket) }
	retryingConn bool
	alertAgent   bool
)

func markBadConn() {
	connLock.Lock()
	if agentConn != nil {
		agentConn.Close()
		agentConn = nil
	}
	connLock.Unlock()
	agentLost()
}

func agentLost() {
	if !alertAgent {
		alertAgent = true
		cwlog.DoLogCW("Agent connection lost, attempting to reconnect")
		disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, "Lost connection to Factorio agent, attempting to reconnect.")
	}
}

func agentReconnected() {
	if alertAgent {
		alertAgent = false
		cwlog.DoLogCW("Agent connection established")
		disc.SmartWriteDiscord(cfg.Global.Discord.ReportChannel, "Reconnected to Factorio agent.")
	}

	if fact.FactIsRunning {
		return
	}

	if AttachRunningFactorio(context.Background()) {
		return
	}

	if fact.FactAutoStart && !*glob.NoAutoLaunch {
		launchFactorio()
	}
}

func getConn() (net.Conn, error) {
	connLock.Lock()
	if agentConn != nil {
		c := agentConn
		connLock.Unlock()
		return c, nil
	}
	connLock.Unlock()

	c, err := dialFn()
	if err != nil {
		connLock.Lock()
		if !retryingConn {
			retryingConn = true
			agentLost()
			go func() {
				for {
					time.Sleep(time.Second)
					nc, nerr := dialFn()
					if nerr == nil {
						connLock.Lock()
						agentConn = nc
						retryingConn = false
						connLock.Unlock()
						agentReconnected()
						return
					}
				}
			}()
		}
		connLock.Unlock()
		return nil, err
	}
	connLock.Lock()
	agentConn = c
	retryingConn = false
	connLock.Unlock()
	agentReconnected()
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
		markBadConn()
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
	if err != nil {
		markBadConn()
	}
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
	if err != nil {
		markBadConn()
	}
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
		markBadConn()
		return false
	}
	resp := make([]byte, 1)
	if _, err = conn.Read(resp); err != nil {
		markBadConn()
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
	if err != nil {
		markBadConn()
	}
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
		markBadConn()
		return nil, err
	}
	r := bufio.NewReader(conn)
	data, err := r.ReadBytes(0)
	if err != nil {
		markBadConn()
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
		var alerted bool
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
					if !alerted {
						alerted = true
						cwlog.DoLogCW("Agent watch dial error: %v", err)
						cwlog.DoLogCW("Attempting to reconnect")
						agentLost()
					}
					time.Sleep(time.Second)
					continue
				}
				alerted = false
				agentReconnected()
				r = bufio.NewReader(conn)
			}
			b1, err := r.ReadByte()
			if err != nil {
				if !alerted {
					alerted = true
					cwlog.DoLogCW("Agent watch read error: %v", err)
					cwlog.DoLogCW("Attempting to reconnect")
					agentLost()
				}
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
				if !alerted {
					alerted = true
					cwlog.DoLogCW("Agent watch read error: %v", err)
					cwlog.DoLogCW("Attempting to reconnect")
					agentLost()
				}
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			if b2 != agentNotifyBuffered {
				continue
			}
			if _, err := conn.Write([]byte{byte(agentCmdRead)}); err != nil {
				if !alerted {
					alerted = true
					cwlog.DoLogCW("Agent watch write error: %v", err)
					cwlog.DoLogCW("Attempting to reconnect")
					agentLost()
				}
				conn.Close()
				conn = nil
				time.Sleep(time.Second)
				continue
			}
			data, err := r.ReadBytes(0)
			if err != nil {
				if !alerted {
					alerted = true
					cwlog.DoLogCW("Agent watch read error: %v", err)
					cwlog.DoLogCW("Attempting to reconnect")
					agentLost()
				}
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
