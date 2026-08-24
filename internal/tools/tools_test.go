package tools

import (
	"net/http"
	"regexp"
	"strings"
	"testing"

	"mcp-server/internal/gateway"
)

// TestUniqueToolNames guards against accidental duplicate registrations.
func TestUniqueToolNames(t *testing.T) {
	reg := NewRegistry()
	seen := map[string]bool{}
	for _, tool := range reg.All() {
		if seen[tool.Name] {
			t.Fatalf("duplicate tool name %q", tool.Name)
		}
		seen[tool.Name] = true
		if tool.Handler == nil {
			t.Fatalf("tool %q has no handler", tool.Name)
		}
	}
}

// TestInjectToolNaming pins the injection tool's name: it simulates inbound
// mail for testing and must not read as sending external mail.
func TestInjectToolNaming(t *testing.T) {
	reg := NewRegistry()
	have := map[string]bool{}
	for _, tool := range reg.All() {
		have[tool.Name] = true
	}
	if !have["mails.inject"] {
		t.Fatal("expected tool mails.inject to be registered")
	}
	if have["mm_send_mail"] {
		t.Fatal("misleading name mm_send_mail must not be registered")
	}
}

// TestSchemasAreValidObjects checks the structural invariants MCP clients
// expect from an inputSchema.
func TestSchemasAreValidObjects(t *testing.T) {
	reg := NewRegistry()
	for _, tool := range reg.All() {
		s := tool.InputSchema
		if s["type"] != "object" {
			t.Fatalf("%s: schema type = %v", tool.Name, s["type"])
		}
		props, ok := s["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s: properties missing", tool.Name)
		}
		if req, ok := s["required"].([]string); ok {
			for _, name := range req {
				if _, ok := props[name]; !ok {
					t.Fatalf("%s: required field %q not in properties", tool.Name, name)
				}
			}
		}
	}
}

func TestArgString(t *testing.T) {
	if _, err := argString(map[string]any{"a": "x"}, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := argString(map[string]any{}, "a"); err == nil {
		t.Fatal("expected missing error")
	}
	if _, err := argString(map[string]any{"a": 42}, "a"); err == nil {
		t.Fatal("expected type error")
	}
	if _, err := argString(map[string]any{"a": "  "}, "a"); err == nil {
		t.Fatal("expected empty error")
	}
}

func TestArgHelpers(t *testing.T) {
	args := map[string]any{
		"n":    float64(5),
		"b":    true,
		"ids":  []any{"a", "b"},
		"objs": []any{map[string]any{"k": "v"}},
	}
	if v, ok, err := argIntOpt(args, "n"); err != nil || !ok || v != 5 {
		t.Fatalf("int: %v %v %v", v, ok, err)
	}
	if _, ok, _ := argIntOpt(args, "missing"); ok {
		t.Fatal("expected not-ok for missing")
	}
	if v, ok := argBoolOpt(args, "b"); !ok || !v {
		t.Fatalf("bool: %v %v", v, ok)
	}
	ids, err := argStringsOpt(args, "ids")
	if err != nil || len(ids) != 2 {
		t.Fatalf("strings: %v %v", ids, err)
	}
	objs, err := argObjectsOpt(args, "objs")
	if err != nil || len(objs) != 1 {
		t.Fatalf("objects: %v %v", objs, err)
	}
}

func TestPathEscaping(t *testing.T) {
	if got := path("v1", "mailboxes", "a/b"); got != "/v1/mailboxes/a%2Fb" {
		t.Fatalf("path = %q", got)
	}
	if got := path("v1", "mailboxes", "mb1"); got != "/v1/mailboxes/mb1" {
		t.Fatalf("path = %q", got)
	}
}

func result(t *testing.T, status int, body []byte) *Result {
	t.Helper()
	res, err := resultFromResponse(&gateway.Response{
		Status: status,
		Body:   body,
		Header: http.Header{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return res
}

func TestResultFromResponse(t *testing.T) {
	r := result(t, 204, nil)
	if r.IsError || !strings.Contains(r.Text, "204") {
		t.Fatalf("empty 204 result: %+v", r)
	}
	r = result(t, 200, []byte(`{"items":[]}`))
	if r.IsError || r.Text != `{"items":[]}` {
		t.Fatalf("200 result: %+v", r)
	}
	r = result(t, 401, []byte(`{}`))
	if !r.IsError || !strings.Contains(r.Text, "API key") {
		t.Fatalf("401 result: %+v", r)
	}
	r = result(t, 403, nil)
	if !r.IsError || !strings.Contains(r.Text, "scope") {
		t.Fatalf("403 result: %+v", r)
	}
}

// TestAllToolsHaveMeta ensures every registered tool carries the MCP metadata
// (title, annotations, outputSchema) that registries and clients expect.
func TestAllToolsHaveMeta(t *testing.T) {
	reg := NewRegistry()
	tools := reg.All()
	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}
	for _, tool := range tools {
		if tool.Title == "" {
			t.Errorf("%s: missing title", tool.Name)
		}
		if tool.Annotations == nil {
			t.Errorf("%s: missing annotations", tool.Name)
			continue
		}
		for _, key := range []string{"title", "readOnlyHint", "destructiveHint", "idempotentHint", "openWorldHint"} {
			if _, ok := tool.Annotations[key]; !ok {
				t.Errorf("%s: annotations missing %q", tool.Name, key)
			}
		}
		if tool.OutputSchema == nil {
			t.Errorf("%s: missing outputSchema", tool.Name)
			continue
		}
		if tool.OutputSchema["type"] == nil {
			t.Errorf("%s: outputSchema has no type", tool.Name)
		}
	}
}

// TestMetaTableCoversAllTools fails on stale entries in toolMetaTable.
func TestMetaTableCoversAllTools(t *testing.T) {
	reg := NewRegistry()
	registered := map[string]bool{}
	for _, tool := range reg.All() {
		registered[tool.Name] = true
	}
	for name := range toolMetaTable {
		if !registered[name] {
			t.Errorf("toolMetaTable has stale entry %q", name)
		}
	}
}

// TestToolNamesUseDotNotation enforces the hierarchical dot-notation naming
// scheme (e.g. mailboxes.list, team.members.add): 2-3 dot-separated
// lower_snake_case segments. Flat names score poorly on MCP registries.
func TestToolNamesUseDotNotation(t *testing.T) {
	seg := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, tool := range NewRegistry().All() {
		parts := strings.Split(tool.Name, ".")
		if len(parts) < 2 || len(parts) > 3 {
			t.Errorf("%s: expected 2-3 dot-separated segments", tool.Name)
			continue
		}
		for _, p := range parts {
			if !seg.MatchString(p) {
				t.Errorf("%s: invalid segment %q", tool.Name, p)
			}
		}
	}
}
