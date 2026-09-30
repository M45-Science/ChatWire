package modupdate

import (
	"ChatWire/cfg"
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/disc"
	"ChatWire/fact"
	"ChatWire/glob"
	"ChatWire/modedit"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	modPortalURL              = "https://mods.factorio.com/api/mods/%v/full"
	displayURL                = "https://mods.factorio.com/mod/%v/changelog"
	downloadPrefix            = "https://mods.factorio.com"
	downloadSuffix            = "?username=%v&token=%v"
	modUpdateTitle            = "Found Mod Updates"
	modUpdateProgressInterval = time.Minute
)

var downloadModInfo = DownloadModInfo

func CheckModsManual(force bool) {
	checkMods(force, true, true)
}

func CheckModsAuto(force bool) {
	checkMods(force, false, false)
}

func checkMods(force bool, reportNone bool, emitProgress bool) {

	if !cfg.Local.Options.ModUpdate && !force {
		return
	}

	updated, err := CheckModUpdates(false, emitProgress, !reportNone, !reportNone)
	if reportNone || updated {
		if err != nil {
			glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Warning:", err.Error(), glob.COLOR_CYAN))
		}
	}
	if updated && fact.FactIsRunning {
		_ = fact.SubmitLifecycleRequest(fact.Request{
			Kind:      fact.ActionRestartFactorio,
			Reason:    "Rebooting Factorio.",
			WhenEmpty: true,
		})
	}
}

const resolveDepsDebug = false

type modUpdateProgress struct {
	token    string
	lastEmit time.Time
}

func newModUpdateProgress(token string) modUpdateProgress {
	return modUpdateProgress{
		token:    token,
		lastEmit: time.Now(),
	}
}

func (p *modUpdateProgress) emit(description string) {
	if p == nil || p.token == "" || description == "" {
		return
	}
	if time.Since(p.lastEmit) < modUpdateProgressInterval {
		return
	}

	fact.UpdateOperationProgress(p.token, "Mod Updates", description, glob.COLOR_CYAN)
	p.lastEmit = time.Now()
}

func dependencySatisfied(depInfo depRequires, installedVersion string, planned []downloadData) (bool, error) {
	version := installedVersion
	for _, item := range planned {
		if item.Name == depInfo.name {
			version = item.Version
			break
		}
	}

	if version == "" {
		return false, nil
	}
	if depInfo.version == "" {
		return true, nil
	}

	return checkVersion(depInfo.equality, depInfo.version, version)
}

func releaseMatchesFactorioVersion(rel modRelease) bool {
	modVersion := strings.TrimSpace(rel.InfoJSON.FactorioVersion)
	gameVersion := strings.TrimSpace(fact.FactorioVersion)
	if modVersion == "" || gameVersion == "" || strings.EqualFold(gameVersion, constants.Unknown) {
		return true
	}

	modParts, err := versionToInt(modVersion)
	if err != nil {
		cwlog.DoLogCW("resolveDeps: invalid factorio_version for release %s: %v", rel.Version, err)
		return false
	}
	gameParts, err := versionToInt(gameVersion)
	if err != nil {
		cwlog.DoLogCW("resolveDeps: invalid Factorio version %s: %v", gameVersion, err)
		return true
	}

	return modParts.parts[0] == gameParts.parts[0] && modParts.parts[1] == gameParts.parts[1]
}

func resolveDeps(modPortalData []modPortalFullData, wasDep bool, depth int, parents []string, progress *modUpdateProgress) ([]downloadData, error) {
	return resolveDepsWithPrefs(modPortalData, wasDep, depth, parents, modedit.VersionPrefs{}, progress)
}

