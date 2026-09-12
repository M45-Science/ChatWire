package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const fileLockRetryInterval = 50 * time.Millisecond

type FileLockMetadata struct {
	PID        int       `json:"pid"`
	Owner      string    `json:"owner"`
	AcquiredAt time.Time `json:"acquired_at"`
}

type FileLock struct {
	file *os.File
}

// AcquireFileLock acquires a kernel-managed advisory lock. The lock file is
// intentionally persistent; the kernel lock itself is released when the file
// descriptor is closed or the process exits.
func AcquireFileLock(path, owner string, timeout time.Duration) (*FileLock, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	deadline := time.Now().Add(timeout)
	for {
		err = unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) && !errors.Is(err, unix.EAGAIN) {
			_ = file.Close()
			return nil, err
		}
		if timeout <= 0 || time.Now().After(deadline) {
			holder := readFileLockMetadata(path)
			_ = file.Close()
			if holder == "" {
				holder = "unknown holder"
			}
			return nil, fmt.Errorf("timed out acquiring lock %q (%s)", path, holder)
		}
		time.Sleep(fileLockRetryInterval)
	}

	metadata := FileLockMetadata{
		PID:        os.Getpid(),
		Owner:      owner,
		AcquiredAt: time.Now().UTC(),
	}
	data, err := json.Marshal(metadata)
	if err == nil {
		err = file.Truncate(0)
	}
	if err == nil {
		_, err = file.Seek(0, 0)
	}
	if err == nil {
		_, err = file.Write(append(data, '\n'))
	}
	if err == nil {
		err = file.Sync()
	}
	if err != nil {
		_ = unix.Flock(int(file.Fd()), unix.LOCK_UN)
		_ = file.Close()
		return nil, fmt.Errorf("write lock metadata: %w", err)
	}

	return &FileLock{file: file}, nil
}

func readFileLockMetadata(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func (lock *FileLock) Release() error {
	if lock == nil || lock.file == nil {
		return nil
	}

	unlockErr := unix.Flock(int(lock.file.Fd()), unix.LOCK_UN)
	closeErr := lock.file.Close()
	lock.file = nil
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
