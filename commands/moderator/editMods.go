package moderator

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modedit"
	"ChatWire/modupdate"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func EditMods(cmd *glob.CommandData, i *discordgo.InteractionCreate) {
	// Acknowledge before waiting for another updater or contacting the mod portal.
	for _, option := range i.ApplicationCommandData().Options {
		if strings.EqualFold(option.Name, "add-mod") {
			disc.InteractionEphemeralResponseColor(i, "Edit-Mods", "Checking requested mods and downloading available releases…", glob.COLOR_CYAN)
			break
		}
	}
	glob.UpdatersLock.Lock()
	defer glob.UpdatersLock.Unlock()

	var msg string
	tmsg := ""

	optionsList := i.ApplicationCommandData().Options
	if len(optionsList) == 0 {
		emsg := "You must choose at least one option."
		disc.InteractionEphemeralResponseColor(i, "Error", emsg, glob.COLOR_RED)
		return
	}

	// Reply early for simple enable/disable requests, then let the normal toggle
	// path complete without trying to send a second interaction response.
	if len(optionsList) == 1 {
		option := optionsList[0]
		switch strings.ToLower(option.Name) {
		case "enable-mod":
			handleToggleImmediate(i, option.StringValue(), true)
			return
		case "disable-mod":
			handleToggleImmediate(i, option.StringValue(), false)
			return
		}
	}

	for _, option := range optionsList {
		oName := strings.ToLower(option.Name)

		switch oName {
		case "mod-history":
			tmsg = tmsg + modupdate.ListHistory(false)
		case "full-history":
			tmsg = tmsg + modupdate.ListHistory(true)
		case "clear-history":
			tmsg = tmsg + modupdate.ClearHistory()
		case "enable-mod":
			msg = toggleMod(i, option.StringValue(), true)
			tmsg = tmsg + msg + "\n"
		case "disable-mod":
			msg = toggleMod(i, option.StringValue(), false)
			tmsg = tmsg + msg + "\n"
		case "add-mod":
			tmsg = tmsg + addMod(i, option.StringValue()) + "\n"
		case "set-version-pref":
			tmsg = tmsg + setVersion(option.StringValue())
		case "clear-all-mods":
			msg = clearAllMods()
			tmsg = tmsg + msg + "\n"
		default:
			msg = oName + " is not a valid option."
			tmsg = tmsg + msg + "\n"
		}
	}

	//Check if we need to proceed
	if tmsg == "" {
		tmsg = "Unknown error"
	}

	if len(tmsg) > constants.MaxDiscordMsgLen {
		disc.InteractionEphemeralFileResponse(i, "Edit-Mods", "See attached results.", "edit-mods-results.txt", []byte(tmsg))
	} else {
		disc.InteractionEphemeralResponseColor(i, "Edit-Mods", tmsg, glob.COLOR_CYAN)
	}
}

func handleToggleImmediate(i *discordgo.InteractionCreate, name string, value bool) {
	msg, ok := previewToggleResponse(name, value)
	if !ok {
		disc.InteractionEphemeralResponseColor(i, "Edit-Mods", msg, glob.COLOR_RED)
		return
	}

	disc.InteractionEphemeralResponseColor(i, "Edit-Mods", msg, glob.COLOR_CYAN)
	result := toggleMod(i, name, value)
	if result == "" {
		return
	}
	cwlog.DoLogCW("EditMods %s: %s", enableStr(value, false), result)
}

func previewToggleResponse(name string, value bool) (string, bool) {
	if fact.FactorioBooted || fact.FactIsRunning {
		return "Factorio is currently running. You must stop Factorio first.", false
	}
	if name == "" {
		return "You must specify a mod(s) to " + enableStr(value, false) + ".", false
	}

	modList, _ := modupdate.GetModList()
	input := strings.ReplaceAll(name, " ", "")
	parts := strings.Split(input, ",")

	responses := []string{}
	for _, part := range parts {
		if part == "" {
			continue
		}

		found := false
		for _, mod := range modList.Mods {
			if mod.Name == "base" {
				continue
			}
			if strings.EqualFold(mod.Name, part) {
				responses = append(responses, "The mod '"+mod.Name+"' is now "+enableStr(value, true)+".")
				found = true
				break
			}
		}
		if !found {
			return "There is no mod by the name: " + part, false
		}
	}

	if len(responses) == 0 {
		return "You must specify a mod(s) to " + enableStr(value, false) + ".", false
	}

	return strings.Join(responses, "\n"), true
}

