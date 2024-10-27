package factUpdater

import (
	"encoding/json"
	"fmt"
)

func quickLatest(info *infoData) (*versionInts, error) {

	FetchLock.Lock()
	defer FetchLock.Unlock()

	urlBuf := "https://factorio.com/api/latest-releases"
	body, _, err := httpGet(info, urlBuf)
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
