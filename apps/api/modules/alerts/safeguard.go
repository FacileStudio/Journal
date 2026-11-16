package alerts

import (
	stderrors "errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/netguard"
)

var errWebhookRedirect = stderrors.New("webhook redirects are not allowed")

func guardedClient(timeout time.Duration) *http.Client {
	return (&netguard.Guard{}).Client(timeout)
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
