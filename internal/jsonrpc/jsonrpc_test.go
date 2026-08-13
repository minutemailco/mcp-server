package jsonrpc

import (
	"encoding/json"
	"testing"
)

func TestParseRequest(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"valid request", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, false},
		{"valid notification", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, false},
		{"string id", `{"jsonrpc":"2.0","id":"abc","method":"ping"}`, false},
		{"null id", `{"jsonrpc":"2.0","id":null,"method":"ping"}`, false},
		{"bad version", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, true},
		{"missing method", `{"jsonrpc":"2.0","id":1}`, true},
		{"invalid json", `{`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := ParseRequest([]byte(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", req)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsNotification(t *testing.T) {
	cases := map[string]bool{
		``:      true,
		`null`:  true,
		`1`:     false,
		`"abc"`: false,
	}
	for id, want := range cases {
		req := &Request{ID: json.RawMessage(id)}
		if got := req.IsNotification(); got != want {
			t.Errorf("IsNotification(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestResponseEncoding(t *testing.T) {
	resp := NewResult(json.RawMessage(`42`), map[string]any{"ok": true})
	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"jsonrpc":"2.0","id":42,"result":{"ok":true}}`
	if string(data) != want {
		t.Fatalf("got %s, want %s", data, want)
	}

	resp = NewError(json.RawMessage(`"err"`), CodeMethodNotFound, "nope")
	data, err = json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	want = `{"jsonrpc":"2.0","id":"err","error":{"code":-32601,"message":"nope"}}`
	if string(data) != want {
		t.Fatalf("got %s, want %s", data, want)
	}
}
