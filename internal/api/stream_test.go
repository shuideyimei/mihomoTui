package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mihomoTui/internal/models"
)

// TestStreamMemoryUsage_ReceivesData verifies that StreamMemoryUsage reads
// chunked JSON lines and delivers them to the callback.
func TestStreamMemoryUsage_ReceivesData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/memory" {
			t.Errorf("expected /memory, got %s", r.URL.Path)
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `{"inuse": 104857600, "oslimit": 4294967296}`+"\n")
		flusher.Flush()
		fmt.Fprintf(w, `{"inuse": 209715200, "oslimit": 4294967296}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var results []*models.MemoryUsage
	done := make(chan error, 1)

	go func() {
		done <- client.StreamMemoryUsage(ctx, func(m *models.MemoryUsage) {
			results = append(results, m)
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Inuse != 104857600 {
		t.Errorf("expected Inuse=104857600, got %d", results[0].Inuse)
	}
	if results[0].Oslimit != 4294967296 {
		t.Errorf("expected Oslimit=4294967296, got %d", results[0].Oslimit)
	}
	if results[1].Inuse != 209715200 {
		t.Errorf("expected Inuse=209715200, got %d", results[1].Inuse)
	}
	if results[1].Oslimit != 4294967296 {
		t.Errorf("expected Oslimit=4294967296, got %d", results[1].Oslimit)
	}
}

// TestStreamLogs_ReceivesLog verifies that StreamLogs reads chunked log JSON
// lines and delivers them to the callback.
func TestStreamLogs_ReceivesLog(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/logs" {
			t.Errorf("expected /logs, got %s", r.URL.Path)
		}
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `{"type": "info", "payload": "test log message"}`+"\n")
		flusher.Flush()
		fmt.Fprintf(w, `{"type": "warning", "payload": "warning message"}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var results []*models.Log
	done := make(chan error, 1)

	go func() {
		done <- client.StreamLogs(ctx, func(l *models.Log) {
			results = append(results, l)
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(results))
	}
	if results[0].Type != "info" || results[0].Payload != "test log message" {
		t.Errorf("unexpected first log: Type=%q Payload=%q", results[0].Type, results[0].Payload)
	}
	if results[1].Type != "warning" || results[1].Payload != "warning message" {
		t.Errorf("unexpected second log: Type=%q Payload=%q", results[1].Type, results[1].Payload)
	}
}

// TestStreamLogsWithLevel_LevelFiltering verifies that StreamLogsWithLevel
// sends the level query parameter in the request URL.
func TestStreamLogsWithLevel_LevelFiltering(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"type": "warning", "payload": "warn"}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- client.StreamLogsWithLevel(ctx, "warning", func(l *models.Log) {})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if !strings.Contains(capturedPath, "level=warning") {
		t.Errorf("expected level=warning in request path, got %s", capturedPath)
	}
}

// TestStreamLogsWithLevel_EmptyLevel verifies that an empty level string
// results in a plain /logs request with no query parameters.
func TestStreamLogsWithLevel_EmptyLevel(t *testing.T) {
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.String()
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"type": "info", "payload": "hello"}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- client.StreamLogsWithLevel(ctx, "", func(l *models.Log) {})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if strings.Contains(capturedPath, "level=") {
		t.Errorf("expected no level parameter, got %s", capturedPath)
	}
}

func TestStream_ContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, `{"inuse": 100, "oslimit": 200}`+"\n")
		flusher.Flush()

		<-r.Context().Done()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	inCallback := make(chan struct{})
	proceed := make(chan struct{})
	errCh := make(chan error, 1)

	go func() {
		errCh <- client.StreamMemoryUsage(ctx, func(m *models.MemoryUsage) {
			close(inCallback)
			<-proceed
		})
	}()

	select {
	case <-inCallback:
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for first data")
	}

	cancel()
	close(proceed)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error after context cancel, got nil")
		}
		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for stream to stop after cancel")
	}
}

// TestStream_ConnectionError verifies that a non-200 status code returns an
// error immediately without calling the callback.
func TestStream_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error": "internal error"}`)
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()

	err := client.StreamMemoryUsage(ctx, func(m *models.MemoryUsage) {
		t.Error("callback should not be called on error")
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

// TestStream_SkipInvalidJSON verifies that non-JSON lines are silently
// skipped without calling the callback.
func TestStream_SkipInvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)

		// Send an invalid line then a valid one
		fmt.Fprintf(w, "this is not json\n")
		flusher.Flush()
		fmt.Fprintf(w, `{"type": "info", "payload": "valid"}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var calls int
	done := make(chan error, 1)
	go func() {
		done <- client.StreamLogs(ctx, func(l *models.Log) {
			calls++
			if l.Payload != "valid" {
				t.Errorf("expected payload 'valid', got %q", l.Payload)
			}
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if calls != 1 {
		t.Fatalf("expected exactly 1 valid callback call, got %d", calls)
	}
}

// TestStream_EmptyLineSkips verifies that empty lines in the stream are
// skipped without calling the callback.
func TestStream_EmptyLineSkips(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected http.Flusher")
		}
		w.WriteHeader(http.StatusOK)

		fmt.Fprintf(w, "\n")
		flusher.Flush()
		fmt.Fprintf(w, `{"type": "info", "payload": "after blank"}`+"\n")
		flusher.Flush()
	}))
	defer srv.Close()

	client := &HttpClient{
		baseURL:    srv.URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var calls int
	done := make(chan error, 1)
	go func() {
		done <- client.StreamLogs(ctx, func(l *models.Log) {
			calls++
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for stream to complete")
	}

	if calls != 1 {
		t.Fatalf("expected 1 callback call (skipping empty line), got %d", calls)
	}
}
