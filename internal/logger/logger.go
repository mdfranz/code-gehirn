package logger

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/httputil"
)

const maxBodyBytes = 2000

// Init opens path for appending, sets it as the slog default, and installs a
// logging HTTP transport so every outbound request/response is recorded.
//
// LangChainGo uses httputil.DefaultClient (not http.DefaultClient), so we
// replace that. Our own EnsureCollection call uses the same client via
// store.HTTPClient.
func Init(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Wrap LangChain's default client so all provider + Qdrant traffic is logged.
	// httputil.DefaultTransport adds the User-Agent; we wrap it to log around it.
	httputil.DefaultClient = &http.Client{
		Transport: &loggingTransport{wrapped: httputil.DefaultTransport},
	}
	// Also expose the client so store.EnsureCollection can reuse it.
	HTTPClient = httputil.DefaultClient
	return nil
}

// HTTPClient is set by Init and should be used for any direct HTTP calls
// (e.g. store.EnsureCollection) so they pass through the logging transport.
var HTTPClient = http.DefaultClient

// loggingTransport logs every HTTP request and response body to slog.
type loggingTransport struct {
	wrapped http.RoundTripper
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqSnippet string
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewReader(raw))
		reqSnippet = snippet(raw)
	}
	slog.Info("http request",
		"method", req.Method,
		"url", req.URL.String(),
		"body", reqSnippet,
	)

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		slog.Error("http response error",
			"method", req.Method,
			"url", req.URL.String(),
			"error", err,
		)
		return nil, err
	}

	var respSnippet string
	if resp.Body != nil {
		raw, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		respSnippet = snippet(raw)
	}
	slog.Info("http response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"body", respSnippet,
	)

	return resp, nil
}

func snippet(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	if len(b) <= maxBodyBytes {
		return string(b)
	}
	return string(b[:maxBodyBytes]) + " …[truncated]"
}
