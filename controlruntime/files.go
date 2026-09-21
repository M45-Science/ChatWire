package controlruntime

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/fact"
	"ChatWire/modupdate"
	"ChatWire/support"
	"ChatWire/webcontrol"
	"archive/zip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxUpload int64 = 512 << 20

type Upload struct {
	ID      string    `json:"id"`
	Kind    string    `json:"kind"`
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	Expires time.Time `json:"expires_at"`
	Actor   string    `json:"-"`
	Path    string    `json:"-"`
}

func saveID(name string) string { return base64.RawURLEncoding.EncodeToString([]byte(name)) }
func saveName(id string) (string, error) {
	b, e := base64.RawURLEncoding.DecodeString(id)
	name := string(b)
	if e != nil || name == "" || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00") || !strings.HasSuffix(name, ".zip") {
		return "", errors.New("invalid save ID")
	}
	return name, nil
}
func openSave(name string) (*os.File, error) {
	root, e := os.OpenRoot(cfg.GetSavesFolder())
	if e != nil {
		return nil, e
	}
	defer root.Close()
	f, e := root.Open(name)
	if e != nil {
		return nil, e
	}
	st, e := f.Stat()
	if e != nil || !st.Mode().IsRegular() {
		f.Close()
		return nil, errors.New("invalid save")
	}
	return f, nil
}
func (rt *Runtime) fileRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /internal/v1/saves", func(w http.ResponseWriter, r *http.Request) {
		entries, e := os.ReadDir(cfg.GetSavesFolder())
		if e != nil {
			webcontrol.Error(w, 503, "unavailable", "Cannot read saves.")
			return
		}
		out := []map[string]any{}
		cursor := r.URL.Query().Get("cursor")
		for _, entry := range entries {
			if entry.Name() <= cursor || !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".zip") {
				continue
			}
			st, e := entry.Info()
			if e == nil {
				out = append(out, map[string]any{"id": saveID(entry.Name()), "name": entry.Name(), "size": st.Size(), "modified_at": st.ModTime()})
			}
			if len(out) >= 100 {
				break
			}
		}
		next := ""
		if len(out) == 100 {
			next = out[len(out)-1]["name"].(string)
		}
		webcontrol.JSON(w, 200, map[string]any{"items": out, "next_cursor": next})
	})
	mux.HandleFunc("GET /internal/v1/saves/{save}/download", func(w http.ResponseWriter, r *http.Request) {
		name, e := saveName(r.PathValue("save"))
		if e != nil {
			webcontrol.Error(w, 400, "invalid_save", e.Error())
			return
		}
		f, e := openSave(name)
		if e != nil {
			webcontrol.Error(w, 404, "not_found", "Save not found.")
			return
		}
		defer f.Close()
		st, e := f.Stat()
		if e != nil {
			return
		}
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": name}))
		w.Header().Set("Content-Type", "application/zip")
		http.ServeContent(w, r, name, st.ModTime(), f)
	})
	mux.HandleFunc("GET /internal/v1/map-generators", func(w http.ResponseWriter, r *http.Request) {
		entries, _ := os.ReadDir(cfg.GetSharedMapGeneratorFolder())
		names := []string{"NONE", constants.CustomMapGeneratorName}
		for _, e := range entries {
			if e.IsDir() {
				names = append(names, e.Name())
			} else if strings.HasSuffix(e.Name(), "-gen.json") {
				names = append(names, strings.TrimSuffix(e.Name(), "-gen.json"))
			}
		}
		sort.Strings(names)
		webcontrol.JSON(w, 200, map[string]any{"generators": names, "presets": constants.MapTypes})
	})
	mux.HandleFunc("GET /internal/v1/mods", func(w http.ResponseWriter, r *http.Request) {
		v, e := modupdate.GetModList()
		if e != nil {
			webcontrol.Error(w, 503, "unavailable", "Cannot read mod list.")
			return
		}
		webcontrol.JSON(w, 200, v)
	})
	mux.HandleFunc("GET /internal/v1/mods/history", func(w http.ResponseWriter, r *http.Request) {
		webcontrol.JSON(w, 200, map[string]string{"history": cfg.RedactWebText(modupdate.ListHistory(r.URL.Query().Get("full") == "true"))})
	})
	mux.HandleFunc("GET /internal/v1/logs", rt.logs)
	mux.HandleFunc("POST /internal/v1/uploads", rt.upload)
	mux.HandleFunc("DELETE /internal/v1/uploads/{upload}", func(w http.ResponseWriter, r *http.Request) {
		rt.mu.Lock()
		defer rt.mu.Unlock()
		u, ok := rt.uploads[r.PathValue("upload")]
		if !ok || u.Actor != actor(r).ID {
			webcontrol.Error(w, 404, "not_found", "Upload not found.")
			return
		}
		_ = os.Remove(u.Path)
		delete(rt.uploads, u.ID)
		webcontrol.JSON(w, 200, map[string]bool{"deleted": true})
	})
}
func (rt *Runtime) logs(w http.ResponseWriter, r *http.Request) {
	stream := r.URL.Query().Get("stream")
	dir, prefix := "audit-log", "cw-"
	if stream == "game" {
		dir, prefix = "log", "game-"
	} else if stream == "audit" {
		prefix = "audit-"
	} else if stream != "" && stream != "chatwire" {
		webcontrol.Error(w, 400, "invalid_stream", "Unknown log stream.")
		return
	}
	entries, e := os.ReadDir(dir)
	if e != nil {
		webcontrol.JSON(w, 200, map[string]string{"text": ""})
		return
	}
	var latest os.FileInfo
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		info, e := entry.Info()
		if e == nil && (latest == nil || info.ModTime().After(latest.ModTime())) {
			latest = info
		}
	}
	if latest == nil {
		webcontrol.JSON(w, 200, map[string]string{"text": ""})
		return
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return
	}
	defer root.Close()
	f, e := root.Open(latest.Name())
	if e != nil {
		return
	}
	defer f.Close()
	offset := latest.Size() - 65536
	if offset < 0 {
		offset = 0
	}
	_, _ = f.Seek(offset, io.SeekStart)
	b, _ := io.ReadAll(io.LimitReader(f, 65536))
	webcontrol.JSON(w, 200, map[string]any{"text": cfg.RedactWebText(string(b)), "truncated": offset > 0})
}
func (rt *Runtime) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload+65536)
	reader, e := r.MultipartReader()
	if e != nil {
		webcontrol.Error(w, 400, "invalid_upload", "Multipart file required.")
		return
	}
	part, e := reader.NextPart()
	if e != nil {
		webcontrol.Error(w, 400, "invalid_upload", "File required.")
		return
	}
	defer part.Close()
	kind := part.FormName()
	limit := maxUpload
	if kind == "mod-list" {
		limit = 1 << 20
	} else if kind == "mod-settings" {
		limit = 16 << 20
	} else if kind != "save" {
		webcontrol.Error(w, 422, "invalid_upload", "Field must be save, mod-list, or mod-settings.")
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	now := time.Now()
	for id, u := range rt.uploads {
		if now.After(u.Expires) {
			os.Remove(u.Path)
			delete(rt.uploads, id)
		}
	}
	if len(rt.uploads) >= 20 {
		webcontrol.Error(w, 429, "upload_quota", "Staging quota reached; remove unused uploads.")
		return
	}
	f, e := os.CreateTemp(filepath.Join(rt.Config.StateDir, "uploads"), "upload-")
	if e != nil {
		webcontrol.Error(w, 500, "storage_error", "Unable to stage upload.")
		return
	}
	keep := false
	defer func() {
		f.Close()
		if !keep {
			os.Remove(f.Name())
		}
	}()
	size, e := io.Copy(f, io.LimitReader(part, limit+1))
	if e != nil || size > limit {
		webcontrol.Error(w, 413, "upload_too_large", "Upload exceeds its size limit.")
		return
	}
	if _, e = reader.NextPart(); e != io.EOF {
		webcontrol.Error(w, 400, "invalid_upload", "Upload one file per request.")
		return
	}
	if e = f.Sync(); e != nil {
		webcontrol.Error(w, 500, "storage_error", "Unable to save upload.")
		return
	}
	if e = validateUpload(f.Name(), kind); e != nil {
		webcontrol.Error(w, 422, "invalid_upload", e.Error())
		return
	}
	u := Upload{ID: "upload_" + webcontrol.Token(), Kind: kind, Name: filepath.Base(part.FileName()), Size: size, Expires: now.Add(time.Hour), Actor: actor(r).ID, Path: f.Name()}
	rt.uploads[u.ID] = u
	keep = true
	webcontrol.JSON(w, 201, u)
}
func validateUpload(filename, kind string) error {
	if kind == "save" {
		z, e := zip.OpenReader(filename)
		if e != nil {
			return errors.New("save must be a valid ZIP archive")
		}
		defer z.Close()
		if len(z.File) > 20000 {
			return errors.New("too many archive entries")
		}
		var total uint64
		level := false
		for _, entry := range z.File {
			n := entry.Name
			if strings.Contains(n, "\\") || strings.HasPrefix(n, "/") || path.Clean(n) == ".." || strings.HasPrefix(path.Clean(n), "../") || entry.Mode()&os.ModeSymlink != 0 {
				return errors.New("unsafe archive entry")
			}
			total += entry.UncompressedSize64
			if total > 8<<30 || entry.UncompressedSize64 > 4<<30 {
				return errors.New("archive expands beyond allowed size")
			}
			if strings.HasSuffix(n, "/level.dat") || strings.HasSuffix(n, "/level.dat0") {
				level = true
			}
		}
		if !level {
			return errors.New("archive does not contain a Factorio level")
		}
		if fact.FileHasZipBomb(filename) {
			return errors.New("archive exceeds compression safety limits")
		}
		return nil
	}
	b, e := os.ReadFile(filename)
	if e != nil {
		return errors.New("cannot validate upload")
	}
	if kind == "mod-list" {
		var list modupdate.ModListData
		if json.Unmarshal(b, &list) != nil || len(list.Mods) == 0 || len(list.Mods) > 1000 {
			return errors.New("invalid mod-list.json")
		}
		for _, m := range list.Mods {
			if !validModName(m.Name) {
				return errors.New("invalid mod name")
			}
		}
		return nil
	}
	if len(b) < 8 || binary.LittleEndian.Uint16(b[6:8]) != 0 || binary.LittleEndian.Uint16(b[:2]) == 0 && binary.LittleEndian.Uint16(b[2:4]) < 12 {
		return errors.New("invalid mod-settings.dat header")
	}
	return nil
}
func (rt *Runtime) applyUploads(a webcontrol.Actor, p Params) (any, error) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	uploads := []Upload{}
	kinds := map[string]bool{}
	if len(p.UploadIDs) == 0 || len(p.UploadIDs) > 3 {
		return nil, fail("Supply one to three staged uploads.")
	}
	for _, id := range p.UploadIDs {
		u, ok := rt.uploads[id]
		if !ok || u.Actor != a.ID || time.Now().After(u.Expires) || kinds[u.Kind] {
			return nil, fail("Missing, expired, or duplicate upload.")
		}
		if e := validateUpload(u.Path, u.Kind); e != nil {
			return nil, e
		}
		kinds[u.Kind] = true
		uploads = append(uploads, u)
	}
	fact.SetAutolaunch(false, false)
	if e := fact.SubmitLifecycleRequestAndWait(fact.Request{Kind: fact.ActionStop, Reason: "Applying moderator upload"}); e != nil {
		return nil, fail("Could not stop Factorio; files were not activated.")
	}
	unlock, lockErr := cfg.LockControlResources()
	if lockErr != nil {
		return nil, fail("Shared game files are busy.")
	}
	locked := true
	defer func() {
		if locked {
			unlock()
		}
	}()
	save := ""
	for _, u := range uploads {
		dest := ""
		switch u.Kind {
		case "save":
			save = "upload-" + webcontrol.Token() + ".zip"
			dest = filepath.Join(cfg.GetSavesFolder(), save)
		case "mod-list":
			dest = filepath.Join(cfg.GetModsFolder(), constants.ModListName)
		case "mod-settings":
			dest = filepath.Join(cfg.GetModsFolder(), constants.ModSettingsName)
		}
		// Atomic copy keeps the old target intact if disk writes fail.
		if e := copyAtomic(u.Path, dest); e != nil {
			return nil, fail("File activation failed; Factorio remains stopped. Some earlier files may have been applied.")
		}
	}
	unlock()
	locked = false
	if save != "" {
		if !support.SyncModsService(filepath.Join(cfg.GetSavesFolder(), save)) {
			return nil, fail("Save staged but mod sync failed; Factorio remains stopped.")
		}
	}
	if p.Start {
		kind := fact.ActionStart
		name := ""
		if save != "" {
			kind = fact.ActionChangeMap
			name = strings.TrimSuffix(save, ".zip")
		}
		if e := fact.SubmitLifecycleRequestAndWait(fact.Request{Kind: kind, SaveName: name, Reason: "Loading moderator upload"}); e != nil {
			return nil, fail("Files applied but Factorio failed to start.")
		}
		if err := waitReady(); err != nil {
			return nil, err
		}
		fact.SetAutolaunch(true, false)
	}
	for _, u := range uploads {
		os.Remove(u.Path)
		delete(rt.uploads, u.ID)
	}
	return map[string]any{"applied": true, "started": p.Start, "save_id": saveID(save)}, nil
}
func copyAtomic(src, dst string) error {
	from, e := os.Open(src)
	if e != nil {
		return e
	}
	defer from.Close()
	to, e := os.CreateTemp(filepath.Dir(dst), ".control-copy-")
	if e != nil {
		return e
	}
	name := to.Name()
	defer os.Remove(name)
	_, e = io.Copy(to, from)
	if e == nil {
		e = to.Sync()
	}
	ce := to.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return os.Rename(name, dst)
}
func archiveSave(id string) (any, error) {
	name, e := saveName(id)
	if e != nil {
		return nil, e
	}
	f, e := openSave(name)
	if e != nil {
		return nil, fail("Save not found.")
	}
	f.Close()
	folder := filepath.Join(cfg.Global.Paths.Folders.MapArchives, "web")
	if e = os.MkdirAll(folder, 0755); e != nil {
		return nil, fail("Archive directory unavailable.")
	}
	dst := filepath.Join(folder, time.Now().UTC().Format("20060102T150405")+"-"+name)
	if e = copyAtomic(filepath.Join(cfg.GetSavesFolder(), name), dst); e != nil {
		return nil, fail("Archive failed.")
	}
	return map[string]string{"archive": filepath.Base(dst)}, nil
}