func resolveDepsWithPrefs(modPortalData []modPortalFullData, wasDep bool, depth int, parents []string, prefs modedit.VersionPrefs, progress *modUpdateProgress) ([]downloadData, error) {

	if depth > 10 {
		return []downloadData{}, nil
	}

	var downloadMods []downloadData
	for i, item := range modPortalData {
		if progress != nil {
			progress.emit(formatResolveDepsProgress(item.Name, i+1, len(modPortalData), depth))
		}

		// Don't follow circular deps
		circular := false
		for _, parent := range parents {
			if item.Name == parent {
				circular = true
				break
			}
		}
		if circular {
			continue
		}

		candidate := modRelease{Version: "0.0.0"}
		if item.installed.Version != "" {
			candidate.Version = item.installed.Version
		}
		var candidateDeps []downloadData
		preferred := modedit.GetVersion(prefs, item.Name)
		if strings.EqualFold(preferred, "auto") {
			preferred = ""
		}
		preferredResolved := false

		//Check all releases
		for _, rel := range item.Releases {
			if preferred != "" && rel.Version != preferred {
				continue
			}
			//cwlog.DoLogCW("RELEASES: %v: Local: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)

			releaseNewer := false
			if preferred != "" || item.installed.Version == "" {
				releaseNewer = true
			} else {
				var err error
				releaseNewer, err = checkVersion(EO_GREATER, item.installed.Version, rel.Version)
				if err != nil {
					return []downloadData{}, err
				}
			}
			//If release is newer, check candidate
			if releaseNewer {
				if resolveDepsDebug {
					cwlog.DoLogCW("NEWER: %v: LOCAL: %v, Rel: %v", item.Name, item.installed.Version, rel.Version)
				}
				releaseNewer, err := checkVersion(EO_GREATER, candidate.Version, rel.Version)
				if err != nil {
					return []downloadData{}, err
				}
				//If release is newer check deps
				if releaseNewer || preferred != "" {
					if !releaseMatchesFactorioVersion(rel) {
						continue
					}

					depsMet := true
					var releaseDeps []downloadData

					for _, dep := range rel.InfoJSON.Dependencies {
						depInfo := parseDep(dep)
						if depInfo.incompatible {
							continue
						}
						//We can ignore optional deps
						if depInfo.optional {
							continue
						}

						//Check base mod version
						if resolveDepsDebug {
							cwlog.DoLogCW("dep name: %v, eq: %v, vers: %v :: inc: %v", depInfo.name, operatorToString(depInfo.equality), depInfo.version, depInfo.incompatible)
						}
						//If dep is a base mod, check it here
						if IsBaseMod(depInfo.name) {
							if depInfo.version != "" {
								good, err := checkVersion(depInfo.equality, depInfo.version, fact.FactorioVersion)
								if !good || err != nil {
									depsMet = false
									continue
								}
							}
							if resolveDepsDebug {
								cwlog.DoLogCW("base dep available: %v", dep)
							}
						} else { //Dep is a mod, check if we have it
							haveDepInfo := false
							depPortalInfo := modPortalFullData{}
							if resolveDepsDebug {
								cwlog.DoLogCW("CHECKING DEP %v-%v", depInfo.name, depInfo.version)
							}
							for _, item := range modPortalData {
								if item.Name == depInfo.name {
									haveDepInfo = true
									depPortalInfo = item
									break
								}
							}
							//We do not have the dep, download info
							if !haveDepInfo {
								depPortalInfo, err = downloadModInfo(depInfo.name)
								if err != nil {
									cwlog.DoLogCW("resolveDeps: dep: DownloadModInfo: %v", err)
									return []downloadData{}, err
								}
							}
							// Recursively check dep's deps
							dl, err := resolveDepsWithPrefs([]modPortalFullData{depPortalInfo}, true, depth+1, append(parents, item.Name), prefs, progress)
							if err != nil {
								cwlog.DoLogCW("resolveDeps: dep: resolveDeps: %v", err)
								return []downloadData{}, err
							}
							good, err := dependencySatisfied(depInfo, depPortalInfo.installed.Version, dl)
							if err != nil {
								cwlog.DoLogCW("resolveDeps: dep: dependencySatisfied: %v", err)
								return []downloadData{}, err
							}
							if !good {
								depsMet = false
								continue
							}
							//Download dep and all of dep's deps.
							if len(dl) > 0 {
								for _, depDownload := range dl {
									if depDownload.RequiredByName == "" {
										depDownload.RequiredByName = item.Name
										depDownload.RequiredByVersion = rel.Version
									}
									releaseDeps = addDownload(depDownload, releaseDeps)
								}
							}
						}
					}
					//If deps were met, we can update the candidate
					if depsMet {
						candidate = rel
						candidateDeps = releaseDeps
						preferredResolved = true
					}
				}
			}
		}

		if preferred != "" && !preferredResolved {
			return nil, fmt.Errorf("cannot resolve preferred version %s for %s with compatible dependencies and Factorio version", preferred, item.Name)
		}
		// A pinned release may stay installed while one of its dependencies changes.
		for _, dep := range candidateDeps {
			downloadMods = addDownload(dep, downloadMods)
		}
		//Add candidate to the download list
		if candidate.Version != "0.0.0" && item.installed.Version != candidate.Version {
			downloadMods = addDownload(downloadData{Title: item.Title, Name: item.Name, Filename: candidate.FileName,
				OldFilename: item.installed.Filename, Data: candidate, Version: candidate.Version,
				OldVersion: item.installed.Version, wasDep: wasDep,
			}, downloadMods)
		}

	}

	//Return list of downloads
	return downloadMods, nil
}

