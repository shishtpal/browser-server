package mcp

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

func validateRemoteURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil {
		return errors.New("url is invalid")
	}
	host := strings.ToLower(u.Hostname())
	localhost := host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()
	if u.Scheme != "https" && !(u.Scheme == "http" && localhost) {
		return errors.New("url must use HTTPS except for localhost")
	}
	return nil
}
