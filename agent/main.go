package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"ChatWire/fact"
)

const socketPath = "/var/run/factorio-agent.sock"

var (
	procCmd  *exec.Cmd
	procIn   io.WriteCloser
	bufLock  sync.Mutex
	outBuf   []string
	connLock sync.Mutex
	conns    = make(map[net.Conn]struct{})
)

func main() {
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}
	go notifier()
	for {
		c, err := ln.Accept()
		if err != nil {
			continue
		}
		connLock.Lock()
		conns[c] = struct{}{}
		connLock.Unlock()
		go handleConn(c)
	}
}

func notifier() {
	t := time.NewTicker(time.Second)
	for range t.C {
		bufLock.Lock()
		has := len(outBuf) > 0
		bufLock.Unlock()
		if has {
			connLock.Lock()
			for c := range conns {
				c.Write([]byte{0x02, 0x06})
			}
			connLock.Unlock()
		}
	}
}

func handleConn(c net.Conn) {
	defer func() {
		connLock.Lock()
		delete(conns, c)
		connLock.Unlock()
		c.Close()
	}()
	r := bufio.NewReader(c)
	for {
		cmd, err := r.ReadByte()
		if err != nil {
			return
		}
		switch cmd {
		case 0x01: // start
			line, _ := r.ReadString('\n')
			args := strings.Fields(strings.TrimSpace(line))
			startFactorio(args)
			c.Write([]byte{0x01})
		case 0x02: // stop
			stopFactorio()
			c.Write([]byte{0x01})
		case 0x03: // running?
			b := byte(0)
			if procCmd != nil && procCmd.Process != nil {
				b = 1
			}
			c.Write([]byte{b})
		case 0x04: // write
			line, _ := r.ReadString('\n')
			writeFactorio(strings.TrimRight(line, "\n"))
			c.Write([]byte{0x01})
		case 0x05: // read buffer
			lines := readBuffer()
			var buf bytes.Buffer
			for _, l := range lines {
				buf.WriteString(l)
				buf.WriteByte('\n')
			}
			buf.WriteByte(0)
			c.Write(buf.Bytes())
		default:
		}
	}
}

func startFactorio(args []string) error {
	if procCmd != nil {
		return errors.New("running")
	}
	procCmd = exec.Command(fact.GetFactorioBinary(), args...)
	procCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdout, err := procCmd.StdoutPipe()
	if err != nil {
		return err
	}
	procIn, err = procCmd.StdinPipe()
	if err != nil {
		return err
	}
	procCmd.Stderr = procCmd.Stdout
	if err := procCmd.Start(); err != nil {
		return err
	}
	go readStdout(stdout)
	go func() {
		procCmd.Wait()
		procCmd = nil
	}()
	return nil
}

func readStdout(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		bufLock.Lock()
		outBuf = append(outBuf, line)
		bufLock.Unlock()
	}
}

func stopFactorio() {
	if procCmd == nil {
		return
	}
	writeFactorio("/quit")
	procCmd.Wait()
	procCmd = nil
}

func writeFactorio(line string) {
	if procIn != nil {
		io.WriteString(procIn, line+"\n")
	}
}

func readBuffer() []string {
	bufLock.Lock()
	defer bufLock.Unlock()
	lines := append([]string(nil), outBuf...)
	outBuf = outBuf[:0]
	return lines
}
