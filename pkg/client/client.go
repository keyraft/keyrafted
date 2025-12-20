package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client is the Keyraft Go SDK client
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// Config holds client configuration
type Config struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// NewClient creates a new Keyraft client
func NewClient(config Config) *Client {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Client{
		baseURL: config.BaseURL,
		token:   config.Token,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Entry represents a key-value entry
type Entry struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Type      string            `json:"type"`
	Version   int64             `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Version represents a historical version
type Version struct {
	Namespace string            `json:"namespace"`
	Key       string            `json:"key"`
	Value     string            `json:"value"`
	Type      string            `json:"type"`
	Version   int64             `json:"version"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Set stores a config value
func (c *Client) Set(namespace, key, value string, metadata map[string]string) (*Entry, error) {
	return c.set(namespace, key, value, "config", metadata)
}

// SetSecret stores a secret value (encrypted)
func (c *Client) SetSecret(namespace, key, value string, metadata map[string]string) (*Entry, error) {
	return c.set(namespace, key, value, "secret", metadata)
}

func (c *Client) set(namespace, key, value, entryType string, metadata map[string]string) (*Entry, error) {
	url := fmt.Sprintf("%s/v1/kv/%s/%s", c.baseURL, namespace, key)

	reqBody := map[string]interface{}{
		"value": value,
		"type":  entryType,
	}
	if metadata != nil {
		reqBody["metadata"] = metadata
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var entry Entry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// Get retrieves the latest version of a key
func (c *Client) Get(namespace, key string) (*Entry, error) {
	url := fmt.Sprintf("%s/v1/kv/%s/%s", c.baseURL, namespace, key)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var entry Entry
	if err := json.NewDecoder(resp.Body).Decode(&entry); err != nil {
		return nil, err
	}

	return &entry, nil
}

// GetVersion retrieves a specific version of a key
func (c *Client) GetVersion(namespace, key string, version int64) (*Version, error) {
	url := fmt.Sprintf("%s/v1/kv/%s/%s?version=%d", c.baseURL, namespace, key, version)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var ver Version
	if err := json.NewDecoder(resp.Body).Decode(&ver); err != nil {
		return nil, err
	}

	return &ver, nil
}

// Delete removes a key
func (c *Client) Delete(namespace, key string) error {
	url := fmt.Sprintf("%s/v1/kv/%s/%s", c.baseURL, namespace, key)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	return nil
}

// List retrieves all keys in a namespace
func (c *Client) List(namespace string) ([]*Entry, error) {
	url := fmt.Sprintf("%s/v1/kv/%s", c.baseURL, namespace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var result struct {
		Keys []*Entry `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Keys, nil
}

// Watch watches for changes in a namespace (single long-poll request)
func (c *Client) Watch(namespace string, timeout time.Duration) (*WatchEvent, error) {
	url := fmt.Sprintf("%s/v1/watch/%s?timeout=%s", c.baseURL, namespace, timeout.String())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var event WatchEvent
	if err := json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, err
	}

	return &event, nil
}

// WatchEvent represents a change event
type WatchEvent struct {
	Action    string    `json:"action"`
	Namespace string    `json:"namespace"`
	Key       string    `json:"key"`
	Entry     *Entry    `json:"entry,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Timeout   bool      `json:"timeout,omitempty"`
}

// WatchStream streams events using Server-Sent Events (SSE)
// Returns a channel that receives events and a function to close the stream
func (c *Client) WatchStream(namespace string) (<-chan *WatchEvent, func(), error) {
	url := fmt.Sprintf("%s/v1/watch/%s?stream=true", c.baseURL, namespace)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/event-stream")

	// Use a client without timeout for streaming
	streamClient := &http.Client{
		Timeout: 0, // No timeout for streaming
	}

	resp, err := streamClient.Do(req)
	if err != nil {
		return nil, nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	events := make(chan *WatchEvent, 10)
	done := make(chan struct{})

		go func() {
			defer resp.Body.Close()
			defer close(events)

			scanner := bufio.NewScanner(resp.Body)
			var currentData strings.Builder

			for scanner.Scan() {
				line := scanner.Text()

				// Empty line indicates end of event
				if line == "" {
					if currentData.Len() > 0 {
						var event WatchEvent
						if err := json.Unmarshal([]byte(currentData.String()), &event); err == nil {
							select {
							case events <- &event:
							case <-done:
								return
							}
						}
						currentData.Reset()
					}
					continue
				}

				// Parse SSE format: "event: <type>" or "data: <json>"
				if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					currentData.WriteString(data)
				}
				// Note: We ignore "event:" lines for now, but could use them for event type filtering
			}

			if err := scanner.Err(); err != nil {
				// Stream ended or error occurred
				return
			}
		}()

	closeFn := func() {
		close(done)
		resp.Body.Close()
	}

	return events, closeFn, nil
}

// Health checks server health
func (c *Client) Health() (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/v1/health", c.baseURL)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
