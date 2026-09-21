package controlruntime

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"ChatWire/factUpdater"
	"ChatWire/glob"
	"ChatWire/modedit"
	"ChatWire/modupdate"
	"ChatWire/support"
	"ChatWire/webcontrol"
	"errors"

	"os"
	"path/filepath"
	"strings"
	"time"
)

func (rt *Runtime) execute(id string, a webcontrol.Actor, action string, p Params) (any, error) {
	if len(p.Reason) > 500 || len(p.Command) > 8192 || len(p.ExchangeString) > 512000 {
		return nil, fail("Input is too long.")
	}
	if p.Reason == "" {
		p.Reason = "Moderator web control"
	}
	lifecycle := map[string]fact.ActionKind{"factorio-start": fact.ActionStart, "factorio-stop": fact.ActionStop, "factorio-restart": fact.ActionRestartFactorio, "chatwire-restart": fact.ActionRestartChatWire, "map-load": fact.ActionChangeMap, "map-reset": fact.ActionMapReset}
	if kind, ok := lifecycle[action]; ok {
		req := fact.Request{Kind: kind, Reason: p.Reason, RequestID: id, WhenEmpty: p.WhenEmpty, ForceChatWireExit: p.Force}
		if kind == fact.ActionRestartChatWire {
			req.BeforeExit = func() error {
				return rt.Jobs.Update(id, "waiting_for_restart", "exiting", map[string]string{"previous_boot_id": rt.boot}, "")
			}
		}
		if p.WhenEmpty && kind != fact.ActionRestartFactorio && kind != fact.ActionRestartChatWire {
			return nil, fail("Waiting for empty is only supported for restarts.")
		}
		if p.Force && kind != fact.ActionRestartChatWire {
			return nil, fail("Force applies only to ChatWire restart.")
		}
		if kind == fact.ActionChangeMap {
			name, e := saveName(p.SaveID)
			if e != nil {
				return nil, e
			}
			if file, openErr := openSave(name); openErr != nil {
				return nil, fail("Save not found.")
			} else {
				file.Close()
			}
			req.SaveName = strings.TrimSuffix(name, ".zip")
		}
		if kind == fact.ActionStart || kind == fact.ActionRestartFactorio {
			if !support.WithinHours() {
				return nil, fail("Current time is outside configured play hours.")
			}
		}
		if kind == fact.ActionStop {
			fact.SetAutolaunch(false, false)
		}
		if p.WhenEmpty {
			_ = rt.Jobs.Update(id, "waiting_for_empty", "waiting", nil, "")
		}
		if e := fact.SubmitLifecycleRequestAndWait(req); e != nil {
			return nil, fail("Lifecycle operation failed; inspect instance logs.")
		}
		if kind == fact.ActionStart || kind == fact.ActionRestartFactorio || kind == fact.ActionChangeMap || kind == fact.ActionMapReset {
			_ = rt.Jobs.Update(id, "running", "waiting_for_ready", nil, "")
			if err := waitReady(); err != nil {
				return nil, err
			}
			fact.SetAutolaunch(true, false)
		}
		return map[string]any{"lifecycle": fact.GetLifecycleState()}, nil
	}
	switch action {
	case "config-reload":
		if e := support.ReloadConfigFilesResult("web"); e != nil {
			return nil, e
		}
		return map[string]string{"status": "reloaded"}, nil

	case "map-create", "map-exchange":
		if fact.GetLifecycleState().Phase != fact.LifecycleStopped {
			return nil, fail("Stop Factorio before generating a map.")
		}
		var name string
		var e error
		if action == "map-create" {
			name, e = fact.GenNewMap()
		} else {
			name, e = fact.GenCustomMapFromExchange(p.ExchangeString)
		}
		if e != nil {
			return nil, fail("Map generation failed; inspect instance logs.")
		}
		return map[string]string{"save": filepath.Base(name)}, nil
	case "rcon":
		out, e := fact.ExecuteRCON(p.Command)
		if errors.Is(e, fact.ErrRCONOutcomeUnknown) {
			e = webcontrol.UnknownOutcomeError{Err: e}
		}
		return map[string]string{"output": cfg.RedactWebText(out)}, e
	case "mods-update", "mods-sync", "factorio-update", "factorio-install", "mods-edit", "mods-clear", "mods-clear-history", "upload-apply":
		if action != "factorio-update" && action != "factorio-install" {
			glob.UpdatersLock.Lock()
			defer glob.UpdatersLock.Unlock()
		}
		if action == "mods-edit" || action == "mods-clear" {
			unlock, e := cfg.LockControlResources()
			if e != nil {
				return nil, fail("Shared game files are busy.")
			}
			defer unlock()
		}

		switch action {
		case "mods-update":
			updated, e := modupdate.CheckModsForControl()
			if e != nil {
				return nil, fail("Mod update failed; inspect instance logs.")
			}
			return map[string]any{"updated": updated, "restart_when_empty": updated && fact.GetLifecycleState().Phase != fact.LifecycleStopped}, nil
		case "mods-sync":
			if fact.GetLifecycleState().Phase != fact.LifecycleStopped {
				return nil, fail("Stop Factorio before syncing mods.")
			}
			if !support.SyncModsService("") {
				return nil, fail("Mod sync failed.")
			}
			return "Mods synchronized.", nil
		case "factorio-update", "factorio-install":
			_, msg, bad, _ := factUpdater.DoQuickLatest(action == "factorio-install")
			if bad {
				return nil, fail("Factorio update/install failed; inspect instance logs.")
			}
			return msg, nil
		case "mods-edit":
			return editMods(p)
		case "mods-clear":
			if fact.GetLifecycleState().Phase != fact.LifecycleStopped {
				return nil, fail("Stop Factorio before clearing mods.")
			}
			root := cfg.GetModsFolder()
			if root == "" || filepath.Clean(root) == "/" {
				return nil, fail("Invalid mods directory.")
			}
			entries, e := os.ReadDir(root)
			if e != nil {
				return nil, fail("Cannot read mods directory.")
			}
			for _, entry := range entries {
				if e = os.RemoveAll(filepath.Join(root, entry.Name())); e != nil {
					return nil, fail("Unable to remove a mod file.")
				}
			}
			if !modupdate.WriteModsList(modupdate.ModListData{}) {
				return nil, fail("Cannot write mod list.")
			}
			return "Mods cleared.", nil
		case "mods-clear-history":
			return modupdate.ClearHistory(), nil
		case "upload-apply":
			return rt.applyUploads(a, p)
		}
	case "map-archive":
		return archiveSave(p.SaveID)
	case "player-level":
		return setPlayerLevel(a, p)
	case "ip-ban", "ip-unban":
		return firewallAction(action, p.IP)
	}
	return nil, errors.New("unsupported action")
}
func editMods(p Params) (any, error) {
	if fact.GetLifecycleState().Phase != fact.LifecycleStopped {
		return nil, fail("Stop Factorio before editing mods.")
	}
	if len(p.Mods) == 0 || len(p.Mods) > 100 {
		return nil, fail("Supply 1-100 mod names.")
	}
	for _, name := range p.Mods {
		if !validModName(name) {
			return nil, fail("Use mod names, without URLs or path separators.")
		}
	}
	list, e := modupdate.GetModList()
	if e != nil {
		return nil, fail("Cannot read mod list.")
	}
	if p.Operation == "version" {
		if len(p.Mods) != 1 {
			return nil, fail("Set one mod version at a time.")
		}
		if e = modedit.SetVersion(p.Mods[0], p.Version); e != nil {
			return nil, fail("Invalid mod version preference.")
		}
		return "Version preference saved.", nil
	}
	for _, name := range p.Mods {
		if modupdate.IsBaseMod(name) && p.Operation != "enable" {
			return nil, fail("Base mods cannot be removed or disabled.")
		}
		idx := -1
		for i, m := range list.Mods {
			if m.Name == name {
				idx = i
				break
			}
		}
		switch p.Operation {
		case "add":
			if idx >= 0 {
				return nil, fail("A requested mod is already listed.")
			}
			info, e := modupdate.DownloadModInfo(name)
			if e != nil || info.Name != name {
				return nil, fail("Mod not found in Factorio portal.")
			}
			list.Mods = append(list.Mods, modupdate.ModData{Name: name, Enabled: true})
		case "enable", "disable":
			if idx < 0 {
				return nil, fail("Mod is not in the list.")
			}
			list.Mods[idx].Enabled = p.Operation == "enable"
		case "remove":
			if idx < 0 {
				return nil, fail("Mod is not in the list.")
			}
			list.Mods = append(list.Mods[:idx], list.Mods[idx+1:]...)
		default:
			return nil, fail("Operation must be add, remove, enable, disable, or version.")
		}
	}
	if !modupdate.WriteModsList(list) {
		return nil, fail("Unable to write mod list.")
	}
	return "Mod list saved; run mod update to download missing versions.", nil
}
func validModName(name string) bool {
	if name == "" || len(name) > 200 {
		return false
	}
	for _, c := range name {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func waitReady() error {
	deadline := time.NewTimer(15 * time.Minute)
	defer deadline.Stop()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	for {
		state := fact.GetLifecycleState()
		if state.Phase == fact.LifecycleRunning && state.Booted {
			return nil
		}
		if state.Phase == fact.LifecycleStopped {
			return fail("Factorio stopped before becoming ready.")
		}
		select {
		case <-deadline.C:
			return fail("Timed out waiting for Factorio readiness; inspect the current state.")
		case <-tick.C:
		}
	}
}