func CheckModUpdates(dryRun bool, emitProgress bool, suppressChecking bool, suppressNoUpdates bool) (bool, error) {
	unlock, lockErr := cfg.LockControlResources()
	if lockErr != nil {
		return false, lockErr
	}
	defer unlock()
	opToken := fact.BeginOperation("Mod Updates", "Checking for mod updates.")
	if suppressChecking {
		fact.SuppressPendingOperationAnnouncement(opToken)
	}
	progress := modUpdateProgress{}
	if emitProgress {
		progress = newModUpdateProgress(opToken)
	}
	resetModInfoCache()
	fact.SetModOperationInProgress(true)
	defer func() {
		fact.SetModOperationInProgress(false)
	}()

	// If needed, get Factorio version
	getFactorioVersion()

	// Read all mod.zip files
	modFileList, err := GetModFiles()
	if err != nil {
		return false, err
	}

	// Read mods-list.json
	jsonModList, _ := GetModList() //Ignore error, mods-list.json missing isn't the end of the world.
	// Merge the two lists
	installedMods := MergeModLists(modFileList, jsonModList)
	versionPrefs := modedit.ReadPrefs()

	// Check if we need to proceed
	if len(installedMods) == 0 {
		emsg := "the game has no installed mods to update"
		fact.FailOperation(opToken, "Mod Updates", emsg, glob.COLOR_RED)
		return false, errors.New(emsg)
	}

	// Fetch mod portal data
	modPortalData := []modPortalFullData{}
	checkableMods := 0
	for _, item := range installedMods {
		if !item.Enabled || IsBaseMod(item.Name) {
			continue
		}
		if !IsBaseMod(item.Name) {
			checkableMods++
		}
	}
	checkedMods := 0
	for _, item := range installedMods {
		if !item.Enabled || IsBaseMod(item.Name) {
			continue
		}
		checkedMods++
		progress.emit(formatCheckModProgress(item.Name, checkedMods, checkableMods))
		//cwlog.DoLogCW("Getting portal info: %v", item.Name)
		newInfo, err := DownloadModInfo(item.Name)
		if err != nil {
			cwlog.DoLogCW("NEWCheckModUpdates: DownloadModInfo" + err.Error())
			fact.FailOperation(opToken, "Mod Updates", "Checking mod updates failed: "+err.Error(), glob.COLOR_RED)
			return false, err
		}

		//Save the filename, for dealing with old versions later
		newInfo.filename = item.Filename
		newInfo.installed = item
		modPortalData = append(modPortalData, newInfo)
		//cwlog.DoLogCW("Got portal info: %v", newInfo.Name)
	}

	downloadList, err := resolveDepsWithPrefs(modPortalData, false, 0, nil, versionPrefs, &progress)

	if err != nil {
		cwlog.DoLogCW("NEWCheckModUpdates: resolveDeps: " + err.Error())
		fact.FailOperation(opToken, "Mod Updates", "Resolving mod dependencies failed: "+err.Error(), glob.COLOR_RED)
		return false, err
	}

	err = validateDownloadPlan(installedMods, downloadList)
	if err != nil {
		cwlog.DoLogCW(err.Error())
		fact.FailOperation(opToken, "Mod Updates", err.Error(), glob.COLOR_RED)
		return false, err
	}

	//Dry run ends here
	if dryRun {
		for _, dl := range downloadList {
			cwlog.DoLogCW("%v-%v: %v", dl.Name, dl.Data.Version, dl.Filename)
		}
		fact.CompleteOperation(opToken, "Mod Updates", "Mod update check complete.", glob.COLOR_GREEN)
		return false, nil
	}

	if len(downloadList) > 0 {
		newHist := ModHistoryItem{Name: "Mod update started", InfoItem: true, Date: time.Now()}
		AddModHistory(newHist)
	}
	shortBuf := downloadMods(downloadList)
	requestedDownloads := getDownloadCount(downloadList)
	completedDownloads := getCompletedDownloadCount(downloadList)

	if requestedDownloads > 0 && completedDownloads == requestedDownloads && len(installedMods) > 0 {
		emsg := "Mod updates complete."
		glob.SetUpdateMessage(disc.SmartEditDiscordEmbed(cfg.Local.Channel.ChatChannel, glob.GetUpdateMessage(), "Mod Updates", emsg, glob.COLOR_CYAN))
		if fact.NumPlayersCurrent() > 0 && shortBuf != "" {
			fact.FactChat("Mod updates: " + shortBuf + " (Mods will update on reboot, when server is empty)")
		}
		glob.SetBootMessage(nil)
		fact.CompleteOperation(opToken, "Mod Updates", "Mod updates are ready and will apply on reboot.", glob.COLOR_GREEN)
		return true, nil
	}
	if requestedDownloads > 0 && completedDownloads > 0 {
		emsg := fmt.Sprintf("Only %d of %d mod updates downloaded successfully. Restart canceled.", completedDownloads, requestedDownloads)
		glob.SetBootMessage(nil)
		fact.FailOperation(opToken, "Mod Updates", emsg, glob.COLOR_RED)
		return false, errors.New(emsg)
	}
	if requestedDownloads > 0 {
		emsg := "Mod update downloads failed. Restart canceled."
		glob.SetBootMessage(nil)
		fact.FailOperation(opToken, "Mod Updates", emsg, glob.COLOR_RED)
		return false, errors.New(emsg)
	}

	glob.SetBootMessage(nil)
	if suppressNoUpdates {
		fact.CancelOperation(opToken)
	} else {
		fact.CompleteOperation(opToken, "Mod Updates", "No mod updates available.", glob.COLOR_GREEN)
	}
	return false, errors.New("no mod updates available")
}

