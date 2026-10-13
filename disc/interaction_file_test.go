package disc

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/bwmarrin/discordgo"
)

type interactionFileTransport func(*http.Request) (*http.Response, error)

func (f interactionFileTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func TestInteractionFileResponsePreservesAttachmentAfterAcknowledgement(t *testing.T) {
	for _, acknowledged := range []bool{false, true} {
		name := "initial response"
		if acknowledged {
			name = "edit acknowledged response"
		}
		t.Run(name, func(t *testing.T) {
			oldSession := DS
			t.Cleanup(func() { DS = oldSession })
			session, err := discordgo.New("Bot test-token")
			if err != nil {
				t.Fatal(err)
			}
			DS = session
			payload := strings.Repeat("Orphan Finder: installed; applies on next start.\n", 80)
			requests := 0
			session.Client = &http.Client{Transport: interactionFileTransport(func(r *http.Request) (*http.Response, error) {
				requests++
				if err := r.ParseMultipartForm(1 << 20); err != nil {
					t.Fatal(err)
				}
				defer r.MultipartForm.RemoveAll()
				file, header, err := r.FormFile("files[0]")
				if err != nil {
					t.Fatal(err)
				}
				defer file.Close()
				content, err := io.ReadAll(file)
				if err != nil {
					t.Fatal(err)
				}
				if header.Filename != "edit-mods-results.txt" || string(content) != payload {
					t.Fatalf("request %d attachment %q has %d bytes, want %d", requests, header.Filename, len(content), len(payload))
				}
				response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"id":"message-id"}`)), Request: r}
				if requests == 1 {
					if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/callback") {
						t.Fatalf("unexpected initial request: %s %s", r.Method, r.URL.Path)
					}
					if acknowledged {
						response.StatusCode = http.StatusBadRequest
						response.Body = io.NopCloser(strings.NewReader(`{"code":40060,"message":"Interaction has already been acknowledged."}`))
					} else {
						response.StatusCode = http.StatusNoContent
						response.Body = io.NopCloser(strings.NewReader(""))
					}
				} else if requests != 2 || r.Method != http.MethodPatch || !strings.HasSuffix(r.URL.Path, "/messages/@original") {
					t.Fatalf("unexpected fallback request: %s %s", r.Method, r.URL.Path)
				}
				return response, nil
			})}
			interaction := &discordgo.InteractionCreate{Interaction: &discordgo.Interaction{
				ID: "interaction-id", AppID: "app-id", Token: "interaction-token",
				Member: &discordgo.Member{User: &discordgo.User{Username: "tester"}},
			}}
			InteractionEphemeralFileResponse(interaction, "Edit-Mods", "See attached results.", "edit-mods-results.txt", []byte(payload))
			want := 1
			if acknowledged {
				want = 2
			}
			if requests != want {
				t.Fatalf("requests = %d, want %d", requests, want)
			}
		})
	}
}
