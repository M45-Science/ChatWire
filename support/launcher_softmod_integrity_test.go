package support

import (
	"ChatWire/cfg"
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInjectionPreservesOriginalOnFailure(t *testing.T) {
	for _, mode := range []string{"missing", "empty", "corrupt"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			old := cfg.Local.Options.SoftModOptions.SoftModPath
			t.Cleanup(func() { cfg.Local.Options.SoftModOptions.SoftModPath = old })
			payload := filepath.Join(dir, "payload")
			cfg.Local.Options.SoftModOptions.SoftModPath = payload
			if mode != "missing" {
				if err := os.Mkdir(payload, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if mode == "corrupt" {
				if err := os.WriteFile(filepath.Join(payload, "control.lua"), []byte("return"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			var b bytes.Buffer
			z := zip.NewWriter(&b)
			for _, name := range []string{"world/level.dat0", "world/control.lua"} {
				w, err := z.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
				if err != nil {
					t.Fatal(err)
				}
				if _, err = w.Write([]byte("original-data")); err != nil {
					t.Fatal(err)
				}
			}
			if err := z.Close(); err != nil {
				t.Fatal(err)
			}
			original := b.Bytes()
			if mode == "corrupt" {
				original = bytes.Replace(original, []byte("original-data"), []byte("corrupt--data"), 1)
			}
			save := filepath.Join(dir, "save.zip")
			if err := os.WriteFile(save, original, 0600); err != nil {
				t.Fatal(err)
			}
			if err := injectSoftMod(save, "world"); err == nil {
				t.Fatal("accepted failed injection")
			}
			after, err := os.ReadFile(save)
			if err != nil || !bytes.Equal(after, original) {
				t.Fatal("original save changed", err)
			}
		})
	}
}

func TestInjectionSuccess(t *testing.T) {
	dir := t.TempDir()
	old := cfg.Local.Options.SoftModOptions.SoftModPath
	t.Cleanup(func() { cfg.Local.Options.SoftModOptions.SoftModPath = old })
	cfg.Local.Options.SoftModOptions.SoftModPath = filepath.Join(dir, "payload")
	if err := os.Mkdir(cfg.Local.Options.SoftModOptions.SoftModPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.Local.Options.SoftModOptions.SoftModPath, "control.lua"), []byte("new-control"), 0600); err != nil {
		t.Fatal(err)
	}
	save := filepath.Join(dir, "save.zip")
	f, err := os.Create(save)
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, name := range []string{"world/level.dat0", "world/control.lua"} {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = w.Write([]byte("original")); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := injectSoftMod(save, "world"); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(save)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if len(archive.File) != 2 {
		t.Fatal("unexpected entries", len(archive.File))
	}
}
