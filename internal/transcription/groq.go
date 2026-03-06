// Package transcription provides voice-to-text transcription via Groq API.
package transcription

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"
)

const (
	groqTranscribeURL    = "https://api.groq.com/openai/v1/audio/transcriptions"
	defaultModel         = "whisper-large-v3-turbo"
	defaultMaxFileSizeMB = 25
	defaultTimeout       = 60 * time.Second
)

// ErrFileTooLarge is returned when the audio file exceeds the configured size limit.
var ErrFileTooLarge = errors.New("audio file exceeds maximum allowed size")

// GroqClient transcribes audio using the Groq Whisper API.
type GroqClient struct {
	apiKey       string
	model        string
	maxFileSizeB int64
	httpClient   *http.Client
}

// Option configures a GroqClient.
type Option func(*GroqClient)

// WithModel sets the Whisper model (default: whisper-large-v3-turbo).
func WithModel(model string) Option {
	return func(c *GroqClient) {
		if model != "" {
			c.model = model
		}
	}
}

// WithMaxFileSizeMB sets the maximum audio file size in MB (default: 25).
func WithMaxFileSizeMB(mb int) Option {
	return func(c *GroqClient) {
		if mb > 0 {
			c.maxFileSizeB = int64(mb) * 1024 * 1024
		}
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *GroqClient) {
		c.httpClient = client
	}
}

// NewGroqClient creates a new Groq transcription client.
func NewGroqClient(apiKey string, opts ...Option) *GroqClient {
	c := &GroqClient{
		apiKey:       apiKey,
		model:        defaultModel,
		maxFileSizeB: defaultMaxFileSizeMB * 1024 * 1024,
		httpClient:   &http.Client{Timeout: defaultTimeout},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// transcribeResponse is the JSON response from Groq transcription endpoint.
type transcribeResponse struct {
	Text  string `json:"text"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// TranscribeFile transcribes audio from raw bytes. filename is used to determine MIME type.
func (c *GroqClient) TranscribeFile(ctx context.Context, filename string, audio []byte) (string, error) {
	if int64(len(audio)) > c.maxFileSizeB {
		return "", fmt.Errorf("%w: %d bytes (max %d)", ErrFileTooLarge, len(audio), c.maxFileSizeB)
	}

	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	// Add model field
	if err := w.WriteField("model", c.model); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}

	// Add audio file
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".ogg"
	}
	part, err := w.CreateFormFile("file", "audio"+ext)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(audio); err != nil {
		return "", fmt.Errorf("write audio data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	return c.doRequest(ctx, w.FormDataContentType(), &body)
}

// TranscribeURL transcribes audio from a URL (Groq fetches the file directly).
func (c *GroqClient) TranscribeURL(ctx context.Context, audioURL string) (string, error) {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)

	if err := w.WriteField("model", c.model); err != nil {
		return "", fmt.Errorf("write model field: %w", err)
	}
	if err := w.WriteField("url", audioURL); err != nil {
		return "", fmt.Errorf("write url field: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	return c.doRequest(ctx, w.FormDataContentType(), &body)
}

// doRequest sends the multipart request to Groq and parses the response.
func (c *GroqClient) doRequest(ctx context.Context, contentType string, body io.Reader) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, groqTranscribeURL, body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("groq request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var result transcribeResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		if result.Error != nil {
			msg = result.Error.Message
		}
		return "", fmt.Errorf("groq API error: %s", msg)
	}

	if result.Error != nil {
		return "", fmt.Errorf("groq error: %s", result.Error.Message)
	}

	return result.Text, nil
}
