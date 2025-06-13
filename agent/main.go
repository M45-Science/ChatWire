package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
)

// FactorioManager manages the Factorio process and exposes a simple HTTP API.
type FactorioManager struct {
	mu    sync.Mutex
	cmd   *exec.Cmd
	stdin io.WriteCloser
	logs  []string
}

func NewFactorioManager() *FactorioManager {
	return &FactorioManager{logs: make([]string, 0, 1024)}
}

// Start launches Factorio with the provided command and arguments.
func (f *FactorioManager) Start(path string, args ...string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cmd != nil {
		return fmt.Errorf("factorio already running")
	}
	cmd := exec.Command(path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	f.cmd = cmd
	f.stdin = stdin
	go f.capture(stdout)
	return nil
}

func (f *FactorioManager) capture(r io.ReadCloser) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			line := string(buf[:n])
			f.mu.Lock()
			f.logs = append(f.logs, line)
			if len(f.logs) > 1000 {
				f.logs = f.logs[len(f.logs)-1000:]
			}
			f.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

// Write sends a command to Factorio's stdin.
func (f *FactorioManager) Write(cmd string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stdin == nil {
		return fmt.Errorf("no running process")
	}
	_, err := io.WriteString(f.stdin, cmd+"\n")
	return err
}

// Stop terminates the running Factorio process.
func (f *FactorioManager) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.cmd == nil {
		return nil
	}
	err := f.cmd.Process.Kill()
	f.cmd.Wait()
	f.cmd = nil
	f.stdin = nil
	return err
}

func (f *FactorioManager) Logs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.logs))
	copy(out, f.logs)
	return out
}

func main() {
	manager := NewFactorioManager()
	binary := flag.String("binary", "factorio", "path to factorio binary")
	var argsStr string
	flag.StringVar(&argsStr, "args", "--start-server-load-latest", "arguments for factorio")
	addr := flag.String("listen", ":8081", "http listen address")
	flag.Parse()

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var argList []string
		if argsStr != "" {
			argList = strings.Split(argsStr, " ")
		}
		if err := manager.Start(*binary, argList...); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := manager.Stop(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	type cmdReq struct {
		Cmd string `json:"cmd"`
	}

	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req cmdReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := manager.Write(req.Cmd); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		logs := manager.Logs()
		w.Header().Set("Content-Type", "text/plain")
		for _, l := range logs {
			fmt.Fprintln(w, l)
		}
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		running := false
		manager.mu.Lock()
		if manager.cmd != nil && manager.cmd.ProcessState == nil {
			running = true
		}
		manager.mu.Unlock()
		type status struct {
			Running bool `json:"running"`
		}
		json.NewEncoder(w).Encode(status{Running: running})
	})

	log.Printf("Listening on %s", *addr)
	http.ListenAndServe(*addr, nil)
}
