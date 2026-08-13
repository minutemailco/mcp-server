package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoForwardsBearerAndBody(t *testing.T) {
	var gotAuth, gotCT string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotCT = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"mb1"}`))
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.Do(context.Background(), "POST", "/v1/mailboxes", []byte(`{"domain":"minutemail.cc"}`), "mmak_test")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 201 {
		t.Fatalf("status = %d, want 201", resp.Status)
	}
	if !resp.OK() {
		t.Fatal("expected OK")
	}
	if gotAuth != "Bearer mmak_test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotCT != "application/json" {
		t.Fatalf("content-type = %q", gotCT)
	}
	if gotBody["domain"] != "minutemail.cc" {
		t.Fatalf("body = %v", gotBody)
	}
}

func TestDoNoBearerNoHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if _, err := New(srv.URL).Do(context.Background(), "GET", "/v1/mailboxes", nil, ""); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "" {
		t.Fatalf("auth header leaked: %q", gotAuth)
	}
}

func TestDoMultipart(t *testing.T) {
	var gotField, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse multipart: %v", err)
			w.WriteHeader(400)
			return
		}
		gotField = r.FormValue("sender")
		if r.MultipartForm == nil || len(r.MultipartForm.File["files"]) != 1 {
			t.Errorf("expected 1 file part, got %v", r.MultipartForm)
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"id":"m1"}`))
	}))
	defer srv.Close()

	files := []FilePart{{Field: "files", Filename: "a.txt", ContentType: "text/plain", Data: []byte("hello")}}
	resp, err := New(srv.URL).DoMultipart(context.Background(), "POST", "/v1/mailboxes/mb1/mails", map[string]string{"sender": "x@y.z"}, files, "mmak_k")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 201 {
		t.Fatalf("status = %d", resp.Status)
	}
	if gotField != "x@y.z" {
		t.Fatalf("sender field = %q", gotField)
	}
	if len(gotCT) == 0 || gotCT[:19] != "multipart/form-data" {
		t.Fatalf("content-type = %q", gotCT)
	}
}
