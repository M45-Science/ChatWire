package factUpdater

import (
	"ChatWire/cfg"
	"encoding/json"
	"fmt"
)

func DoQuickLatest() error {
	info := &infoData{xreleases: cfg.Local.Options.ExpUpdates}
	err := getFactorioVersion(info)
	if err != nil {
		return fmt.Errorf("DoQuickLatest: unable to get factorio version: %v", err.Error())
	}

	newVersion, err := quickLatest(info)
	if err != nil {
		return fmt.Errorf("DoQuickLatest: %v", err.Error())
	}

	if isVersionNewerThan(*newVersion, info.vInt) {
		err := fullPackage(info)
		if err != nil {
			return fmt.Errorf("DoQuickLatest: fullPackage: %v", err.Error())
		}

	}

	return nil
}

func quickLatest(info *infoData) (*versionInts, error) {

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
	if info.xreleases {
		return &tempList.Experimental.HeadlessInt, err
	} else {
		return &tempList.Stable.HeadlessInt, err
	}
}
