package support

import (
	"bufio"
	"net"
	"strings"
	"sync"
)

const agentSocket = "/var/run/factorio-agent.sock"

var (
	agentConn net.Conn
	connLock  sync.Mutex
	dialFn    = func() (net.Conn, error) { return net.Dial("unix", agentSocket) }
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

func AgentStart(args []string) error {
	conn, err := getConn()
	if err != nil {
		return err
	}
	buf := []byte{0x01}
	if len(args) > 0 {
		buf = append(buf, []byte(strings.Join(args, " ")+"\n")...)
	} else {
		buf = append(buf, '\n')
	}
	_, err = conn.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

func AgentStop() error {
	conn, err := getConn()
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte{0x02})
	return err
}

func AgentRunning() bool {
	conn, err := getConn()
	if err != nil {
		return false
	}
	if _, err = conn.Write([]byte{0x03}); err != nil {
		return false
	}
	resp := make([]byte, 1)
	if _, err = conn.Read(resp); err != nil {
		return false
	}
	return resp[0] == 1
}

func AgentWrite(line string) error {
	conn, err := getConn()
	if err != nil {
		return err
	}
	_, err = conn.Write(append([]byte{0x04}, []byte(line+"\n")...))
	return err
}

func AgentReadBuffered() ([]string, error) {
	conn, err := getConn()
	if err != nil {
		return nil, err
	}
	if _, err = conn.Write([]byte{0x05}); err != nil {
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
