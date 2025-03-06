package factUpdater

import (
	"ChatWire/constants"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var FetchLock sync.Mutex

const httpDownloadTimeout = time.Minute * 15
const httpGetTimeout = time.Second * 30

func HttpGet(url string, quick bool) ([]byte, string, error) {

	timeout := httpDownloadTimeout
	if quick {
		timeout = httpGetTimeout
	}

	// Set timeout
	hClient := http.Client{
		Timeout: timeout,
	}
	//HTTP GET
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("get failed: %v", err.Error())
	}

	req.Header.Set("User-Agent", constants.ProgName+"-"+constants.Version)

	//Get response
	res, err := hClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get response: %v", err.Error())
	}

	//Close once complete, if valid
	if res.Body != nil {
		defer res.Body.Close()
	}

	//Read all
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read response body: %v", err.Error())
	}

	realurl := res.Request.URL.String()
	parts := strings.Split(realurl, "/")
	query := parts[len(parts)-1]
	parts = strings.Split(query, "?")
	return body, parts[0], nil
}
