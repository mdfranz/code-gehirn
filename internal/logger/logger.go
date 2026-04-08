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

var appLogFile *os.File

// LogFile returns the open app log file handle (set by Init).
// Returns nil if Init has not been called.
func LogFile() *os.File {
	return appLogFile
}

// Init opens appPath and apiPath for appending.
// Application events (startup, errors, lifecycle) go to appPath.
// HTTP request/response traffic (LLM calls, Qdrant calls) goes to apiPath.
//
// LangChainGo uses httputil.DefaultClient (not http.DefaultClient), so we
// replace that. Our own EnsureCollection call uses the same client via
// store.HTTPClient.
func Init(appPath, apiPath string) error {
	if err := os.MkdirAll(filepath.Dir(appPath), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(apiPath), 0755); err != nil {
		return err
	}

	af, err := os.OpenFile(appPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	appLogFile = af

	apif, err := os.OpenFile(apiPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		af.Close()
		return err
	}

	// Application logger — default slog for lifecycle events, errors, etc.
	slog.SetDefault(slog.New(slog.NewTextHandler(af, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	// Dedicated API traffic logger — only HTTP request/response bodies.
	apiLogger := slog.New(slog.NewTextHandler(apif, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Wrap LangChain's default client so all provider + Qdrant traffic is logged.
	httputil.DefaultClient = &http.Client{
		Transport: &loggingTransport{wrapped: httputil.DefaultTransport, logger: apiLogger},
	}
	HTTPClient = httputil.DefaultClient

	slog.Info("logger started", "app_file", appPath, "api_file", apiPath)
	return nil
}

// HTTPClient is set by Init and should be used for any direct HTTP calls
// (e.g. store.EnsureCollection) so they pass through the logging transport.
var HTTPClient = http.DefaultClient

// loggingTransport logs every HTTP request and response body to its dedicated logger.
type loggingTransport struct {
	wrapped http.RoundTripper
	logger  *slog.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	reqSnippet := requestSnippet(req)
	t.logger.Info("http request",
		"method", req.Method,
		"url", req.URL.String(),
		"body", reqSnippet,
	)

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		t.logger.Error("http response error",
			"method", req.Method,
			"url", req.URL.String(),
			"error", err,
		)
		return nil, err
	}

	respSnippet := responseSnippet(resp)
	t.logger.Info("http response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.StatusCode,
		"body", respSnippet,
	)

	return resp, nil
}

func requestSnippet(req *http.Request) string {
	if req == nil || req.Body == nil {
		return ""
	}

	// Prefer cloning from GetBody so we don't touch the real request stream.
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err == nil {
			defer rc.Close()
			return readSnippet(rc)
		}
	}

	// Fallback: peek only a bounded prefix and stitch it back in front.
	peek, body, err := peekBody(req.Body)
	if err == nil {
		req.Body = body
		return snippet(peek, len(peek) > maxBodyBytes)
	}
	return ""
}

func responseSnippet(resp *http.Response) string {
	if resp == nil || resp.Body == nil {
		return ""
	}
	peek, body, err := peekBody(resp.Body)
	if err != nil {
		return ""
	}
	resp.Body = body
	return snippet(peek, len(peek) > maxBodyBytes)
}

func readSnippet(rc io.Reader) string {
	b, err := io.ReadAll(io.LimitReader(rc, maxBodyBytes+1))
	if err != nil {
		return ""
	}
	return snippet(b, len(b) > maxBodyBytes)
}

func peekBody(rc io.ReadCloser) ([]byte, io.ReadCloser, error) {
	b, err := io.ReadAll(io.LimitReader(rc, maxBodyBytes+1))
	if err != nil {
		return nil, rc, err
	}
	return b, io.NopCloser(io.MultiReader(bytes.NewReader(b), rc)), nil
}

func snippet(b []byte, truncated bool) string {
	if len(b) == 0 {
		return ""
	}
	if truncated {
		if len(b) > maxBodyBytes {
			b = b[:maxBodyBytes]
		}
		return string(b) + " …[truncated]"
	}
	return string(b)
}