func formatCheckModProgress(name string, current, total int) string {
	if total <= 0 {
		total = 0
	}
	if name == "" {
		name = "unknown mod"
	}
	return fmt.Sprintf("Checking %d/%d: %s", current, total, name)
}

func formatResolveDepsProgress(name string, current, total, depth int) string {
	if name == "" {
		name = "unknown mod"
	}
	if total <= 0 {
		return fmt.Sprintf("Deps d%d: %s", depth, name)
	}
	return fmt.Sprintf("Deps %d/%d d%d: %s", current, total, depth, name)
}

type incMod struct {
	Name, Version string
	Deps          []string
}

// validateDownloadPlan checks the actual versions that will coexist after the
// update. Replacements supersede installed metadata, and merged dependency
// downloads must still satisfy every enabled mod, including pinned releases.
func validateDownloadPlan(installed []modZipInfo, downloads []downloadData) error {
	final := map[string]incMod{}
	for _, mod := range installed {
		if mod.Enabled {
			final[mod.Name] = incMod{Name: mod.Name, Version: mod.Version, Deps: mod.Dependencies}
		}
	}
	for _, mod := range downloads {
		final[mod.Name] = incMod{Name: mod.Name, Version: mod.Version, Deps: mod.Data.InfoJSON.Dependencies}
	}
	names := make([]string, 0, len(final))
	for name := range final {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mod := final[name]
		for _, raw := range mod.Deps {
			dep := parseDep(raw)
			version := final[dep.name].Version
			if dep.name == "base" {
				version = fact.FactorioVersion
			}
			present := version != "" && version != "0.0.0"
			if !present && (dep.optional || dep.incompatible) {
				continue
			}
			matches := present
			if present && dep.version != "" {
				var err error
				matches, err = checkVersion(dep.equality, dep.version, version)
				if err != nil {
					return fmt.Errorf("invalid dependency %q for %s: %w", raw, name, err)
				}
			}
			if dep.incompatible {
				if matches {
					return fmt.Errorf("planned mod %s-%s is incompatible with %s-%s", name, mod.Version, dep.name, version)
				}
			} else if !matches {
				return fmt.Errorf("planned mod %s-%s requires %s; selected dependency version is %q", name, mod.Version, raw, version)
			}
		}
	}
	return nil
}

