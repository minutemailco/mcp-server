package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"mcp-server/internal/gateway"
	"mcp-server/internal/metrics"
	"mcp-server/internal/tools"
)

func newTestServer(gwURL string) *httptest.Server {
	reg := tools.NewRegistry()
	return httptest.NewServer(New(reg, gateway.New(gwURL)).Handler())
}

func post(t *testing.T, url, body, auth string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest("POST", url+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var out map[string]any
	_ = json.Unmarshal(data, &out)
	return resp.StatusCode, out
}

func TestInitialize(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	status, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26"}}`, "")
	if status != 200 {
		t.Fatalf("status = %d", status)
	}
	if resp["result"].(map[string]any)["protocolVersion"] != "2025-03-26" {
		t.Fatalf("protocol negotiation failed: %v", resp)
	}
}

func TestInitializeUnknownVersionFallsBack(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	_, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"1999-01-01"}}`, "")
	if resp["result"].(map[string]any)["protocolVersion"] != LatestProtocolRevision {
		t.Fatalf("expected fallback to %s: %v", LatestProtocolRevision, resp)
	}
}

func TestNotificationAcceptedNoBody(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL+"/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", resp.StatusCode)
	}
}

func TestUnsupportedCapabilityReturnsEmptyList(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	for method, key := range map[string]string{
		"resources/list":           "resources",
		"prompts/list":             "prompts",
		"resources/templates/list": "resourceTemplates",
	} {
		_, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":7,"method":"`+method+`"}`, "")
		result := resp["result"].(map[string]any)
		list, ok := result[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("%s: expected empty %s list, got: %v", method, key, resp)
		}
	}
}

func TestMethodNotFound(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	_, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":7,"method":"sampling/createMessage"}`, "")
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != -32601 {
		t.Fatalf("error = %v", resp)
	}
}

func TestGetAndDeleteRejected(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	for _, method := range []string{"GET", "DELETE"} {
		req, _ := http.NewRequest(method, srv.URL+"/mcp", nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("%s status = %d, want 405", method, resp.StatusCode)
		}
	}
}

func TestToolsList(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	_, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, "")
	result := resp["result"].(map[string]any)
	list := result["tools"].([]any)
	if len(list) != len(tools.NewRegistry().All()) {
		t.Fatalf("tools count = %d", len(list))
	}
	first := list[0].(map[string]any)
	if first["name"] == "" || first["inputSchema"] == nil {
		t.Fatalf("malformed tool entry: %v", first)
	}
}

func TestToolsCallMissingAuth(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("gateway should not be called without bearer")
	}))
	defer gw.Close()

	srv := newTestServer(gw.URL)
	defer srv.Close()

	_, resp := post(t, srv.URL, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"mailboxes.list","arguments":{}}}`, "")
	result := resp["result"].(map[string]any)
	if result["isError"] != true {
		t.Fatalf("expected isError result: %v", resp)
	}
}

func TestToolsCallProxiesToGateway(t *testing.T) {
	var gotPath, gotAuth, gotMethod string
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotMethod = r.URL.Path, r.Header.Get("Authorization"), r.Method
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items":[{"id":"mb1","address":"a@minutemail.cc"}]}`))
	}))
	defer gw.Close()

	srv := newTestServer(gw.URL)
	defer srv.Close()

	_, resp := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"mailboxes.list","arguments":{"address":"a@minutemail.cc"}}}`,
		"Bearer mmak_test")
	if gotPath != "/v1/mailboxes" {
		t.Fatalf("gateway path = %q", gotPath)
	}
	if gotAuth != "Bearer mmak_test" {
		t.Fatalf("gateway auth = %q", gotAuth)
	}
	if gotMethod != "GET" {
		t.Fatalf("gateway method = %q", gotMethod)
	}
	result := resp["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("unexpected error result: %v", resp)
	}
	content := result["content"].([]any)[0].(map[string]any)
	if !strings.Contains(content["text"].(string), `"mb1"`) {
		t.Fatalf("unexpected content: %v", content)
	}
}

func TestToolsCallGatewayError(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Header().Set("X-RateLimit-Remaining", "0")
		_, _ = w.Write([]byte(`{"error":"limit","message":"daily quota exceeded"}`))
	}))
	defer gw.Close()

	srv := newTestServer(gw.URL)
	defer srv.Close()

	_, resp := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"mailboxes.list","arguments":{}}}`,
		"Bearer mmak_test")
	result := resp["result"].(map[string]any)
	if result["isError"] != true {
		t.Fatalf("expected isError: %v", resp)
	}
	content := result["content"].([]any)[0].(map[string]any)
	text := content["text"].(string)
	if !strings.Contains(text, "quota") || !strings.Contains(text, "429") {
		t.Fatalf("error text = %q", text)
	}
}

func TestToolsCallUnknownTool(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	_, resp := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"nope","arguments":{}}}`,
		"Bearer mmak_test")
	errObj := resp["error"].(map[string]any)
	if errObj["code"].(float64) != -32602 {
		t.Fatalf("error = %v", resp)
	}
}

func TestMetrics(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !bytes.Contains(data, []byte("go_goroutines")) {
		t.Fatalf("metrics = %d %s", resp.StatusCode, data)
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer("http://127.0.0.1:1")
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || !bytes.Contains(data, []byte("ok")) {
		t.Fatalf("health = %d %s", resp.StatusCode, data)
	}
}

func TestToolsCallMetricsIncrement(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer gw.Close()

	srv := newTestServer(gw.URL)
	defer srv.Close()

	successBefore := testutil.ToFloat64(metrics.ToolCallsTotal.WithLabelValues("mailboxes.list", metrics.ResultSuccess))
	errorBefore := testutil.ToFloat64(metrics.ToolCallsTotal.WithLabelValues("mailboxes.list", metrics.ResultError))

	_, resp := post(t, srv.URL,
		`{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"mailboxes.list","arguments":{}}}`,
		"Bearer mmak_test")
	if result := resp["result"].(map[string]any); result["isError"] == true {
		t.Fatalf("unexpected error result: %v", resp)
	}

	// Missing bearer resolves the tool but ends in an error result.
	post(t, srv.URL,
		`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"mailboxes.list","arguments":{}}}`,
		"")

	if got := testutil.ToFloat64(metrics.ToolCallsTotal.WithLabelValues("mailboxes.list", metrics.ResultSuccess)) - successBefore; got != 1 {
		t.Fatalf("success delta = %v, want 1", got)
	}
	if got := testutil.ToFloat64(metrics.ToolCallsTotal.WithLabelValues("mailboxes.list", metrics.ResultError)) - errorBefore; got != 1 {
		t.Fatalf("error delta = %v, want 1", got)
	}
}
