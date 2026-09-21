// Package webcontrol defines the private, local-only control protocol.
package webcontrol

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Actor struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Admin bool   `json:"admin"`
}
type Authorization struct {
	UserID  string   `json:"user_id"`
	Name    string   `json:"name"`
	GuildID string   `json:"guild_id"`
	Roles   []string `json:"roles"`
}
type GrantRequest struct {
	UserID        string `json:"user_id"`
	InteractionID string `json:"interaction_id"`
}
type Grant struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Endpoint struct {
	ID             string `json:"id"`
	Label          string `json:"label"`
	Enabled        bool   `json:"enabled"`
	Socket         string `json:"socket"`
	CredentialFile string `json:"credential_file"`
}
type InstanceConfig struct {
	ID                   string `json:"id"`
	Socket               string `json:"socket"`
	CredentialFile       string `json:"credential_file"`
	BrokerSocket         string `json:"broker_socket"`
	BrokerCredentialFile string `json:"broker_credential_file"`
	StateDir             string `json:"state_dir"`
}
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]any{"error": APIError{code, message}})
}
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func Decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return errors.New("invalid JSON request")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}
func Token() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}
func Digest(s string) string { b := sha256.Sum256([]byte(s)); return hex.EncodeToString(b[:]) }
func Equal(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(Digest(a)), []byte(Digest(b))) == 1
}
func Credential(path string) (string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", errors.New("cannot read credential file")
	}
	if !st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0 {
		return "", errors.New("credential must be a regular file with mode 0600")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	s := strings.TrimSpace(string(b))
	if len(s) < 32 {
		return "", errors.New("credential must contain at least 32 characters")
	}
	return s, nil
}
func LoadConfig(path string, v any) error {
	b, e := os.ReadFile(path)
	if e != nil {
		return e
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	if e = d.Decode(v); e != nil {
		return e
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing config data")
	}
	return nil
}
func ValidID(s string) bool {
	if len(s) == 0 || len(s) > 80 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
func Authenticate(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !Equal(r.Header.Get("Authorization"), "Bearer "+secret) {
			Error(w, 401, "unauthorized", "Private credential required.")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func UnixClient(socket string) *http.Client {
	return &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{Timeout: 3 * time.Second}).DialContext(ctx, "unix", socket)
	}}}
}
func Call(ctx context.Context, client *http.Client, secret, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return e
		}
		body = strings.NewReader(string(b))
	}
	r, e := http.NewRequestWithContext(ctx, method, "http://local"+path, body)
	if e != nil {
		return e
	}
	r.Header.Set("Authorization", "Bearer "+secret)
	r.Header.Set("Content-Type", "application/json")
	resp, e := client.Do(r)
	if e != nil {
		return errors.New("local control service unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("local control request failed (%d)", resp.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
	}
	return nil
}

// Listen uses an advisory lock so a restarted process can reclaim its stale
// socket without ever removing a socket still owned by a running process.
func Listen(path string) (net.Listener, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("socket path must be absolute")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	st, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if st.Mode().Perm()&0077 != 0 {
		return nil, errors.New("socket directory must have mode 0700")
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("socket is owned by another process")
	}
	success := false
	defer func() {
		if !success {
			unix.Flock(int(lock.Fd()), unix.LOCK_UN)
			lock.Close()
		}
	}()
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return nil, errors.New("socket path exists and is not a socket")
		}
		conn, err := net.DialTimeout("unix", path, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil, errors.New("socket already has a live listener")
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			return nil, errors.New("cannot verify stale socket ownership")
		}
		if err = os.Remove(path); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err = os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, err
	}
	success = true
	return &lockedListener{Listener: l, lock: lock}, nil
}

type lockedListener struct {
	net.Listener
	lock *os.File
	once sync.Once
	err  error
}

func (l *lockedListener) Close() error {
	l.once.Do(func() { l.err = l.Listener.Close(); _ = unix.Flock(int(l.lock.Fd()), unix.LOCK_UN); _ = l.lock.Close() })
	return l.err
}
