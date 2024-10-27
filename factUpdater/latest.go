package factUpdater

import (
	"ChatWire/cfg"
	"ChatWire/fact"
	"encoding/json"
	"fmt"
)

func DoQuickLatest(force bool) (*InfoData, string, bool, bool) {
	info := &InfoData{Xreleases: cfg.Local.Options.ExpUpdates, Build: "headless", Distro: "linux64"}

	if force {
		newVersion, err := quickLatest(info)
		if err != nil {
			return info, fmt.Sprintf("quickLatest: %v", err.Error()), true, false
		}
		info.VersInt = *newVersion
		err = fullPackage(info)
		if err != nil {
			return info, fmt.Sprintf("DoQuickLatest: fullPackage: %v", err.Error()), true, false
		}
		return info, fmt.Sprintf("Factorio %v installed.", newVersion.IntToString()), false, false
	}

	err := getFactorioVersion(info)
	if err != nil {
		return info, fmt.Sprintf("Unable to get factorio version: %v", err.Error()), true, false
	}
	oldVersion := info.VersInt

	newVersion, err := quickLatest(info)
	if err != nil {
		return info, fmt.Sprintf("quickLatest: %v", err.Error()), true, false
	}
	fact.NewVersion = newVersion.IntToString()

	if isVersionNewerThan(*newVersion, oldVersion) || force {
		info.VersInt = *newVersion
		err := fullPackage(info)
		if err != nil {
			return info, fmt.Sprintf("DoQuickLatest: fullPackage: %v", err.Error()), true, false
		}
		return info, fmt.Sprintf("Factorio upgraded from %v to %v.", oldVersion.IntToString(), newVersion.IntToString()), false, false
	} else if isVersionEqual(*newVersion, info.VersInt) {
		return info, "Factorio is up-to-date.", false, true
	} else {
		return info, "Latest version is older, use force option if you want to downgrade.\nWARNING: downgrading will likely break existing maps/saves.", false, true
	}
}

func quickLatest(info *InfoData) (*versionInts, error) {

	FetchLock.Lock()
	defer FetchLock.Unlock()

	body, _, err := httpGet(latestReleaseURL)
	if err != nil {
		return nil, fmt.Errorf("downloading release list failed: %v", err.Error())
	}

	//Unmarshal into temporary list
	tempList := &getLatestData{}
	err = json.Unmarshal(body, &tempList)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal json data: %v", err.Error())
	}

	parseStringsLatest(tempList)
	if info.Xreleases {
		return &tempList.Experimental.HeadlessInt, err
	} else {
		return &tempList.Stable.HeadlessInt, err
	}
}