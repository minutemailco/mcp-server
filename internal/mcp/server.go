// Package mcp implements a stateless MCP server over the Streamable HTTP
// transport (protocol revision 2025-06-18). Only the tools capability is
// exposed; there are no sessions, subscriptions or server-push streams.
package mcp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"mcp-server/internal/gateway"
	"mcp-server/internal/jsonrpc"
	"mcp-server/internal/metrics"
	"mcp-server/internal/tools"
)

// LatestProtocolRevision is the protocol version this server implements.
const LatestProtocolRevision = "2025-06-18"

// ServerVersion is reported in the initialize handshake's serverInfo. It
// mirrors the module version; bump it when tagging a release.
const ServerVersion = "1.1.2"

var supportedRevisions = map[string]bool{
	"2025-06-18": true,
	"2025-03-26": true,
	"2024-11-05": true,
}

// Server handles MCP requests on POST /mcp.
type Server struct {
	registry *tools.Registry
	gateway  *gateway.Client
}

// New builds the MCP server.
func New(registry *tools.Registry, gw *gateway.Client) *Server {
	return &Server{registry: registry, gateway: gw}
}

// Handler returns the HTTP handler with /mcp, /metrics and health endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/ready", handleHealth)
	return mux
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handlePost(w, r)
	case http.MethodGet:
		// Stateless server: no server-push SSE stream and no sessions.
		w.Header().Set("Allow", "POST")
		http.Error(w, "streamable HTTP GET (SSE) is not supported by this stateless server", http.StatusMethodNotAllowed)
	case http.MethodDelete:
		w.Header().Set("Allow", "POST")
		http.Error(w, "sessions are not supported by this stateless server", http.StatusMethodNotAllowed)
	default:
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		writeJSONRPC(w, http.StatusBadRequest, jsonrpc.NewError(nullID(), jsonrpc.CodeParseError, "unable to read request body"))
		return
	}

	req, err := jsonrpc.ParseRequest(body)
	if err != nil {
		writeJSONRPC(w, http.StatusBadRequest, jsonrpc.NewError(nullID(), jsonrpc.CodeParseError, err.Error()))
		return
	}

	// Notifications get no response.
	if req.IsNotification() {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	resp := s.dispatch(r, req)
	writeJSONRPC(w, http.StatusOK, resp)
}

func (s *Server) dispatch(r *http.Request, req *jsonrpc.Request) *jsonrpc.Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		// Notifications never reach dispatch (IsNotification short-circuit),
		// but a client may mislabel it with an id.
		return jsonrpc.NewResult(req.ID, map[string]any{})
	case "ping":
		return jsonrpc.NewResult(req.ID, map[string]any{})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(r, req)
	case "resources/list":
		// Tools-only server: answer with an empty list rather than
		// method-not-found so capability probes from clients and registries
		// don't log warnings.
		return jsonrpc.NewResult(req.ID, map[string]any{"resources": []any{}})
	case "prompts/list":
		return jsonrpc.NewResult(req.ID, map[string]any{"prompts": []any{}})
	case "resources/templates/list":
		return jsonrpc.NewResult(req.ID, map[string]any{"resourceTemplates": []any{}})
	default:
		return jsonrpc.NewError(req.ID, jsonrpc.CodeMethodNotFound, "method not found: "+req.Method)
	}
}

func (s *Server) handleInitialize(req *jsonrpc.Request) *jsonrpc.Response {
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(req.Params, &params)

	version := LatestProtocolRevision
	if supportedRevisions[params.ProtocolVersion] {
		version = params.ProtocolVersion
	}

	return jsonrpc.NewResult(req.ID, map[string]any{
		"protocolVersion": version,
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":    "minutemail-mcp-server",
			"version": ServerVersion,
		},
	})
}

func (s *Server) handleToolsList(req *jsonrpc.Request) *jsonrpc.Response {
	list := s.registry.All()
	out := make([]map[string]any, 0, len(list))
	for _, t := range list {
		entry := map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": t.InputSchema,
		}
		if t.Title != "" {
			entry["title"] = t.Title
		}
		if t.Annotations != nil {
			entry["annotations"] = t.Annotations
		}
		if t.OutputSchema != nil {
			entry["outputSchema"] = t.OutputSchema
		}
		out = append(out, entry)
	}
	return jsonrpc.NewResult(req.ID, map[string]any{"tools": out})
}

func (s *Server) handleToolsCall(r *http.Request, req *jsonrpc.Request) *jsonrpc.Response {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return jsonrpc.NewError(req.ID, jsonrpc.CodeInvalidParams, "invalid tools/call params: "+err.Error())
	}

	tool, ok := s.registry.Get(params.Name)
	if !ok {
		return jsonrpc.NewError(req.ID, jsonrpc.CodeInvalidParams, "unknown tool: "+params.Name)
	}

	var args map[string]any
	if len(params.Arguments) > 0 {
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			metrics.ToolCallsTotal.WithLabelValues(tool.Name, metrics.ResultError).Inc()
			return jsonrpc.NewError(req.ID, jsonrpc.CodeInvalidParams, "arguments must be a JSON object: "+err.Error())
		}
	}

	bearer := bearerFromRequest(r)
	if bearer == "" {
		metrics.ToolCallsTotal.WithLabelValues(tool.Name, metrics.ResultError).Inc()
		return toolErrorResult(req.ID, "missing Authorization header: send \"Authorization: Bearer <mmak_...>\" with your MinuteMail API key")
	}

	result, err := tool.Handler(r.Context(), args, bearer, s.gateway)
	if err != nil {
		// Handler errors are either invalid tool arguments or gateway
		// transport failures; distinguish by code.
		metrics.ToolCallsTotal.WithLabelValues(tool.Name, metrics.ResultError).Inc()
		code := jsonrpc.CodeInvalidParams
		var upstream *tools.UpstreamError
		if errors.As(err, &upstream) {
			code = jsonrpc.CodeServerError
		}
		return jsonrpc.NewError(req.ID, code, err.Error())
	}

	// Anything non-success (isError results from non-2xx gateway responses)
	// counts as an error.
	callResult := metrics.ResultSuccess
	if result.IsError {
		callResult = metrics.ResultError
	}
	metrics.ToolCallsTotal.WithLabelValues(tool.Name, callResult).Inc()
	return jsonrpc.NewResult(req.ID, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": result.Text},
		},
		"isError": result.IsError,
	})
}

func toolErrorResult(id json.RawMessage, message string) *jsonrpc.Response {
	return jsonrpc.NewResult(id, map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": message},
		},
		"isError": true,
	})
}

// bearerFromRequest extracts the raw API key from the Authorization header.
func bearerFromRequest(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func writeJSONRPC(w http.ResponseWriter, status int, resp *jsonrpc.Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

func nullID() json.RawMessage {
	return json.RawMessage("null")
}
