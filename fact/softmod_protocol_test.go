package fact

import "testing"

func TestWriteSoftModCommandProducesJSONEnvelope(t *testing.T) {
	pipe := installPlayerLevelCapturePipe(t)
	id := WriteSoftModCommand("chat", map[string]any{"text": "quote: \" and newline\n"})
	if id == "" {
		t.Fatal("request ID is empty")
	}
	request := decodeSoftModTestRequest(t, pipe.String())
	if request.Version != SoftModProtocolVersion || request.ID != id || request.Command != "chat" {
		t.Fatalf("unexpected envelope: %#v", request)
	}
	data := request.Data.(map[string]any)
	if got := data["text"]; got != "quote: \" and newline\n" {
		t.Fatalf("text = %#v", got)
	}
}
