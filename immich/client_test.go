package immich

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPingUsesAuthenticatedEndpoint(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if r.Header.Get("x-api-key") == "" {
			t.Error("missing API key")
		}
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"id":"me"}`)
	}))
	defer srv.Close()

	c := &Client{URL: srv.URL, APIKey: "secret"}
	if err := c.Ping(); err != nil {
		t.Fatal(err)
	}
	if path != "/api/users/me" {
		t.Fatalf("ping path = %s", path)
	}
}

func TestClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()

	c := &Client{
		URL:    srv.URL,
		APIKey: "secret",
		HTTP:   &http.Client{Timeout: 50 * time.Millisecond},
	}
	if err := c.Ping(); err == nil {
		t.Fatal("expected timeout")
	}
}
