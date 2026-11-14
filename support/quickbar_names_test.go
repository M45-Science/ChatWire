package support

import (
	"os"
	"strings"
	"testing"

	"ChatWire/fact"
)

// These tests exercise the existing data boundaries without imposing a new
// general naming policy. The only new input restriction is rich-text markup
// in user-visible preset labels; player identity names remain unchanged.
func TestQuickbarNamesRemainDataAcrossStorageAndMachineCommands(t *testing.T) {
	path, w, exchange := quickbarBatchFixture(t)
	for _, label := range []string{
		"[Rail]", "[.NET]", "Café 火箭 👨‍👩‍👧", "hidden\u200bjoiner", "non\u00a0breaking space",
		`"; print('test'); --`, `<script>test</script>`, `100% "quoted" \ path`, "line\nbreak", "nul\x00name",
	} {
		owner := `../"Alice%\\`
		r := quickbarReply{Labels: make([]string, 4)}
		if err := readWriteQuickbar(path, quickbarRequest{Name: owner, Action: "store", Slot: 2, Label: &label, Exchange: exchange}, &r); err != nil || r.Error != "" || r.Labels[1] != label {
			t.Fatalf("name changed or escaped its storage field: %q %+v %v", label, r, err)
		}
		w.Reset()
		r.Name = owner
		fact.WriteSoftModCommand("quickbar-result", r)
		command, data := capturedChatWireRequest(t, w.String())
		if strings.Count(w.String(), "\n") != 1 || command != "quickbar-result" || data["name"] != owner || data["labels"].([]any)[1] != label {
			t.Fatal("name escaped its encoded JSON field", data)
		}
		db, err := readQuickbarDatabase(path)
		if err != nil || len(db) != 1 || db[strings.ToLower(owner)][1].Exchange != exchange {
			t.Fatal("name changed its owner or toolbar data", db, err)
		}
	}
	entries, err := os.ReadDir(strings.TrimSuffix(path, "/quickbar-presets.json"))
	if err != nil || len(entries) != 2 {
		t.Fatal("expected only the fixed JSON file and its lock; names must not create paths", entries, err)
	}
}

func TestQuickbarExistingLabelLengthLimitPreventsWrites(t *testing.T) {
	path, _, exchange := quickbarBatchFixture(t)
	label := strings.Repeat("火", 33)
	r := quickbarReply{Labels: make([]string, 4)}
	if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "store", Slot: 2, Label: &label, Exchange: exchange}, &r); err != nil || r.Error != "Preset names can have up to 32 characters." {
		t.Fatal("existing length limit lost", r, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("overlong label wrote a file")
	}
}

func TestQuickbarRejectsOnlyMarkupInPresetLabels(t *testing.T) {
	path, _, exchange := quickbarBatchFixture(t)
	for _, label := range []string{"[img=utility/close]", "[font=x]x[/font]", "[color=red]x[.color]", "[gps=1,2]", "[item=iron-plate]", "[/font]", "[.color]", "[img=unfinished", "[[img=utility/close]"} {
		for _, action := range []string{"store", "rename"} {
			r := quickbarReply{Labels: make([]string, 4)}
			if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: action, Slot: 2, Label: &label, Exchange: exchange}, &r); err != nil || r.Error != "Preset names cannot contain rich-text tags." {
				t.Fatal("markup label accepted", label, r, err)
			}
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("rejected label touched preset JSON")
	}
}

func TestQuickbarLegacyMarkupOnlyChangesDisplay(t *testing.T) {
	path, _, exchange := quickbarBatchFixture(t)
	original := []byte(`{"alice":[{}, {"label":"[img=utility/close]","exchange":"` + exchange + `"}, {"label":"[Rail]"}]}`)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	r := quickbarReply{Labels: make([]string, 4)}
	if err := readWriteQuickbar(path, quickbarRequest{Name: "Alice", Action: "load", Slot: 2}, &r); err != nil || r.Labels[1] != "" || r.Labels[2] != "[Rail]" || r.Exchange != exchange {
		t.Fatal(r, err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("display filtering changed saved data", err)
	}
}
