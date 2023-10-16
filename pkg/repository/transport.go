package repository

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ibice/opa-bundle-github/pkg/log"
)

type loggerRoundTripper struct {
	logger *slog.Logger
	upper  http.RoundTripper
}

func newLoggerRoundTripper(clientName string) *loggerRoundTripper {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	defaultTransport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &loggerRoundTripper{
		logger: log.Logger.With("clientName", "github"),
		upper:  defaultTransport,
	}
}

func (logrt loggerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	logrt.logger.Debug("Request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", req.Header,
	)
	return logrt.upper.RoundTrip(req)
}
