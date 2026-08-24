// Package tools defines the MCP tools exposed by mcp-server. Every tool
// maps 1:1 onto a MinuteMail /v1 API route reachable through the api-gateway.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"mcp-server/internal/gateway"
)

// Tool is a single MCP tool definition plus its handler.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	// Title, Annotations and OutputSchema are spec metadata filled by
	// applyMeta from the toolMetaTable.
	Title        string
	Annotations  map[string]any
	OutputSchema map[string]any
	Handler      Handler
}

// Handler executes a tool call. arguments is the decoded "arguments" object
// from the tools/call request; bearer is the caller's API key.
type Handler func(ctx context.Context, args map[string]any, bearer string, gw *gateway.Client) (*Result, error)

// Result is the MCP tools/call result content.
type Result struct {
	Text    string
	IsError bool
}

// Registry holds all tools by name.
type Registry struct {
	tools  []Tool
	byName map[string]*Tool
}

// All returns the full ordered tool list.
func (r *Registry) All() []Tool {
	return r.tools
}

// Get looks a tool up by name.
func (r *Registry) Get(name string) (*Tool, bool) {
	t, ok := r.byName[name]
	return t, ok
}

// NewRegistry builds the tool registry with every MinuteMail API operation.
func NewRegistry() *Registry {
	list := buildTools()
	applyMeta(list)
	reg := &Registry{tools: list, byName: make(map[string]*Tool, len(list))}
	for i := range reg.tools {
		reg.byName[reg.tools[i].Name] = &reg.tools[i]
	}
	return reg
}

// ---------- schema helpers ----------

func schema(props map[string]any, required ...string) map[string]any {
	s := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		req := make([]string, len(required))
		copy(req, required)
		s["required"] = req
	}
	return s
}

func str(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func strEnum(desc string, values ...string) map[string]any {
	vals := make([]any, len(values))
	for i, v := range values {
		vals[i] = v
	}
	return map[string]any{"type": "string", "description": desc, "enum": vals}
}

func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func arr(desc string) map[string]any {
	return map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": desc}
}

// ---------- argument helpers ----------

func argString(args map[string]any, name string) (string, error) {
	v, ok := args[name]
	if !ok || v == nil {
		return "", fmt.Errorf("missing required argument %q", name)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("argument %q must be a string", name)
	}
	if strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("argument %q must not be empty", name)
	}
	return s, nil
}

func argStringOpt(args map[string]any, name string) string {
	if v, ok := args[name]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func argIntOpt(args map[string]any, name string) (int64, bool, error) {
	v, ok := args[name]
	if !ok || v == nil {
		return 0, false, nil
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true, nil
	case int64:
		return n, true, nil
	case int:
		return int64(n), true, nil
	default:
		return 0, false, fmt.Errorf("argument %q must be an integer", name)
	}
}

func argBoolOpt(args map[string]any, name string) (bool, bool) {
	if v, ok := args[name]; ok {
		if b, ok := v.(bool); ok {
			return b, true
		}
	}
	return false, false
}

// argStringsOpt extracts an optional array of strings.
func argStringsOpt(args map[string]any, name string) ([]string, error) {
	v, ok := args[name]
	if !ok || v == nil {
		return nil, nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("argument %q must be an array of strings", name)
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		s, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("argument %q must contain only strings", name)
		}
		out = append(out, s)
	}
	return out, nil
}

// argObjectsOpt extracts an optional array of objects.
func argObjectsOpt(args map[string]any, name string) ([]map[string]any, error) {
	v, ok := args[name]
	if !ok || v == nil {
		return nil, nil
	}
	raw, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("argument %q must be an array of objects", name)
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("argument %q must contain only objects", name)
		}
		out = append(out, m)
	}
	return out, nil
}

// path joins URL path segments, escaping each segment.
func path(segments ...string) string {
	escaped := make([]string, len(segments))
	for i, s := range segments {
		escaped[i] = url.PathEscape(s)
	}
	return "/" + strings.Join(escaped, "/")
}

// ---------- gateway call helpers ----------

// UpstreamError wraps gateway transport failures so the MCP layer can
// distinguish them from invalid tool arguments.
type UpstreamError struct{ Err error }

func (e *UpstreamError) Error() string { return e.Err.Error() }
func (e *UpstreamError) Unwrap() error { return e.Err }

// callJSON performs a gateway request and wraps the response as a tool
// result. Non-2xx responses become isError results with the status and body.
func callJSON(ctx context.Context, gw *gateway.Client, method, p string, body any, bearer string) (*Result, error) {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}
	return callRaw(ctx, gw, method, p, payload, bearer)
}

func callRaw(ctx context.Context, gw *gateway.Client, method, p string, payload []byte, bearer string) (*Result, error) {
	resp, err := gw.Do(ctx, method, p, payload, bearer)
	if err != nil {
		return nil, &UpstreamError{Err: err}
	}
	return resultFromResponse(resp)
}

func resultFromResponse(resp *gateway.Response) (*Result, error) {
	body := strings.TrimSpace(string(resp.Body))
	if resp.OK() {
		if body == "" {
			return &Result{Text: fmt.Sprintf(`{"status":"ok","http_status":%d}`, resp.Status)}, nil
		}
		return &Result{Text: body}, nil
	}

	msg := fmt.Sprintf("gateway returned HTTP %d", resp.Status)
	switch resp.Status {
	case http.StatusUnauthorized:
		msg += ": invalid or revoked API key"
	case http.StatusForbidden:
		msg += ": API key lacks the required scope or domain for this operation"
	case http.StatusTooManyRequests:
		remaining := resp.Header.Get("X-RateLimit-Remaining")
		if remaining != "" {
			msg += fmt.Sprintf(": quota exceeded (remaining: %s)", remaining)
		} else {
			msg += ": quota exceeded"
		}
	case http.StatusBadGateway, http.StatusServiceUnavailable:
		msg += ": upstream service unavailable"
	}
	if body != "" {
		msg += " — " + body
	}
	return &Result{Text: msg, IsError: true}, nil
}
