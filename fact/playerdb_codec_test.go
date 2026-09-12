package fact

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"ChatWire/glob"
)

func samplePlayerDatabase(count int) map[string]*glob.PlayerData {
	players := make(map[string]*glob.PlayerData, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("player-%06d", i)
		players[name] = &glob.PlayerData{
			Level:         i%6 - 1,
			ID:            fmt.Sprintf("100000000000%06d", i),
			BanReason:     "repeated test ban reason",
			Creation:      1_000_000 + int64(i),
			LastSeen:      2_000_000 + int64(i),
			Minutes:       int64(i * 17),
			SusScore:      int64(i % 23),
			AlreadyBanned: true,
			SpamScore:     99,
		}
	}
	return players
}

func persistentPlayerData(players map[string]*glob.PlayerData) map[string]*glob.PlayerData {
	result := make(map[string]*glob.PlayerData, len(players))
	for name, player := range players {
		copy := *player
		copy.Name = ""
		copy.AlreadyBanned = false
		copy.SpamScore = 0
		result[name] = &copy
	}
	return result
}

func TestBinaryPlayerDatabaseRoundTripIsDeterministic(t *testing.T) {
	players := samplePlayerDatabase(100)

	first, err := encodePlayerDatabase(players, playerDBFormatBinary)
	if err != nil {
		t.Fatal(err)
	}
	second, err := encodePlayerDatabase(players, " BINARY ")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("binary player database encoding is not deterministic")
	}
	if !bytes.HasPrefix(first, playerDBMagic[:]) {
		t.Fatalf("binary player database lacks magic header: %x", first[:8])
	}

	got, format, err := decodePlayerDatabase(first)
	if err != nil {
		t.Fatal(err)
	}
	if format != playerDBFormatBinary {
		t.Fatalf("detected format %q, want %q", format, playerDBFormatBinary)
	}
	if want := persistentPlayerData(players); !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPlayerDatabaseAutoDetectsLegacyJSON(t *testing.T) {
	want := samplePlayerDatabase(3)
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}

	got, format, err := decodePlayerDatabase(data)
	if err != nil {
		t.Fatal(err)
	}
	if format != playerDBFormatJSON {
		t.Fatalf("detected format %q, want %q", format, playerDBFormatJSON)
	}
	if want = persistentPlayerData(want); !reflect.DeepEqual(got, want) {
		t.Fatalf("legacy JSON mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestJSONPlayerDatabaseRemainsSelectable(t *testing.T) {
	want := samplePlayerDatabase(3)
	data, err := encodePlayerDatabase(want, "json")
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(data) {
		t.Fatalf("configured JSON output is invalid: %q", data)
	}
	got, format, err := decodePlayerDatabase(data)
	if err != nil {
		t.Fatal(err)
	}
	if format != playerDBFormatJSON || !reflect.DeepEqual(got, persistentPlayerData(want)) {
		t.Fatalf("JSON round trip failed: format=%q players=%#v", format, got)
	}
}

func TestBinaryPlayerDatabaseRejectsCorruptionAndUnknownVersion(t *testing.T) {
	data, err := encodePlayerDatabase(samplePlayerDatabase(3), playerDBFormatBinary)
	if err != nil {
		t.Fatal(err)
	}

	corrupt := bytes.Clone(data)
	binary.LittleEndian.PutUint32(corrupt[20:24], binary.LittleEndian.Uint32(corrupt[20:24])+1)
	if _, _, err := decodePlayerDatabase(corrupt); err == nil {
		t.Fatal("checksum corruption was accepted")
	}

	unknownVersion := bytes.Clone(data)
	binary.LittleEndian.PutUint16(unknownVersion[8:10], playerDBBinaryVersion+1)
	if _, _, err := decodePlayerDatabase(unknownVersion); err == nil {
		t.Fatal("unknown binary database version was accepted")
	}
}

func TestPlayerDatabaseRejectsUnknownOutputFormat(t *testing.T) {
	if _, err := encodePlayerDatabase(nil, "yaml"); err == nil {
		t.Fatal("unknown player database output format was accepted")
	}
}

func BenchmarkPlayerDatabaseCodec(b *testing.B) {
	players := samplePlayerDatabase(10_000)
	for _, format := range []string{playerDBFormatBinary, playerDBFormatJSON} {
		b.Run("encode-"+format, func(b *testing.B) {
			for b.Loop() {
				if _, err := encodePlayerDatabase(players, format); err != nil {
					b.Fatal(err)
				}
			}
		})
		data, err := encodePlayerDatabase(players, format)
		if err != nil {
			b.Fatal(err)
		}
		b.Run("decode-"+format, func(b *testing.B) {
			for b.Loop() {
				if _, _, err := decodePlayerDatabase(data); err != nil {
					b.Fatal(err)
				}
			}
			b.ReportMetric(float64(len(data)), "file-B")
		})
	}
}