func parseDep(input string) depRequires {
	incompatible, optional := false, false

	input = strings.TrimSpace(input)
	//Mark incompatible
	if strings.Contains(input, "!") {
		incompatible = true
	}
	//Mark optional
	if strings.HasPrefix(input, "?") || strings.HasPrefix(input, "(?)") || strings.HasPrefix(input, "( ? )") {
		optional = true
	}
	//Remove prefixes before processing
	input = strings.TrimPrefix(input, "~")
	input = strings.TrimPrefix(input, "!")
	input = strings.TrimPrefix(input, "?")
	input = strings.TrimPrefix(input, "(?)")
	input = strings.TrimPrefix(input, "( ? )")
	input = strings.TrimSpace(input)

	nameEnd := 0
	versionStart := 0
	// This properly handles malformed dependencies (no spaces) *cough flare stack cough*
	for c, ch := range input {
		if ch == '>' || ch == '<' || ch == '=' {
			if nameEnd == 0 {
				nameEnd = c
			}
			versionStart = c
		}
	}
	if nameEnd == 0 {
		return depRequires{name: input, optional: optional, incompatible: incompatible}
	}
	name := strings.TrimSpace(input[:nameEnd])
	equality := strings.TrimSpace(input[nameEnd : versionStart+1])
	version := strings.TrimSpace(input[versionStart+1:])

	return depRequires{name: name, equality: parseOperator(equality), version: version, optional: optional, incompatible: incompatible}
}

// addDownload checks for duplicate downloads and replaces older entries.
func addDownload(input downloadData, list []downloadData) []downloadData {
	for i, item := range list {
		if item.Name == input.Name {
			newer, err := checkVersion(EO_GREATER, item.Data.Version, input.Data.Version)
			if err != nil {
				cwlog.DoLogCW("addDownload: Unable to parse version")
				return list
			}
			if newer {
				list[i] = input
				if resolveDepsDebug {
					cwlog.DoLogCW("Added newer download: %v-%v", input.Name, input.Version)
				}
			} else if resolveDepsDebug {
				cwlog.DoLogCW("DID NOT ADD download: %v-%v", input.Name, input.Version)
			}
			return list
		}
	}
	if resolveDepsDebug {
		cwlog.DoLogCW("Added download: %v-%v", input.Name, input.Version)
	}
	return append(list, input)
}

// CheckModsForControl preserves manual-update restart scheduling for HTTP.
func CheckModsForControl() (bool, error) {
	updated, err := CheckModUpdates(false, true, false, false)
	if err == nil && updated && fact.GetLifecycleState().Phase != fact.LifecycleStopped {
		err = fact.SubmitLifecycleRequest(fact.Request{Kind: fact.ActionRestartFactorio, Reason: "Rebooting Factorio after mod updates.", WhenEmpty: true})
	}
	return updated, err
}
