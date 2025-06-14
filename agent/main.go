package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Agent command bytes. Notifications reuse cmdStop followed by notifyBuffered.
const (
	cmdStart agentCmd = iota + 1
	cmdStop
	cmdRunning
	cmdWrite
	cmdRead
)

const respOK byte = byte(cmdStart)

const notifyBuffered byte = 0x06

type agentCmd byte

var (
	procCmd   *exec.Cmd
	procIn    io.WriteCloser
	bufLock   sync.Mutex
	outBuf    []string
	connLock  sync.Mutex
	conns     = make(map[net.Conn]struct{})
	debug     = flag.Bool("debug", false, "enable debug logging")
	commandFn = exec.Command
)

func main() {
	flag.Parse()
	if !*debug {
		log.SetOutput(io.Discard)
	}
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	socketPath := "../factorio-agent.sock"
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Printf("agent listening on %s", socketPath)
	go notifier()
	for {
		c, err := ln.Accept()
		if err != nil {
			log.Printf("accept error: %v", err)
			continue
		}
		log.Printf("connection from %v", c.RemoteAddr())
		connLock.Lock()
		conns[c] = struct{}{}
		connLock.Unlock()
		go handleConn(c)
	}
}

func notifier() {
	t := time.NewTicker(time.Second / 2)
	for range t.C {
		bufLock.Lock()
		has := len(outBuf) > 0
		bufLock.Unlock()
		if has {
			connLock.Lock()
			for c := range conns {
				c.Write([]byte{byte(cmdStop), notifyBuffered})
			}
			connLock.Unlock()
			log.Printf("sent buffered notification to %d connections", len(conns))
		}
	}
}

func handleConn(c net.Conn) {
	defer func() {
		connLock.Lock()
		delete(conns, c)
		connLock.Unlock()
		c.Close()
		log.Printf("connection from %v closed", c.RemoteAddr())
	}()
	r := bufio.NewReader(c)
	for {
		cmd, err := r.ReadByte()
		if err != nil {
			log.Printf("read error from %v: %v", c.RemoteAddr(), err)
			return
		}
		log.Printf("command %02x from %v", cmd, c.RemoteAddr())
		switch agentCmd(cmd) {
		case cmdStart: // start
			line, _ := r.ReadString('\n')
			args := strings.Fields(strings.TrimSpace(line))
			log.Printf("starting Factorio with args: %v", args)
			if err := startFactorio(args); err != nil {
				log.Printf("start error: %v", err)
			}
			c.Write([]byte{respOK})
		case cmdStop: // stop
			log.Printf("stop command received")
			stopFactorio()
			c.Write([]byte{respOK})
		case cmdRunning: // running?
			b := byte(0)
			if procCmd != nil && procCmd.Process != nil {
				b = 1
			}
			c.Write([]byte{b})
		case cmdWrite: // write
			line, _ := r.ReadString('\n')
			writeFactorio(strings.TrimRight(line, "\n"))
			log.Printf("wrote line: %s", strings.TrimSpace(line))
			c.Write([]byte{respOK})
		case cmdRead: // read buffer
			lines := readBuffer()
			var buf bytes.Buffer
			for _, l := range lines {
				buf.WriteString(l)
				buf.WriteByte('\n')
			}
			buf.WriteByte(0)
			c.Write(buf.Bytes())
		default:
			log.Printf("unknown command %02x", cmd)
		}
	}
}

func startFactorio(args []string) error {
	if procCmd != nil {
		return errors.New("running")
	}
	if len(args) == 0 {
		return errors.New("no binary path provided")
	}
	bin := args[0]
	if !filepath.IsAbs(bin) {
		bin = filepath.Join("..", bin)
	}
	log.Printf("launching Factorio: %s %v", bin, args[1:])
	procCmd = commandFn(bin, args[1:]...)
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
		log.Printf("start error: %v", err)
		return err
	}
	go readStdout(stdout)
	go func() {
		procCmd.Wait()
		log.Printf("factorio exited")
		procCmd = nil
	}()
	return nil
}

func readStdout(r io.Reader) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("stdout: %s", line)
		bufLock.Lock()
		outBuf = append(outBuf, line)
		bufLock.Unlock()
	}
}

func stopFactorio() {
	if procCmd == nil {
		return
	}
	log.Printf("stopping factorio")
	writeFactorio("/quit")
	if err := procCmd.Wait(); err != nil {
		log.Printf("wait error: %v", err)
	}
	log.Printf("factorio stopped")
	procCmd = nil
}

func writeFactorio(line string) {
	if procIn == nil {
		return
	}
	if _, err := io.WriteString(procIn, line+"\n"); err != nil {
		log.Printf("stdin write error: %v", err)
		return
	}
	log.Printf("sent to stdin: %s", line)
}

func readBuffer() []string {
	bufLock.Lock()
	defer bufLock.Unlock()
	lines := append([]string(nil), outBuf...)
	outBuf = outBuf[:0]
	return lines
}
