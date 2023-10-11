// Package easyproxy provides a method to quickly create a http.Transport or net.Conn using given proxy details (SOCKS5 or HTTP).
package easyproxy

import (
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/magisterquis/connectproxy"
	"golang.org/x/net/proxy"
)

type Protocol int

const (
	SOCKS5 Protocol = iota // SOCKS5
	HTTP                   // HTTP
)

type ProxyConfig struct {
	Protocol Protocol
	Addr     string
	User     string
	Password string
}

// NewTransport returns a http.Transport using the given proxy details. Leave user/pass blank if not needed.
func NewTransport(c ProxyConfig) (*http.Transport, error) {
	t := &http.Transport{}
	if c.Protocol == HTTP {
		u := &url.URL{
			Scheme: "http",
			Host:   c.Addr,
		}
		if c.User != "" && c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		}
		t.Proxy = http.ProxyURL(u)
		return t, nil
	}
	var auth *proxy.Auth = nil
	if c.User != "" && c.Password != "" {
		auth = &proxy.Auth{User: c.User, Password: c.Password}
	}
	dialer, err := proxy.SOCKS5("tcp", c.Addr, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	t.Dial = dialer.Dial
	return t, nil
}

// NewConn returns a tls.Conn to "addr" using the given proxy details. Leave user/pass blank if not needed.
func NewConn(c ProxyConfig, addr string, tlsConf *tls.Config) (*tls.Conn, error) {
	var proxyDialer proxy.Dialer
	var err error
	if c.Protocol == SOCKS5 {
		var auth *proxy.Auth = nil
		if c.User != "" && c.Password != "" {
			auth = &proxy.Auth{User: c.User, Password: c.Password}
		}
		proxyDialer, err = proxy.SOCKS5("tcp", c.Addr, auth, proxy.Direct)
		if err != nil {
			return nil, err
		}
	} else {
		u := &url.URL{
			Scheme: "http",
			Host:   c.Addr,
		}
		if c.User != "" && c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password)
		}
		proxyDialer, err = connectproxy.New(u, proxy.Direct)
	}

	dialer, err := proxyDialer.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	conn := tls.Client(dialer, tlsConf)
	return conn, nil
}
