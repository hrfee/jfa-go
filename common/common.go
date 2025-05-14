package common

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	lm "github.com/hrfee/jfa-go/logmessages"
)

// TimeoutHandler recovers from an http timeout or panic.
type TimeoutHandler func()

// NewTimeoutHandler returns a new Timeout handler.
func NewTimeoutHandler(name, addr string, noFail bool) TimeoutHandler {
	return func() {
		if r := recover(); r != nil {
			out := fmt.Sprintf(lm.FailedAuth, name, addr, 0, lm.TimedOut)
			if noFail {
				log.Print(out)
			} else {
				log.Fatalf(out)
			}
		}
	}
}

// most 404 errors are from UserNotFound, so this generic error doesn't really need any detail.
type ErrNotFound error

type ErrUnauthorized struct{}

func (err ErrUnauthorized) Error() string {
	return lm.Unauthorized
}

type ErrForbidden struct{}

func (err ErrForbidden) Error() string {
	return lm.Forbidden
}

var (
	NotFound ErrNotFound = errors.New(lm.NotFound)
)

type ErrUnknown struct {
	code int
}

func (err ErrUnknown) Error() string {
	msg := fmt.Sprintf(lm.FailedGenericWithCode, err.code)
	return msg
}

// GenericErr returns an error appropriate to the given HTTP status (or actual error, if given).
func GenericErr(status int, err error) error {
	if err != nil {
		return err
	}
	switch status {
	case 200, 204, 201:
		return nil
	case 401, 400:
		return ErrUnauthorized{}
	case 404:
		return NotFound
	case 403:
		return ErrForbidden{}
	default:
		return ErrUnknown{code: status}
	}
}

func GenericErrFromResponse(resp *http.Response, err error) error {
	if resp == nil {
		return ErrUnknown{code: -2}
	}
	return GenericErr(resp.StatusCode, err)
}

type ConfigurableTransport interface {
	// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
	SetTransport(t *http.Transport)
}

// Stripped down-ish version of rough http request function used in most of the API clients.
func Req(httpClient *http.Client, timeoutHandler TimeoutHandler, mode string, uri string, data any, queryParams url.Values, headers map[string]string, response bool) (string, int, error) {
	var params []byte
	if data != nil {
		params, _ = json.Marshal(data)
	}
	if qp := queryParams.Encode(); qp != "" {
		uri += "?" + qp
	}
	var req *http.Request
	if data != nil {
		req, _ = http.NewRequest(mode, uri, bytes.NewBuffer(params))
	} else {
		req, _ = http.NewRequest(mode, uri, nil)
	}
	req.Header.Add("Content-Type", "application/json")
	for name, value := range headers {
		req.Header.Add(name, value)
	}
	resp, err := httpClient.Do(req)
	if resp == nil {
		return "", 0, err
	}
	err = GenericErr(resp.StatusCode, err)
	if timeoutHandler != nil {
		defer timeoutHandler()
	}
	var responseText string
	defer resp.Body.Close()
	if response || err != nil {
		responseText, err = decodeResp(resp)
		if err != nil {
			return responseText, resp.StatusCode, err
		}
	}
	if err != nil {
		var msg any
		err = json.Unmarshal([]byte(responseText), &msg)
		if err != nil {
			return responseText, resp.StatusCode, err
		}
		if msg != nil {
			err = fmt.Errorf("got %d: %+v", resp.StatusCode, msg)
		}
		return responseText, resp.StatusCode, err
	}
	return responseText, resp.StatusCode, err
}

func decodeResp(resp *http.Response) (string, error) {
	var out io.Reader
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		out, _ = gzip.NewReader(resp.Body)
	default:
		out = resp.Body
	}
	buf := new(strings.Builder)
	_, err := io.Copy(buf, out)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
