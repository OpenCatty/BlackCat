package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

const (
	defaultWebTimeout  = 30 * time.Second
	defaultMaxSize     = 1 << 20 // 1 MB
	webToolName        = "web"
	webToolDescription = "Fetch content from a URL"
)

var webToolParameters = json.RawMessage(`{
	"type": "object",
	"properties": {
		"url": {
			"type": "string",
			"description": "URL to fetch"
		}
	},
	"required": ["url"]
}`)

// privateRanges contains CIDR blocks that are considered private/internal.
var privateRanges []*net.IPNet

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Sprintf("tools: failed to parse CIDR %q: %v", cidr, err))
		}
		privateRanges = append(privateRanges, block)
	}
}

// isPrivateIP checks if an IP falls within any private/internal range.
func isPrivateIP(ip net.IP) bool {
	for _, block := range privateRanges {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// WebTool fetches content from URLs with SSRF protection.
type WebTool struct {
	httpClient *http.Client
	timeout    time.Duration
	maxSize    int
}

// NewWebTool creates a WebTool with default timeout and size limits.
func NewWebTool(timeout time.Duration) *WebTool {
	if timeout <= 0 {
		timeout = defaultWebTimeout
	}
	return &WebTool{
		httpClient: &http.Client{Timeout: timeout},
		timeout:    timeout,
		maxSize:    defaultMaxSize,
	}
}

func (t *WebTool) Name() string                { return webToolName }
func (t *WebTool) Description() string         { return webToolDescription }
func (t *WebTool) Parameters() json.RawMessage { return webToolParameters }

// Execute fetches a URL after validating it against SSRF protections.
func (t *WebTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("web: invalid arguments: %w", err)
	}
	if params.URL == "" {
		return "", fmt.Errorf("web: url is required")
	}

	// Parse URL and extract hostname.
	parsed, err := url.Parse(params.URL)
	if err != nil {
		return "", fmt.Errorf("web: invalid URL: %w", err)
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return "", fmt.Errorf("web: missing hostname in URL")
	}

	// SSRF protection: resolve hostname and check against private ranges.
	ips, err := net.LookupHost(hostname)
	if err != nil {
		return "", fmt.Errorf("web: DNS lookup failed for %q: %w", hostname, err)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil && isPrivateIP(ip) {
			return "", fmt.Errorf("SSRF: private IP blocked")
		}
	}

	// Perform HTTP GET.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return "", fmt.Errorf("web: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("web: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body up to maxSize.
	limited := io.LimitReader(resp.Body, int64(t.maxSize)+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return "", fmt.Errorf("web: read body: %w", err)
	}
	if len(body) > t.maxSize {
		body = body[:t.maxSize]
	}

	return string(body), nil
}
