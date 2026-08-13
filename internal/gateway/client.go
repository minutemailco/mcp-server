// Package gateway provides a minimal HTTP client for the MinuteMail
// api-gateway. The MCP server forwards the caller's Bearer API key verbatim;
// the gateway remains the single enforcement point for authentication,
// scopes and quotas.
package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Response is the outcome of a gateway call.
type Response struct {
	Status      int
	ContentType string
	Body        []byte
	Header      http.Header
}

// OK reports whether the gateway returned a 2xx status.
func (r *Response) OK() bool {
	return r.Status >= 200 && r.Status < 300
}

// Client calls the MinuteMail api-gateway.
type Client struct {
	base string
	hc   *http.Client
}

// New builds a gateway client for the given base URL (e.g.
// http://mm-api-gateway:80).
func New(base string) *Client {
	return &Client{
		base: base,
		hc: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// FilePart is one attachment in a multipart mail upload.
type FilePart struct {
	Field       string
	Filename    string
	ContentType string
	Data        []byte
}

// Do performs a JSON request against the gateway. A nil body sends no
// payload; bearer must be the caller's API key ("mmak_...") without the
// "Bearer " prefix.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, bearer string) (*Response, error) {
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	return c.do(req)
}

// DoMultipart performs a multipart/form-data request against the gateway
// (used by the mail send endpoint). fields are plain form fields.
func (c *Client) DoMultipart(ctx context.Context, method, path string, fields map[string]string, files []FilePart, bearer string) (*Response, error) {
	body := &bytes.Buffer{}
	writer := newMultipartWriter(body)

	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, fmt.Errorf("write field %q: %w", name, err)
		}
	}
	for _, f := range files {
		if err := writer.writeFile(f); err != nil {
			return nil, fmt.Errorf("write file %q: %w", f.Filename, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}

	return c.do(req)
}

func (c *Client) do(req *http.Request) (*Response, error) {
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gateway call: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, fmt.Errorf("read gateway response: %w", err)
	}

	return &Response{
		Status:      resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        data,
		Header:      resp.Header,
	}, nil
}
