package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	endpoint  string
	token     string
	userAgent string
	http      *http.Client
}

func New(endpoint, token, userAgent string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		endpoint:  strings.TrimRight(endpoint, "/"),
		token:     token,
		userAgent: userAgent,
		http:      httpClient,
	}
}

type APIError struct {
	Operation string
	Status    int
	Code      string `json:"code"`
	Message   string `json:"message"`
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("garage %s: %s (HTTP %d): %s", e.Operation, e.Code, e.Status, e.Message)
	}
	return fmt.Sprintf("garage %s: HTTP %d: %s", e.Operation, e.Status, e.Message)
}

func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound
}

func (c *Client) get(ctx context.Context, op string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, op, query, nil, out)
}

func (c *Client) post(ctx context.Context, op string, query url.Values, body, out any) error {
	return c.do(ctx, http.MethodPost, op, query, body, out)
}

func (c *Client) do(ctx context.Context, method, op string, query url.Values, body, out any) error {
	u := c.endpoint + "/v2/" + op
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("garage %s: encoding request: %w", op, err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reader)
	if err != nil {
		return fmt.Errorf("garage %s: %w", op, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("garage %s: %w", op, err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("garage %s: reading response: %w", op, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		apiErr := &APIError{Operation: op, Status: resp.StatusCode}
		if json.Unmarshal(data, apiErr) != nil || apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(data))
		}
		apiErr.Operation = op
		apiErr.Status = resp.StatusCode
		return apiErr
	}

	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("garage %s: decoding response: %w", op, err)
	}
	return nil
}
