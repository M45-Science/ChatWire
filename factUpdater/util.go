package factUpdater

import (
	"ChatWire/cwlog"
	"archive/tar"
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ulikunitz/xz"
)

func readZipFile(zf *zip.File) ([]byte, error) {

	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func parseStringsLatest(entry *getLatestData) error {
	sInt, err := parseVersionString(entry.Stable.Headless)
	if err != nil {
		return err
	}
	xInt, err := parseVersionString(entry.Experimental.Headless)
	if err != nil {
		return err
	}

	entry.Stable.HeadlessInt = sInt
	entry.Experimental.HeadlessInt = xInt
	return nil
}

func parseVersionString(input string) (versionInts, error) {
	if input == "" {
		return versionInts{}, nil
	}

	parts := strings.Split(input, ".")

	numParts := len(parts)
	if numParts != 3 {
		return versionInts{}, fmt.Errorf("malformed version number: incorrect number of parts: %v: %v", numParts, parts)
	}

	a, erra := strconv.ParseUint(parts[0], 10, 64)
	b, errb := strconv.ParseUint(parts[1], 10, 64)
	c, errc := strconv.ParseUint(parts[2], 10, 64)

	if erra != nil || errb != nil || errc != nil {
		return versionInts{}, fmt.Errorf("malformed version number: A part of the version number wasn't a valid positive integer")
	}

	return versionInts{A: int(a), B: int(b), C: int(c)}, nil
}

func (input versionInts) intToString() string {
	return fmt.Sprintf("%v.%v.%v", input.A, input.B, input.C)
}

func isVersionNewerThan(VA, VB versionInts) bool {
	return VA.A >= VB.A && VA.B >= VB.B && VA.C > VB.C
}

func isVersionEqual(VA, VB versionInts) bool {
	return VA.A == VB.A && VA.B == VB.B && VA.C == VB.C
}

func fileExistsSize(filename string) (bool, int64, error) {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, 0, nil
	}
	if err != nil {
		return false, 0, err
	}
	return true, info.Size(), nil
}

func unXZData(data []byte) ([]byte, error) {
	r, err := xz.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}

func untar(dst string, data []byte) error {

	tr := tar.NewReader(bytes.NewReader(data))

	for {
		header, err := tr.Next()

		switch {

		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}
		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {

		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0770)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}

func combinedPackageName(info *infoData) string {

	dlen := len(info.gDistro)
	getBits := info.gDistro
	buf := fmt.Sprintf("core-%v_%v%v", getBits[:dlen-2], info.gBuild, getBits[dlen-2:dlen])
	return buf
}

func attemptThrottle(att int) {
	if att > 0 {
		delay := att * att * 10
		if delay > 0 {
			cwlog.DoLogCW("Will attempt again in %v sec.", int((delay*11)/10.0))
			for i := 0; i < delay*11 && att > 0; i++ {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
	att++
}
