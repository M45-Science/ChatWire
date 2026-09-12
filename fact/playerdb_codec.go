package fact

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"sort"
	"strings"
	"sync"

	"ChatWire/glob"

	"github.com/bytedance/sonic"
	"github.com/klauspost/compress/zstd"
)

const (
	playerDBFormatBinary = "binary"
	playerDBFormatJSON   = "json"

	playerDBBinaryVersion = uint16(1)
	playerDBCodecZstd     = byte(1)
	playerDBHeaderSize    = 24

	maxPlayerDBPayload = 512 << 20
	maxPlayerDBRecords = 5_000_000
	maxPlayerDBString  = 16 << 20
)

var (
	playerDBMagic     = [8]byte{'C', 'W', 'P', 'L', 'A', 'Y', 'E', 'R'}
	playerDBCRC       = crc32.MakeTable(crc32.Castagnoli)
	playerDBCodecs    sync.Once
	playerDBEncoder   *zstd.Encoder
	playerDBDecoder   *zstd.Decoder
	playerDBCodecInit error
)

// Binary v1 uses this little-endian envelope:
//
//	[8] magic "CWPLAYER"
//	[2] version
//	[1] compression codec (1 = Zstandard)
//	[1] reserved flags
//	[8] uncompressed payload size
//	[4] CRC32C of the uncompressed payload
//	[ ] compressed payload
//
// The payload starts with a record count. Each key-sorted record stores the
// player name, level, Discord ID, ban reason, creation/seen times, minutes, and
// suspicion score using length-prefixed strings and signed varints.
func initPlayerDBCodecs() error {
	playerDBCodecs.Do(func() {
		playerDBEncoder, playerDBCodecInit = zstd.NewWriter(nil,
			zstd.WithEncoderLevel(zstd.SpeedFastest),
			zstd.WithEncoderConcurrency(1),
			zstd.WithEncoderCRC(false),
		)
		if playerDBCodecInit != nil {
			return
		}
		playerDBDecoder, playerDBCodecInit = zstd.NewReader(nil,
			zstd.WithDecoderConcurrency(1),
			zstd.WithDecoderMaxMemory(maxPlayerDBPayload),
		)
	})
	return playerDBCodecInit
}

func normalizePlayerDBFormat(format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", playerDBFormatBinary:
		return playerDBFormatBinary, nil
	case playerDBFormatJSON:
		return playerDBFormatJSON, nil
	default:
		return "", fmt.Errorf("unsupported player database format %q", format)
	}
}

func encodePlayerDatabase(players map[string]*glob.PlayerData, format string) ([]byte, error) {
	format, err := normalizePlayerDBFormat(format)
	if err != nil {
		return nil, err
	}
	if format == playerDBFormatJSON {
		return sonic.MarshalIndent(players, "", "\t")
	}
	return encodeBinaryPlayerDatabase(players)
}

func decodePlayerDatabase(data []byte) (map[string]*glob.PlayerData, string, error) {
	if len(data) == 0 {
		return make(map[string]*glob.PlayerData), "", nil
	}
	if len(data) >= len(playerDBMagic) && bytes.Equal(data[:len(playerDBMagic)], playerDBMagic[:]) {
		players, err := decodeBinaryPlayerDatabase(data)
		return players, playerDBFormatBinary, err
	}

	players := make(map[string]*glob.PlayerData)
	if err := sonic.Unmarshal(data, &players); err != nil {
		return nil, playerDBFormatJSON, fmt.Errorf("decode JSON player database: %w", err)
	}
	for name, player := range players {
		if player == nil {
			return nil, playerDBFormatJSON, fmt.Errorf("decode JSON player database: player %q is null", name)
		}
	}
	return players, playerDBFormatJSON, nil
}