func clearAllMods() string {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return emsg
	}
	err := os.RemoveAll(cfg.GetModsFolder())
	if err != nil {
		return "Unable to delete mods folder: " + err.Error()
	}
	err = os.Mkdir(cfg.GetModsFolder(), 0755)
	if err != nil {
		return "Unable to create a new mods folder: " + err.Error()
	}

	modupdate.WriteModsList(modupdate.ModListData{})
	return "All mods, settings and old mods were deleted."
}

func addMod(i *discordgo.InteractionCreate, input string) string {
	return addMods(input, i.Member.User.Username, parseModName, func() (bool, error) {
		return modupdate.CheckModUpdates(false, false, false, false)
	})
}

// A list entry is a request to install a mod, not proof that it downloaded.
// Keep failed requests retryable and process the rest of a comma-separated batch.
func addMods(input, actor string, resolve func(string) (string, bool), update func() (bool, error)) string {
	modList, err := modupdate.GetModList()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "Unable to read mod list: " + err.Error()
	}
	files, err := modupdate.GetModFiles()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "Unable to read mod files: " + err.Error()
	}
	installed := map[string]bool{}
	for _, file := range files {
		installed[strings.ToLower(file.Name)] = true
	}
	var responses, requested, added []string
	seen := map[string]bool{}
	for _, inputName := range strings.Split(input, ",") {
		inputName = strings.TrimSpace(inputName)
		if inputName == "" {
			continue
		}
		name, ok := resolve(inputName)
		if !ok {
			responses = append(responses, name)
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		listed := false
		for _, item := range modList.Mods {
			if strings.EqualFold(item.Name, name) {
				listed = true
				if !item.Enabled {
					responses = append(responses, name+": already listed but disabled; use enable-mod first.")
					name = ""
				}
				break
			}
		}
		if name == "" {
			continue
		}
		if installed[key] || modupdate.IsBaseMod(name) {
			responses = append(responses, name+": already installed.")
			continue
		}
		requested = append(requested, name)
		if !listed {
			modList.Mods = append(modList.Mods, modupdate.ModData{Name: name, Enabled: true})
			added = append(added, name)
		}
	}
	if len(requested) > 0 {
		if len(added) > 0 && !modupdate.WriteModsList(modList) {
			responses = append(responses, "Unable to save mod list; no downloads were attempted.")
			return strings.Join(responses, "\n")
		}
		for _, name := range added {
			modupdate.AddModHistory(modupdate.ModHistoryItem{
				Name: name, Notes: "Added by " + actor, Date: time.Now(),
			})
		}
		_, updateErr := update()
		if updateErr != nil {
			responses = append(responses, "Mod installation/update failed: "+updateErr.Error())
		}
		files, err = modupdate.GetModFiles()
		if err != nil {
			responses = append(responses, "Unable to verify downloaded mods: "+err.Error())
		} else {
			for _, file := range files {
				installed[strings.ToLower(file.Name)] = true
			}
			for _, name := range requested {
				if installed[strings.ToLower(name)] {
					if updateErr != nil {
						responses = append(responses, name+": downloaded, but the batch failed; resolve the reported errors before restarting.")
					} else {
						responses = append(responses, name+": installed; applies on the next Factorio start.")
					}
				} else {
					responses = append(responses, name+": not installed. Still listed for retry; check release compatibility and the reported errors, then run add-mod again.")
				}
			}
		}
	}
	if len(responses) == 0 {
		return "You must specify mods to add."
	}
	return strings.Join(responses, "\n")
}

func modInputName(input string) (string, error) {
	name := strings.TrimSpace(input)
	if strings.Contains(name, "://") {
		u, err := url.Parse(name)
		if err != nil {
			return "", fmt.Errorf("invalid mod URL: %w", err)
		}
		if (u.Scheme != "https" && u.Scheme != "http") || !strings.EqualFold(u.Host, "mods.factorio.com") {
			return "", fmt.Errorf("expected a mods.factorio.com URL")
		}
		// URL.Path is decoded, so names such as Orphan%20Finder are passed to
		// DownloadModInfo as Orphan Finder and escaped exactly once there.
		parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
		if len(parts) < 2 || (parts[0] != "mod" && parts[0] != "download") {
			return "", fmt.Errorf("expected a mod or download URL")
		}
		name = parts[1]
	}
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("missing mod name")
	}
	return name, nil
}

