package common

import (
	"errors"
	"fmt"
	"log"
	"net/http"

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

type ConfigurableTransport interface {
	// SetTransport sets the http.Transport to use for requests. Can be used to set a proxy.
	SetTransport(t *http.Transport)
}
