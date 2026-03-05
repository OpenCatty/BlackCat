package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	httpClient   *http.Client
	timeout      time.Duration
	maxSize      int
	pinchEnabled bool
	pinchBaseURL string
	pinchToken   string
}

// NewWebTool creates a WebTool with default timeout and size limits.
func NewWebTool(timeout time.Duration) *WebTool {
	if timeout <= 0 {
		timeout = defaultWebTimeout
	}
	pinchEnabled := false
	pinchBaseURL := strings.TrimSpace(os.Getenv("BLACKCAT_PINCHTAB_BASE_URL"))
	if pinchBaseURL != "" {
		pinchEnabled = true
	}
	if v := strings.TrimSpace(strings.ToLower(os.Getenv("BLACKCAT_PINCHTAB_ENABLED"))); v == "1" || v == "true" || v == "yes" {
		pinchEnabled = true
	}
	if pinchEnabled && pinchBaseURL == "" {
		pinchBaseURL = "http://127.0.0.1:9867"
	}
	pinchToken := strings.TrimSpace(os.Getenv("BLACKCAT_PINCHTAB_TOKEN"))
	return &WebTool{
		httpClient:   &http.Client{Timeout: timeout},
		timeout:      timeout,
		maxSize:      defaultMaxSize,
		pinchEnabled: pinchEnabled,
		pinchBaseURL: strings.TrimRight(pinchBaseURL, "/"),
		pinchToken:   pinchToken,
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

	if t.pinchEnabled {
		return t.executeViaPinchTab(ctx, params.URL)
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

func (t *WebTool) executeViaPinchTab(ctx context.Context, targetURL string) (string, error) {
	defaultText, defaultErr := t.executeViaPinchTabDefaultFlow(ctx, targetURL)
	if defaultErr == nil {
		if strings.TrimSpace(defaultText) == "" {
			return "", fmt.Errorf("web: pinchtab returned empty text")
		}
		return defaultText, nil
	}

	legacyText, legacyErr := t.executeViaPinchTabInstanceFlow(ctx, targetURL)
	if legacyErr == nil {
		if strings.TrimSpace(legacyText) == "" {
			return "", fmt.Errorf("web: pinchtab returned empty text")
		}
		return legacyText, nil
	}

	return "", fmt.Errorf("web: pinchtab default flow failed: %v; legacy flow failed: %w", defaultErr, legacyErr)
}

func (t *WebTool) executeViaPinchTabDefaultFlow(ctx context.Context, targetURL string) (string, error) {
	body := map[string]string{"url": targetURL}
	buf, _ := json.Marshal(body)

	navResp, err := t.pinchRequest(ctx, http.MethodPost, "/navigate", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	defer navResp.Body.Close()
	if navResp.StatusCode < 200 || navResp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(navResp.Body, 1024))
		return "", fmt.Errorf("navigate status %d: %s", navResp.StatusCode, strings.TrimSpace(string(data)))
	}

	_, _ = io.Copy(io.Discard, io.LimitReader(navResp.Body, 4096))

	textResp, err := t.pinchRequest(ctx, http.MethodGet, "/text", nil)
	if err != nil {
		return "", err
	}
	defer textResp.Body.Close()
	if textResp.StatusCode < 200 || textResp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(textResp.Body, 1024))
		return "", fmt.Errorf("text status %d: %s", textResp.StatusCode, strings.TrimSpace(string(data)))
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(textResp.Body, int64(t.maxSize)+1))
	if err != nil {
		return "", err
	}
	if len(bodyBytes) > t.maxSize {
		bodyBytes = bodyBytes[:t.maxSize]
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(bodyBytes, &payload); err == nil && payload.Text != "" {
		return payload.Text, nil
	}

	return string(bodyBytes), nil
}

func (t *WebTool) executeViaPinchTabInstanceFlow(ctx context.Context, targetURL string) (string, error) {
	instanceID, err := t.pinchStartInstance(ctx)
	if err != nil {
		return "", fmt.Errorf("web: pinchtab start instance: %w", err)
	}
	defer t.pinchStopInstance(context.Background(), instanceID)

	tabID, err := t.pinchOpenTab(ctx, instanceID, targetURL)
	if err != nil {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
		}
		tabID, err = t.pinchOpenTab(ctx, instanceID, targetURL)
		if err != nil {
			return "", fmt.Errorf("web: pinchtab open tab: %w", err)
		}
	}

	text, err := t.pinchReadText(ctx, tabID)
	if err != nil {
		return "", fmt.Errorf("web: pinchtab read text: %w", err)
	}
	return text, nil
}

func (t *WebTool) pinchStartInstance(ctx context.Context) (string, error) {
	body := map[string]string{"mode": "headless"}
	buf, _ := json.Marshal(body)

	resp, err := t.pinchRequest(ctx, http.MethodPost, "/instances/start", bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ID == "" {
		return "", fmt.Errorf("missing instance id")
	}
	return payload.ID, nil
}

func (t *WebTool) pinchOpenTab(ctx context.Context, instanceID, targetURL string) (string, error) {
	body := map[string]string{"url": targetURL}
	buf, _ := json.Marshal(body)

	path := fmt.Sprintf("/instances/%s/tabs/open", instanceID)
	resp, err := t.pinchRequest(ctx, http.MethodPost, path, bytes.NewReader(buf))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload struct {
		ID    string `json:"id"`
		TabID string `json:"tabId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if payload.ID != "" {
		return payload.ID, nil
	}
	if payload.TabID != "" {
		return payload.TabID, nil
	}
	return "", fmt.Errorf("missing tab id")
}

func (t *WebTool) pinchReadText(ctx context.Context, tabID string) (string, error) {
	path := fmt.Sprintf("/tabs/%s/text", tabID)
	resp, err := t.pinchRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(t.maxSize)+1))
	if err != nil {
		return "", err
	}
	if len(body) > t.maxSize {
		body = body[:t.maxSize]
	}

	var payload struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Text != "" {
		return payload.Text, nil
	}

	return string(body), nil
}

func (t *WebTool) pinchStopInstance(ctx context.Context, instanceID string) {
	path := fmt.Sprintf("/instances/%s/stop", instanceID)
	resp, err := t.pinchRequest(ctx, http.MethodPost, path, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
}

func (t *WebTool) pinchRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if strings.TrimSpace(t.pinchBaseURL) == "" {
		return nil, fmt.Errorf("pinchtab base URL is empty")
	}
	req, err := http.NewRequestWithContext(ctx, method, t.pinchBaseURL+path, body)
	if err != nil {
		return nil, err
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	if t.pinchToken != "" {
		req.Header.Set("Authorization", "Bearer "+t.pinchToken)
	}
	return t.httpClient.Do(req)
}
