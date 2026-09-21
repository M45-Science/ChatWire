package cfg

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"ChatWire/constants"
	"ChatWire/util"
)

type WebField struct {
	Label    string   `json:"label"`
	Key      string   `json:"key"`
	Type     string   `json:"type"`
	Admin    bool     `json:"admin"`
	Secret   bool     `json:"secret"`
	ReadOnly bool     `json:"read_only"`
	Effect   string   `json:"effect"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Choices  []string `json:"choices,omitempty"`
}
type WebSettings struct {
	Revision    string         `json:"revision"`
	Values      map[string]any `json:"values"`
	Application string         `json:"application"`
}

var configFileMu sync.Mutex
var configBaselines = map[string]map[string]any{}

func configPath(global bool) string {
	if global {
		return constants.CWGlobalConfig
	}
	return constants.CWLocalConfig
}
func flatten(v map[string]any, prefix string, out map[string]any) {
	for k, x := range v {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		if m, ok := x.(map[string]any); ok {
			flatten(m, key, out)
		} else {
			out[key] = x
		}
	}
}
func unflatten(values map[string]any) map[string]any {
	out := map[string]any{}
	for key, v := range values {
		parts := strings.Split(key, ".")
		m := out
		for _, p := range parts[:len(parts)-1] {
			if _, ok := m[p]; !ok {
				m[p] = map[string]any{}
			}
			m = m[p].(map[string]any)
		}
		m[parts[len(parts)-1]] = v
	}
	return out
}
func jsonValues(v any) (map[string]any, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	var raw map[string]any
	if e = json.Unmarshal(b, &raw); e != nil {
		return nil, e
	}
	out := map[string]any{}
	flatten(raw, "", out)
	return out, nil
}
func readConfigValues(path string) (map[string]any, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var raw map[string]any
	if e = json.Unmarshal(b, &raw); e != nil {
		return nil, e
	}
	out := map[string]any{}
	flatten(raw, "", out)
	return out, nil
}
func revision(v map[string]any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// rememberConfig captures the runtime baseline for three-way disk writes. A web
// edit stays on disk until explicitly reloaded and cannot be erased by an
// unrelated write from a process that still has the old value in memory.
func rememberConfig(path string, v any) {
	values, e := jsonValues(v)
	if e != nil {
		return
	}
	configFileMu.Lock()
	abs, _ := filepath.Abs(path)
	configBaselines[abs] = values
	configFileMu.Unlock()
}
func writeConfigMerged(path string, v any) error {
	values, e := jsonValues(v)
	if e != nil {
		return e
	}
	configFileMu.Lock()
	defer configFileMu.Unlock()
	lock, e := util.AcquireFileLock(path+".write.lock", "configuration", 5*time.Second)
	if e != nil {
		return e
	}
	defer lock.Release()
	disk, e := readConfigValues(path)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	abs, _ := filepath.Abs(path)
	base := configBaselines[abs]
	if base != nil && disk != nil {
		for k, x := range values {
			if reflect.DeepEqual(x, base[k]) {
				if d, ok := disk[k]; ok {
					values[k] = d
				}
			}
		}
		for k, x := range disk {
			if _, ok := values[k]; !ok {
				values[k] = x
			}
		}
	}
	if e = util.WriteJSONAtomic(path, unflatten(values), 0600); e != nil {
		return e
	}
	// Keep the baseline equal to runtime, not to deferred disk changes.
	configBaselines[abs], _ = jsonValues(v)
	return nil
}
func WebSchema(global bool) []WebField {
	t := reflect.TypeOf(Local)
	if global {
		t = reflect.TypeOf(Global)
	}
	out := []WebField{}
	var walk func(reflect.Type, string)
	walk = func(t reflect.Type, prefix string) {
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.Tag.Get("json") == "-" {
				continue
			}
			key := sf.Name
			if prefix != "" {
				key = prefix + "." + key
			}
			if sf.Type.Kind() == reflect.Struct && sf.Type != reflect.TypeOf(time.Time{}) {
				walk(sf.Type, key)
				continue
			}
			label := sf.Tag.Get("web")
			if label == "" || label == "-" {
				label = sf.Name
			}
			f := WebField{Label: label, Key: key, Type: sf.Type.Kind().String(), Admin: global, Effect: "config_reload"}
			if sf.Type == reflect.TypeOf(time.Time{}) {
				f.Type = "datetime"
			}
			f.Secret = key == "Discord.Token" || key == "Factorio.Token"
			f.ReadOnly = sf.Name == "Comment" || strings.HasPrefix(key, "Discord.Roles.RoleCache.") || key == "LastSaveBackup" || key == "PendingSave" || key == "Settings.NewMap" || key == "ModPackList" || key == "Options.Whitelist"
			if key == "Callsign" || strings.HasPrefix(key, "Paths.") || key == "PrimaryServer" || strings.HasPrefix(key, "Discord.Roles.") || key == "Discord.Guild" || key == "Discord.Application" || f.Secret || key == "Factorio.Username" || key == "Options.SoftModOptions.SoftModPath" {
				f.Admin = true
				f.Effect = "chatwire_restart"
			}
			if key == "Port" || key == "Options.SoftModOptions.InjectSoftMod" {
				f.Effect = "factorio_restart"
			}
			if strings.HasPrefix(key, "Settings.Map") || key == "Settings.Scenario" || key == "Settings.Seed" {
				f.Effect = "next_map"
			}
			if f.Type == "int" || f.Type == "float32" {
				min, max := float64(0), float64(2147483647)
				f.Min = &min
				f.Max = &max
			}
			limits := map[string][2]float64{"Port": {1024, 65535}, "Settings.AFKMin": {5, 120}, "Settings.AutosaveMin": {5, 30}, "Settings.Heartbeats": {1, 240}, "Settings.Seed": {0, 4294967295}, "Options.Speed": {0.01, 10}, "Options.AutosaveMax": {64, 1024}, "Options.RconOffset": {1, 64511}, "Options.PlayStartHour": {0, 23}, "Options.PlayEndHour": {0, 23}, "Options.ResetHour": {0, 23}, "Options.ResetInterval.Months": {0, 3}, "Options.ResetInterval.Weeks": {0, 13}, "Options.ResetInterval.Days": {0, 93}, "Options.ResetInterval.Hours": {0, 2232}, "Options.PlayerPollIntervalSec": {1, 86400}, "Options.RoleRefreshIntervalSec": {1, 86400}, "Options.MapResetCheckIntervalSec": {1, 86400}}
			if l, ok := limits[key]; ok {
				a, b := l[0], l[1]
				f.Min = &a
				f.Max = &b
			}
			if key == "Settings.MapPreset" {
				f.Choices = constants.MapTypes
			}
			if key == "Paths.DataFiles.DBFormat" {
				f.Choices = []string{"binary", "json"}
			}
			scope := "local"
			if global {
				scope = "global"
			}
			if !classifiedWebFields[scope+"."+key] {
				f.ReadOnly = true
				f.Admin = true
				f.Secret = true
			}
			out = append(out, f)
		}
	}
	walk(t, "")
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
func WebRead(global bool) (WebSettings, error) {
	values, e := readConfigValues(configPath(global))
	if e != nil {
		return WebSettings{}, e
	}
	rev := revision(values)
	out := map[string]any{}
	for _, f := range WebSchema(global) {
		if f.Secret {
			out[f.Key] = map[string]bool{"configured": values[f.Key] != nil && values[f.Key] != ""}
		} else if !strings.HasSuffix(f.Key, ".Comment") && !strings.Contains(f.Key, "RoleCache") {
			out[f.Key] = values[f.Key]
		}
	}
	return WebSettings{rev, out, "Saved values; use reload-config or the indicated restart to apply changes."}, nil
}

var ErrRevision = errors.New("configuration changed; refresh before saving")

func WebPatch(global, admin bool, expected string, patch map[string]json.RawMessage) (WebSettings, error) {
	path := configPath(global)
	configFileMu.Lock()
	defer configFileMu.Unlock()
	lock, e := util.AcquireFileLock(path+".write.lock", "web-settings", 5*time.Second)
	if e != nil {
		return WebSettings{}, e
	}
	defer lock.Release()
	values, e := readConfigValues(path)
	if e != nil {
		return WebSettings{}, e
	}
	if expected != revision(values) {
		return WebSettings{}, ErrRevision
	}
	fields := map[string]WebField{}
	for _, f := range WebSchema(global) {
		fields[f.Key] = f
	}
	for key, raw := range patch {
		f, ok := fields[key]
		if !ok || f.ReadOnly || f.Admin && !admin {
			return WebSettings{}, fmt.Errorf("%s is not editable with your permissions", key)
		}
		if bytes.Equal(raw, []byte("null")) {
			return WebSettings{}, fmt.Errorf("%s cannot be null", key)
		}
		var v any
		if e = json.Unmarshal(raw, &v); e != nil {
			return WebSettings{}, errors.New("invalid value")
		}
		switch f.Type {
		case "bool":
			if _, ok := v.(bool); !ok {
				return WebSettings{}, fmt.Errorf("%s must be boolean", key)
			}
		case "int", "float32":
			n, ok := v.(float64)
			if !ok || n < *f.Min || n > *f.Max || f.Type == "int" && n != float64(int64(n)) {
				return WebSettings{}, fmt.Errorf("%s is outside its numeric range", key)
			}
		case "string", "datetime":
			s, ok := v.(string)
			if !ok || len(s) > 4096 {
				return WebSettings{}, fmt.Errorf("%s must be text of at most 4096 bytes", key)
			}
			if f.Type == "datetime" {
				if _, e = time.Parse(time.RFC3339, s); e != nil {
					return WebSettings{}, fmt.Errorf("%s must be RFC3339", key)
				}
			}
			if len(f.Choices) > 0 {
				valid := false
				for _, c := range f.Choices {
					valid = valid || s == c
				}
				if !valid {
					return WebSettings{}, fmt.Errorf("%s is not a supported choice", key)
				}
			}
		default:
			return WebSettings{}, fmt.Errorf("%s cannot be edited", key)
		}
		if key == "Name" && (len(v.(string)) < 1 || len(v.(string)) > 64 || strings.ContainsAny(v.(string), "/\\\x00")) {
			return WebSettings{}, errors.New("Name must be 1-64 characters without path separators")
		}
		if (key == "Callsign" || key == "PrimaryServer") && (len(v.(string)) < 1 || len(v.(string)) > 2 || strings.ContainsAny(v.(string), "/\\.")) {
			return WebSettings{}, errors.New("callsign must be 1-2 characters without path separators")
		}
		if key == "Settings.MapGenerator" || key == "Settings.Scenario" {
			if strings.ContainsAny(v.(string), "/\\") || v == ".." {
				return WebSettings{}, errors.New("generator/scenario must be a name")
			}
		}
		if key == "Options.SoftModOptions.OneLife" && values[key] == true && v == false {
			return WebSettings{}, errors.New("one-life mode cannot be disabled on the current map")
		}
		values[key] = v
	}
	if values["Options.MembersOnly"] == true && values["Options.RegularsOnly"] == true {
		return WebSettings{}, errors.New("members-only and regulars-only are mutually exclusive")
	}
	if global {
		if v, ok := values["GroupName"].(string); ok && (len(v) < 2 || len(v) > 5) {
			return WebSettings{}, errors.New("GroupName must be 2-5 characters")
		}
	} else {
		g, e := readConfigValues(constants.CWGlobalConfig)
		if e != nil {
			return WebSettings{}, errors.New("cannot validate global port settings")
		}
		p, _ := values["Port"].(float64)
		offset, _ := g["Options.RconOffset"].(float64)
		if p+offset > 65535 {
			return WebSettings{}, errors.New("computed RCON port exceeds 65535")
		}
		months, _ := values["Options.ResetInterval.Months"].(float64)
		weeks, _ := values["Options.ResetInterval.Weeks"].(float64)
		days, _ := values["Options.ResetInterval.Days"].(float64)
		hours, _ := values["Options.ResetInterval.Hours"].(float64)
		if months*31+weeks*7+days+hours/24 > 93 {
			return WebSettings{}, errors.New("combined map reset interval exceeds three months")
		}
	}
	if !global {
		intervalChanged := false
		for key := range patch {
			if strings.HasPrefix(key, "Options.ResetInterval.") {
				intervalChanged = true
			}
		}
		if intervalChanged {
			if _, explicit := patch["Options.NextReset"]; explicit {
				return WebSettings{}, errors.New("change the reset interval or the explicit reset date in separate saves")
			}
			get := func(key string) int { v, _ := values[key].(float64); return int(v) }
			months, weeks, days, hours := get("Options.ResetInterval.Months"), get("Options.ResetInterval.Weeks"), get("Options.ResetInterval.Days"), get("Options.ResetInterval.Hours")
			next := time.Time{}
			if months+weeks+days+hours > 0 {
				now := time.Now().UTC()
				base := now
				if hour := get("Options.ResetHour"); hour > 0 {
					base = time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
				}
				next = base.AddDate(0, months, days+weeks*7).Add(time.Duration(hours) * time.Hour).Round(time.Second)
			}
			values["Options.NextReset"] = next.Format(time.RFC3339)
		} else if _, explicit := patch["Options.NextReset"]; explicit {
			next, e := time.Parse(time.RFC3339, values["Options.NextReset"].(string))
			if e != nil || (!next.IsZero() && !next.After(time.Now())) {
				return WebSettings{}, errors.New("reset date must be in the future, or the zero date to disable it")
			}
		}
	}
	// Decode to the real type as a final type/overflow check, without publishing it.
	b, _ := json.Marshal(unflatten(values))
	if global {
		var v globalConfigAlias
		if e = json.Unmarshal(b, &v); e != nil {
			return WebSettings{}, errors.New("invalid global settings")
		}
	} else {
		var v local
		if e = json.Unmarshal(b, &v); e != nil {
			return WebSettings{}, errors.New("invalid local settings")
		}
	}
	if e = util.WriteJSONAtomic(path, unflatten(values), 0600); e != nil {
		return WebSettings{}, errors.New("unable to persist settings")
	}
	return WebRead(global)
}

type globalConfigAlias global

// RuntimeConfigRevision reports the last config snapshot loaded/written by this
// process. It does not inspect mutable runtime fields from an HTTP goroutine.
func RuntimeConfigRevision(global bool) string {
	abs, _ := filepath.Abs(configPath(global))
	configFileMu.Lock()
	defer configFileMu.Unlock()
	if v := configBaselines[abs]; v != nil {
		return revision(v)
	}
	return ""
}

// RedactWebText removes known current and pending service credentials from
// diagnostic text returned to a browser. Configuration values never enter logs.
func RedactWebText(text string) string {
	secrets := []string{}
	if values, e := readConfigValues(constants.CWGlobalConfig); e == nil {
		for _, key := range []string{"Discord.Token", "Factorio.Token"} {
			if v, ok := values[key].(string); ok && v != "" {
				secrets = append(secrets, v)
			}
		}
	}
	abs, _ := filepath.Abs(constants.CWGlobalConfig)
	configFileMu.Lock()
	for _, key := range []string{"Discord.Token", "Factorio.Token"} {
		if v, ok := configBaselines[abs][key].(string); ok && v != "" {
			secrets = append(secrets, v)
		}
	}
	configFileMu.Unlock()
	for _, secret := range secrets {
		text = strings.ReplaceAll(text, secret, "[redacted]")
	}
	return text
}
