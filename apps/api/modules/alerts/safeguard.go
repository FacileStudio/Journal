package alerts

import (
	stderrors "errors"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

var (
	errBlockedWebhookTarget = stderrors.New("webhook target resolves to a blocked address")
	errWebhookRedirect      = stderrors.New("webhook redirects are not allowed")
)

func guardedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return errBlockedWebhookTarget
			}
			ip := net.ParseIP(host)
			if ip == nil || isBlockedIP(ip) {
				return errBlockedWebhookTarget
			}
			return nil
		},
	}
	return &http.Client{
		Timeout:       timeout,
		Transport:     &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: refuseRedirect,
	}
}

func trustedClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: timeout, KeepAlive: 30 * time.Second}
	return &http.Client{
		Timeout:       timeout,
		Transport:     &http.Transport{DialContext: dialer.DialContext},
		CheckRedirect: refuseRedirect,
	}
}

func refuseRedirect(_ *http.Request, _ []*http.Request) error {
	return errWebhookRedirect
}

func isBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsPrivate() {
		return true
	}
	if v4 := ip.To4(); v4 != nil && v4[0] == 169 && v4[1] == 254 {
		return true
	}
	return false
}

func hostAllowed(host string, allowed []string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(host, strings.TrimSpace(candidate)) {
			return true
		}
	}
	return false
}