func encodeBinaryPlayerDatabase(players map[string]*glob.PlayerData) ([]byte, error) {
	if len(players) > maxPlayerDBRecords {
		return nil, fmt.Errorf("player database contains %d records; maximum is %d", len(players), maxPlayerDBRecords)
	}

	keys := make([]string, 0, len(players))
	for name, player := range players {
		if player == nil {
			return nil, fmt.Errorf("player %q is null", name)
		}
		keys = append(keys, name)
	}
	sort.Strings(keys)

	payload := make([]byte, 0, len(players)*32)
	payload = binary.AppendUvarint(payload, uint64(len(keys)))
	for _, name := range keys {
		player := players[name]
		var err error
		payload, err = appendPlayerDBString(payload, name)
		if err != nil {
			return nil, fmt.Errorf("encode player %q: %w", name, err)
		}
		payload = binary.AppendVarint(payload, int64(player.Level))
		payload, err = appendPlayerDBString(payload, player.ID)
		if err != nil {
			return nil, fmt.Errorf("encode player %q ID: %w", name, err)
		}
		payload, err = appendPlayerDBString(payload, player.BanReason)
		if err != nil {
			return nil, fmt.Errorf("encode player %q ban reason: %w", name, err)
		}
		payload = binary.AppendVarint(payload, player.Creation)
		payload = binary.AppendVarint(payload, player.LastSeen)
		payload = binary.AppendVarint(payload, player.Minutes)
		payload = binary.AppendVarint(payload, player.SusScore)
		if len(payload) > maxPlayerDBPayload {
			return nil, fmt.Errorf("player database payload exceeds %d bytes", maxPlayerDBPayload)
		}
	}

	if err := initPlayerDBCodecs(); err != nil {
		return nil, fmt.Errorf("initialize player database compression: %w", err)
	}
	compressed := playerDBEncoder.EncodeAll(payload, nil)
	result := make([]byte, playerDBHeaderSize, playerDBHeaderSize+len(compressed))
	copy(result, playerDBMagic[:])
	binary.LittleEndian.PutUint16(result[8:10], playerDBBinaryVersion)
	result[10] = playerDBCodecZstd
	result[11] = 0
	binary.LittleEndian.PutUint64(result[12:20], uint64(len(payload)))
	binary.LittleEndian.PutUint32(result[20:24], crc32.Checksum(payload, playerDBCRC))
	return append(result, compressed...), nil
}

