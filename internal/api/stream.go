package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mihomoTui/internal/models"
	"net/http"
	"net/url"
	"strings"
)

// contextReader wraps an io.ReadCloser and checks ctx.Done() before every Read call.
// This ensures that a blocking bufio.Scanner.Scan() returns promptly when the
// context is cancelled — even when the remote server keeps the connection open
// without sending data.
type contextReader struct {
	ctx context.Context
	r   io.ReadCloser
}

func (cr *contextReader) Read(p []byte) (int, error) {
	select {
	case <-cr.ctx.Done():
		return 0, cr.ctx.Err()
	default:
	}
	return cr.r.Read(p)
}

func (cr *contextReader) Close() error {
	return cr.r.Close()
}

// StreamMemoryUsage retrieves the current memory usage
func (c *HttpClient) StreamMemoryUsage(ctx context.Context, callback func(*models.MemoryUsage)) error {
	resp, err := c.makeRequestWithContext(ctx, "GET", "/memory", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(&contextReader{ctx: ctx, r: resp.Body})
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var memUsage models.MemoryUsage
		if err := json.Unmarshal([]byte(line), &memUsage); err != nil {
			continue // Skip invalid JSON
		}

		callback(&memUsage)
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}
	return nil
}

// StreamLogs streams real-time logs via chunked transfer
func (c *HttpClient) StreamLogs(ctx context.Context, callback func(*models.Log)) error {
	return c.streamLogsEndpoint(ctx, "/logs", callback)
}

// StreamLogsWithLevel streams real-time logs filtered by level
func (c *HttpClient) StreamLogsWithLevel(ctx context.Context, level string, callback func(*models.Log)) error {
	endpoint := "/logs"
	if level != "" {
		query := url.Values{}
		query.Set("level", level)
		endpoint = fmt.Sprintf("/logs?%s", query.Encode())
	}
	return c.streamLogsEndpoint(ctx, endpoint, callback)
}

func (c *HttpClient) streamLogsEndpoint(ctx context.Context, endpoint string, callback func(*models.Log)) error {
	resp, err := c.makeRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(&contextReader{ctx: ctx, r: resp.Body})
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var log models.Log
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			continue // Skip invalid JSON
		}

		callback(&log)
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}
	return nil
}

// StreamTraffic streams real-time traffic data via chunked transfer
func (c *HttpClient) StreamTraffic(ctx context.Context, callback func(*models.Traffic)) error {
	resp, err := c.makeRequestWithContext(ctx, "GET", "/traffic", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(&contextReader{ctx: ctx, r: resp.Body})
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var traffic models.Traffic
		if err := json.Unmarshal([]byte(line), &traffic); err != nil {
			continue // Skip invalid JSON
		}

		callback(&traffic)
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}
	return nil
}
