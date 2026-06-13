package subscription

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"time"

	"mihomoTui/internal/parser"
	"mihomoTui/internal/types"
)

const maxBodySize = 50 * 1024 * 1024 // 50 MB max download size

var blockedRemotePrefixes = []netip.Prefix{
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001:db8::/32"),
}

type Manager struct {
	httpClient *http.Client
}

func NewManager() *Manager {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.DialContext = safeDialContext
	return &Manager{
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("重定向次数过多")
				}
				return validateRemoteURL(req.URL)
			},
		},
	}
}

func (m *Manager) ImportFromURL(ctx context.Context, subURL string) (*types.ConfigDocument, error) {
	if subURL == "" {
		return nil, fmt.Errorf("URL is required")
	}

	parsed, err := url.ParseRequestURI(subURL)
	if err != nil {
		return nil, fmt.Errorf("无效的 URL: %w", err)
	}
	if err := validateRemoteURL(parsed); err != nil {
		return nil, err
	}

	resp, err := m.downloadWithRetry(ctx, subURL, nil, 3)
	if err != nil {
		return nil, fmt.Errorf("下载失败: %w", err)
	}
	defer resp.Body.Close()

	// Limit response body to prevent OOM from large responses
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if len(body) > maxBodySize {
		return nil, fmt.Errorf("响应体过大 (>50MB)")
	}

	doc, err := parser.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("解析失败: %w", err)
	}

	return doc, nil
}

func (m *Manager) ImportFromLocal(filePath string) (*types.ConfigDocument, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if len(data) > maxBodySize {
		return nil, fmt.Errorf("文件过大 (>50MB)")
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

func validateRemoteURL(parsed *url.URL) error {
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("不支持的 URL 协议: %s (仅支持 http/https)", parsed.Scheme)
	}
	host := parsed.Hostname()
	if host == "" {
		return fmt.Errorf("URL 缺少主机名")
	}
	if ip, err := netip.ParseAddr(host); err == nil && !isPublicIP(ip) {
		return fmt.Errorf("URL 指向内网或保留地址，已被拒绝: %s", host)
	}
	return nil
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("解析目标地址失败: %w", err)
	}

	var addresses []netip.Addr
	if ip, parseErr := netip.ParseAddr(host); parseErr == nil {
		addresses = []netip.Addr{ip}
	} else {
		addresses, err = net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("解析主机名失败: %w", err)
		}
	}

	dialer := net.Dialer{}
	for _, ip := range addresses {
		if !isPublicIP(ip) {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("目标主机仅解析到内网或保留地址")
}

func isPublicIP(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() ||
		ip.IsPrivate() ||
		ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() {
		return false
	}
	for _, prefix := range blockedRemotePrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return ip.IsGlobalUnicast()
}
