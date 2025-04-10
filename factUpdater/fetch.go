package factUpdater

import (
	"ChatWire/constants"
	"ChatWire/glob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const httpDownloadTimeout = time.Minute * 15
const httpGetTimeout = time.Second * 30

var HTTPLock sync.Mutex

func HttpGet(noproxy bool, input string, quick bool) ([]byte, string, error) {
	if noproxy {
		HTTPLock.Lock()
		time.Sleep(time.Millisecond * 200)
		defer HTTPLock.Unlock()
	}

	//Change timeout based on request type
	timeout := httpDownloadTimeout
	if quick {
		timeout = httpGetTimeout
	}

	// Set timeout
	hClient := http.Client{
		Timeout: timeout,
	}

	//Use proxy if provided
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

	//Header
	req.Header.Set("User-Agent", constants.ProgName+"-"+constants.Version)

	//Get response
	res, err := hClient.Do(req)
	if err != nil {
		return nil, "", errors.New("failed to get response: " + err.Error())
	}

	//Check status code
	if res.StatusCode != 200 {
		return nil, "", fmt.Errorf("http status error: %v", res.StatusCode)
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

	//Check data length
	if *glob.ProxyURL != "" {
		if res.ContentLength > 0 {
			if len(body) != int(res.ContentLength) {
				return nil, "", errors.New("data ended early")
			}
		} else {
			return nil, "", errors.New("proxy did not supply content length")
		}
	} else {
		if res.ContentLength > 0 {
			if len(body) != int(res.ContentLength) {
				return nil, "", errors.New("data ended early")
			}
		} else if res.ContentLength != -1 {
			return nil, "", errors.New("content length did not match")
		}
	}

	realurl := res.Request.URL.String()
	parts := strings.Split(realurl, "/")
	query := parts[len(parts)-1]
	parts = strings.Split(query, "?")
	return body, parts[0], nil
}