func parseModName(input string) (string, bool) {
	name, err := modInputName(input)
	if err != nil {
		return input + ": " + err.Error(), false
	}
	if modupdate.IsBaseMod(name) {
		return name, true
	}
	modData, err := modupdate.DownloadModInfo(name)
	if err != nil {
		return "Error looking up mod " + name + ": " + err.Error(), false
	}
	return modData.Name, true
}

func toggleMod(i *discordgo.InteractionCreate, name string, value bool) string {
	if fact.FactorioBooted || fact.FactIsRunning {
		emsg := "Factorio is currently running. You must stop Factorio first."
		return emsg
	}
	if name == "" {
		emsg := "You must specify a mod(s) to " + enableStr(value, false) + "."
		return emsg
	}

	modFiles, err := modupdate.GetModFiles()
	if err != nil {
		return "Unable to read mod files."
	}
	modList, _ := modupdate.GetModList()
	mergedMods := modupdate.MergeModLists(modFiles, modList)

	//Remove spaces
	input := strings.ReplaceAll(name, " ", "")
	parts := strings.Split(input, ",")

	emsg := ""
	dirty := false
	for _, part := range parts {
		found := false
		for m, mod := range mergedMods {
			if mod.Name == "base" {
				continue
			}
			if strings.EqualFold(mod.Name, part) {
				if mod.Enabled != value {
					emsg = emsg + "The mod '" + mod.Name + "' is now " + enableStr(value, true) + "."
					mergedMods[m].Enabled = value

					modupdate.AddModHistory(modupdate.ModHistoryItem{
						Name: mod.Name, Notes: enableStr(value, true) + " by " + i.Member.User.Username, Date: time.Now()})

					dirty = true

				} else {
					emsg = emsg + "The mod '" + mod.Name + "' was already " + enableStr(value, true) + "!"
				}
				found = true
				break
			}
		}
		if !found {
			emsg = emsg + "There is no mod by the name: " + input
		}
	}
	if dirty {
		modupdate.WriteModHistory()
		outMods := []modupdate.ModData{}
		for _, item := range mergedMods {
			outMods = append(outMods, modupdate.ModData{Name: item.Name, Enabled: item.Enabled})
		}
		outputList := modupdate.ModListData{Mods: outMods}
		modupdate.WriteModsList(outputList)
	}
	return emsg
}

/* Bool to string */
func enableStr(b bool, pastTense bool) string {
	if pastTense {
		if b {
			return "enabled"
		} else {
			return "disabled"
		}
	} else {
		if b {
			return "enable"
		} else {
			return "disable"
		}
	}
}

func setVersion(input string) string {
	if input == "" {
		return "You must specify a mod=version pair."
	}
	input = strings.ReplaceAll(input, " ", "")
	parts := strings.Split(input, ",")

	modFiles, _ := modupdate.GetModFiles()
	valid := map[string]bool{}
	for _, m := range modFiles {
		valid[strings.ToLower(m.Name)] = true
	}

	buf := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		seg := strings.SplitN(p, "=", 2)
		if len(seg) != 2 {
			if buf != "" {
				buf += "\n"
			}
			buf += p + ": invalid format"
			continue
		}
		name := strings.ToLower(seg[0])
		vers := seg[1]
		if !valid[name] {
			if buf != "" {
				buf += "\n"
			}
			buf += seg[0] + ": mod not installed"
			continue
		}
		if err := modedit.SetVersion(name, vers); err != nil {
			if buf != "" {
				buf += "\n"
			}
			buf += seg[0] + ": " + err.Error()
			continue
		}
		if buf != "" {
			buf += "\n"
		}
		buf += seg[0] + " set to " + vers
	}

	if _, err := modupdate.CheckModUpdates(false, false, false, false); err != nil {
		cwlog.DoLogCW("CheckModUpdates failed: %v", err)
	}
	return buf
}
