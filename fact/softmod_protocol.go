package fact

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync/atomic"

	"ChatWire/constants"
	"ChatWire/glob"
)

const SoftModProtocolVersion = 1

var softModRequestSequence atomic.Uint64

type softModRequest struct {
	Version int    `json:"v"`
	ID      string `json:"id"`
	Command string `json:"command"`
	Data    any    `json:"data"`
}

// WriteSoftModCommand sends one typed request through the SoftMod's sole
// machine-only command. The returned ID is echoed by the response.
func WriteSoftModCommand(command string, data any) string {
	id := fmt.Sprintf("cw-%d", softModRequestSequence.Add(1))
	request := softModRequest{
		Version: SoftModProtocolVersion,
		ID:      id,
		Command: command,
		Data:    data,
	}
	payload, err := json.Marshal(request)
	if err != nil {
		return ""
	}
	encoded, err := EncodeSoftModPayload(payload)
	if err != nil {
		return ""
	}
	WriteFact("/chatwire %s", encoded)
	return id
}

// EncodeSoftModPayload matches Factorio helpers.encode_string: DEFLATE wrapped
// by zlib, then standard base64.
func EncodeSoftModPayload(payload []byte) (string, error) {
	var compressed bytes.Buffer
	encoder, err := zlib.NewWriterLevel(&compressed, zlib.BestSpeed)
	if err != nil {
		return "", err
	}
	if _, err = encoder.Write(payload); err != nil {
		_ = encoder.Close()
		return "", err
	}
	if err = encoder.Close(); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(compressed.Bytes()), nil
}

func DecodeSoftModPayload(encoded string) ([]byte, error) {
	compressed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	reader, err := zlib.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	const maxPayload = 1 << 20
	payload, err := io.ReadAll(io.LimitReader(reader, maxPayload+1))
	if err != nil {
		return nil, err
	}
	if len(payload) > maxPayload {
		return nil, fmt.Errorf("SoftMod payload exceeds %d bytes", maxPayload)
	}
	return payload, nil
}

func WriteSoftModSpeed(speed float32) string {
	return WriteSoftModCommand("config", map[string]any{"speed": speed})
}

func RequestOnlinePlayers() {
	if glob.SoftModVersion == constants.Unknown {
		WriteFact(glob.OnlineCommand)
		return
	}
	WriteSoftModCommand("online", nil)
}
