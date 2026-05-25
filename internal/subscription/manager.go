package subscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"mihomoTui/internal/parser"
	"mihomoTui/internal/types"
)

type Manager struct {
	httpClient *http.Client
}

func NewManager() *Manager {
	return &Manager{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *Manager) ImportFromURL(ctx context.Context, subURL string) (*types.ConfigDocument, error) {
	if subURL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	resp, err := m.downloadWithRetry(ctx, subURL, nil, 3)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	doc, err := parser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	return doc, nil
}

func (m *Manager) ImportFromLocal(filePath string) (*types.ConfigDocument, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	doc, err := parser.Parse(data)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	return doc, nil
}

func (m *Manager) downloadWithRetry(ctx context.Context, subURL string, extraHeaders map[string]string, maxRetry int) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetry; attempt++ {
		if attempt > 0 {
			wait := time.Duration(1<<(attempt-1)) * time.Second
			if wait > 30*time.Second {
				wait = 30 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		for k, v := range extraHeaders {
			req.Header.Set(k, v)
		}

		resp, err := m.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			lastErr = fmt.Errorf("服务器错误 HTTP %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotModified {
			return resp, nil
		}

		resp.Body.Close()
		return nil, fmt.Errorf("下载返回状态码 %d", resp.StatusCode)
	}

	return nil, fmt.Errorf("下载失败，已重试 %d 次: %w", maxRetry, lastErr)
}
