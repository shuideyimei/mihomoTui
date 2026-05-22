package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"mihomoTui/internal/models"
	"net/http"
	"strings"
)

// GetMemoryUsage retrieves the current memory usage
func (c *HttpClient) StreamMemoryUsage(ctx context.Context, callback func(*models.MemoryUsage)) error {
	resp, err := c.makeRequest("GET", "/memory", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
	}

	return scanner.Err()
}

// StreamLogs streams real-time logs via chunked transfer
func (c *HttpClient) StreamLogs(ctx context.Context, callback func(*models.Log)) error {
	resp, err := c.makeRequest("GET", "/logs", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
	}

	return scanner.Err()
}

// StreamTraffic streams real-time traffic data via chunked transfer
func (c *HttpClient) StreamTraffic(ctx context.Context, callback func(*models.Traffic)) error {
	resp, err := c.makeRequest("GET", "/traffic", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
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
	}

	return scanner.Err()
}
