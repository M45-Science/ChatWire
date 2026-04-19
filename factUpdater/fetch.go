package factUpdater

import (
	"ChatWire/constants"
	"ChatWire/cwlog"
	"ChatWire/glob"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const httpDownloadTimeout = time.Minute * 15
const httpGetTimeout = time.Second * 30
const httpStatusBodyLimit = 256
const httpMaxRetries = 2

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

	var res *http.Response
	var err error
	for attempt := 0; attempt <= httpMaxRetries; attempt++ {
		//HTTP GET
		req, reqErr := http.NewRequest(http.MethodGet, URL, nil)
		if reqErr != nil {
			return nil, "", errors.New("get failed: " + reqErr.Error())
		}

		//Header
		req.Header.Set("User-Agent", constants.ProgName+"-"+constants.Version)

		//Get response
		res, err = hClient.Do(req)
		if err != nil {
			return nil, "", errors.New("failed to get response: " + err.Error())
		}

		if res.StatusCode == http.StatusTooManyRequests && attempt < httpMaxRetries {
			retryWait := retryAfterDelay(res, attempt)
			if res.Body != nil {
				if cerr := res.Body.Close(); cerr != nil {
					cwlog.DoLogCW("fetch: failed to close body after 429: %v", cerr)
				}
			}
			cwlog.DoLogCW("HttpGet: received 429 for %s, retrying in %v", input, retryWait)
			time.Sleep(retryWait)
			continue
		}

		break
	}

	//Check status code
	if res.StatusCode != http.StatusOK {
		err = formatHTTPStatusError(res)
		if res.Body != nil {
			if cerr := res.Body.Close(); cerr != nil {
				cwlog.DoLogCW("fetch: failed to close body after status error: %v", cerr)
			}
		}
		return nil, "", err
	}

	//Close once complete, if valid
	if res.Body != nil {
		defer func() {
			if err := res.Body.Close(); err != nil {
				cwlog.DoLogCW("fetch: failed to close body: %v", err)
			}
		}()
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

func retryAfterDelay(res *http.Response, attempt int) time.Duration {
	if res != nil {
		header := strings.TrimSpace(res.Header.Get("Retry-After"))
		if header != "" {
			if secs, err := strconv.Atoi(header); err == nil && secs >= 0 {
				return time.Duration(secs) * time.Second
			}
			if t, err := http.ParseTime(header); err == nil {
				wait := time.Until(t)
				if wait > 0 {
					return wait
				}
			}
		}
	}

	wait := time.Second * time.Duration(attempt+1)
	if wait < time.Second {
		return time.Second
	}
	return wait
}

func formatHTTPStatusError(res *http.Response) error {
	if res == nil {
		return errors.New("http status error: no response")
	}

	msg := fmt.Sprintf("http status error: %d %s", res.StatusCode, http.StatusText(res.StatusCode))
	if res.Body == nil {
		return errors.New(msg)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, httpStatusBodyLimit))
	if err != nil {
		return errors.New(msg)
	}

	snippet := strings.TrimSpace(string(body))
	if snippet == "" {
		return errors.New(msg)
	}

	snippet = strings.ReplaceAll(snippet, "\n", " ")
	snippet = strings.ReplaceAll(snippet, "\r", " ")
	return fmt.Errorf("%s: %s", msg, snippet)
}
