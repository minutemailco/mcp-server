// Package jsonrpc implements the minimal JSON-RPC 2.0 message types needed
// for the MCP Streamable HTTP transport.
package jsonrpc

import (
	"encoding/json"
	"fmt"
)

// Standard JSON-RPC 2.0 error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	// CodeServerError is the reserved implementation-defined server error
	// range start (-32000 to -32099).
	CodeServerError = -32000
)

// Request is an incoming JSON-RPC 2.0 request or notification. ID is kept as
// raw JSON so string, number and null IDs round-trip unchanged.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// IsNotification reports whether the request carries no id (a notification,
// which must not receive a response).
func (r *Request) IsNotification() bool {
	trimmed := trim(r.ID)
	return len(trimmed) == 0 || string(trimmed) == "null"
}

// Response is an outgoing JSON-RPC 2.0 response. Exactly one of Result and
// Error must be set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// NewResult builds a success response for the given request ID.
func NewResult(id json.RawMessage, result any) *Response {
	return &Response{JSONRPC: "2.0", ID: id, Result: result}
}

// NewError builds an error response for the given request ID.
func NewError(id json.RawMessage, code int, message string) *Response {
	return &Response{JSONRPC: "2.0", ID: id, Error: &Error{Code: code, Message: message}}
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc %d: %s", e.Code, e.Message)
}

// ParseRequest decodes a JSON-RPC 2.0 request from raw bytes.
func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	if req.JSONRPC != "" && req.JSONRPC != "2.0" {
		return nil, fmt.Errorf("unexpected jsonrpc version %q", req.JSONRPC)
	}
	if req.Method == "" {
		return nil, fmt.Errorf("missing method")
	}
	return &req, nil
}

func trim(raw json.RawMessage) json.RawMessage {
	// json.RawMessage from encoding/json has no surrounding whitespace.
	return raw
}