func decodeBinaryPlayerDatabase(data []byte) (map[string]*glob.PlayerData, error) {
	if len(data) < playerDBHeaderSize {
		return nil, errors.New("binary player database header is truncated")
	}
	if binary.LittleEndian.Uint16(data[8:10]) != playerDBBinaryVersion {
		return nil, fmt.Errorf("unsupported binary player database version %d", binary.LittleEndian.Uint16(data[8:10]))
	}
	if data[10] != playerDBCodecZstd {
		return nil, fmt.Errorf("unsupported binary player database codec %d", data[10])
	}
	if data[11] != 0 {
		return nil, fmt.Errorf("unsupported binary player database flags %#x", data[11])
	}

	expectedSize := binary.LittleEndian.Uint64(data[12:20])
	if expectedSize > maxPlayerDBPayload {
		return nil, fmt.Errorf("binary player database payload is %d bytes; maximum is %d", expectedSize, maxPlayerDBPayload)
	}
	if err := initPlayerDBCodecs(); err != nil {
		return nil, fmt.Errorf("initialize player database compression: %w", err)
	}
	payload, err := playerDBDecoder.DecodeAll(data[playerDBHeaderSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decompress binary player database: %w", err)
	}
	if uint64(len(payload)) != expectedSize {
		return nil, fmt.Errorf("binary player database size mismatch: got %d, want %d", len(payload), expectedSize)
	}
	if got, want := crc32.Checksum(payload, playerDBCRC), binary.LittleEndian.Uint32(data[20:24]); got != want {
		return nil, fmt.Errorf("binary player database checksum mismatch: got %08x, want %08x", got, want)
	}
	return decodePlayerDBPayload(payload)
}

func appendPlayerDBString(dst []byte, value string) ([]byte, error) {
	if len(value) > maxPlayerDBString {
		return nil, fmt.Errorf("string is %d bytes; maximum is %d", len(value), maxPlayerDBString)
	}
	dst = binary.AppendUvarint(dst, uint64(len(value)))
	return append(dst, value...), nil
}

type playerDBReader struct {
	data []byte
	off  int
}

func (r *playerDBReader) uvarint() (uint64, error) {
	value, size := binary.Uvarint(r.data[r.off:])
	if size == 0 {
		return 0, errors.New("unexpected end of payload")
	}
	if size < 0 {
		return 0, errors.New("integer overflow")
	}
	r.off += size
	return value, nil
}

func (r *playerDBReader) varint() (int64, error) {
	value, size := binary.Varint(r.data[r.off:])
	if size == 0 {
		return 0, errors.New("unexpected end of payload")
	}
	if size < 0 {
		return 0, errors.New("integer overflow")
	}
	r.off += size
	return value, nil
}

func (r *playerDBReader) string() (string, error) {
	size, err := r.uvarint()
	if err != nil {
		return "", err
	}
	if size > maxPlayerDBString {
		return "", fmt.Errorf("string is %d bytes; maximum is %d", size, maxPlayerDBString)
	}
	if size > uint64(len(r.data)-r.off) {
		return "", errors.New("string extends past end of payload")
	}
	value := string(r.data[r.off : r.off+int(size)])
	r.off += int(size)
	return value, nil
}

func decodePlayerDBPayload(payload []byte) (map[string]*glob.PlayerData, error) {
	reader := playerDBReader{data: payload}
	count, err := reader.uvarint()
	if err != nil {
		return nil, fmt.Errorf("decode player count: %w", err)
	}
	if count > maxPlayerDBRecords {
		return nil, fmt.Errorf("player database contains %d records; maximum is %d", count, maxPlayerDBRecords)
	}
	if count > uint64(math.MaxInt) {
		return nil, fmt.Errorf("player count %d exceeds platform capacity", count)
	}
	// Even the smallest valid record needs one name byte, three string lengths,
	// one level, and four persistent integer fields. Check this before sizing the
	// map so a tiny corrupt payload cannot request a huge allocation.
	const minimumRecordSize = 9
	if count > uint64((len(payload)-reader.off)/minimumRecordSize) {
		return nil, fmt.Errorf("player count %d cannot fit in %d payload bytes", count, len(payload)-reader.off)
	}

	players := make(map[string]*glob.PlayerData, int(count))
	for i := uint64(0); i < count; i++ {
		name, err := reader.string()
		if err != nil {
			return nil, fmt.Errorf("decode player %d name: %w", i, err)
		}
		if name == "" {
			return nil, fmt.Errorf("decode player %d: name is empty", i)
		}
		if _, exists := players[name]; exists {
			return nil, fmt.Errorf("decode player %d: duplicate name %q", i, name)
		}
		level, err := reader.varint()
		if err != nil || int64(int(level)) != level {
			if err == nil {
				err = errors.New("level exceeds platform integer range")
			}
			return nil, fmt.Errorf("decode player %q level: %w", name, err)
		}
		id, err := reader.string()
		if err != nil {
			return nil, fmt.Errorf("decode player %q ID: %w", name, err)
		}
		banReason, err := reader.string()
		if err != nil {
			return nil, fmt.Errorf("decode player %q ban reason: %w", name, err)
		}
		creation, err := reader.varint()
		if err != nil {
			return nil, fmt.Errorf("decode player %q creation: %w", name, err)
		}
		lastSeen, err := reader.varint()
		if err != nil {
			return nil, fmt.Errorf("decode player %q last seen: %w", name, err)
		}
		minutes, err := reader.varint()
		if err != nil {
			return nil, fmt.Errorf("decode player %q minutes: %w", name, err)
		}
		susScore, err := reader.varint()
		if err != nil {
			return nil, fmt.Errorf("decode player %q suspicious score: %w", name, err)
		}

		players[name] = &glob.PlayerData{
			Level:     int(level),
			ID:        id,
			BanReason: banReason,
			Creation:  creation,
			LastSeen:  lastSeen,
			Minutes:   minutes,
			SusScore:  susScore,
		}
	}
	if reader.off != len(payload) {
		return nil, fmt.Errorf("binary player database has %d trailing payload bytes", len(payload)-reader.off)
	}
	return players, nil
}
