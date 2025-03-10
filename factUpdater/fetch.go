package factUpdater

import (
	"ChatWire/constants"
	"ChatWire/glob"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const httpDownloadTimeout = time.Minute * 15
const httpGetTimeout = time.Second * 30

func HttpGet(noproxy bool, input string, quick bool) ([]byte, string, error) {

	timeout := httpDownloadTimeout
	if quick {
		timeout = httpGetTimeout
	}

	// Set timeout
	hClient := http.Client{
		Timeout: timeout,
	}

	var URL string
	if *glob.ProxyURL != "" && !noproxy {
		proxy := strings.TrimSuffix(*glob.ProxyURL, "/")
		URL = proxy + "/" + url.PathEscape(input)
	} else {
		URL = input
	}

	//HTTP GET
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, "", errors.New("get failed: " + err.Error())
	}

	req.Header.Set("User-Agent", constants.ProgName+"-"+constants.Version)

	//Get response
	res, err := hClient.Do(req)
	if err != nil {
		return nil, "", errors.New("failed to get response: " + err.Error())
	}

	//Close once complete, if valid
	if res.Body != nil {
		defer res.Body.Close()
	}

	//Read all
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, "", errors.New("unable to read response body: " + err.Error())
	}

	realurl := res.Request.URL.String()
	parts := strings.Split(realurl, "/")
	query := parts[len(parts)-1]
	parts = strings.Split(query, "?")
	return body, parts[0], nil
}
