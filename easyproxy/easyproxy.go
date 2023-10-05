// Package easyproxy provides a method to quickly create a http.Transport using given proxy details (SOCKS5 or HTTP).
package easyproxy

import (
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

type Protocol int

const (
	SOCKS5 Protocol = iota // SOCKS5
	HTTP                   // HTTP
)

// NewTransport returns a http.Transport using the given proxy details. Leave user/pass blank if not needed.
func NewTransport(p Protocol, addr, user, pass string) (*http.Transport, error) {
	t := &http.Transport{}
	if p == HTTP {
		u := &url.URL{
			Scheme: "http",
			Host:   addr,
		}
		if user != "" && pass != "" {
			u.User = url.UserPassword(user, pass)
		}
		t.Proxy = http.ProxyURL(u)
		return t, nil
	}
	var auth *proxy.Auth = nil
	if user != "" && pass != "" {
		auth = &proxy.Auth{User: user, Password: pass}
	}
	dialer, err := proxy.SOCKS5("tcp", addr, auth, proxy.Direct)
	if err != nil {
		return nil, nil
	}
	t.Dial = dialer.Dial
	return t, nil
}
