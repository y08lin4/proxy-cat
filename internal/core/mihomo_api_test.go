package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMihomoClientRequiresLocalhost(t *testing.T) {
	if _, err := NewMihomoClient("http://example.com:9090", "", nil); err == nil {
		t.Fatal("NewMihomoClient() error = nil, want non-localhost error")
	}
}

func TestSelectProxyRequest(t *testing.T) {
	var gotMethod string
	var gotPath string
	var gotAuth string
	var gotBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewMihomoClient(server.URL, "secret-token", server.Client())
	if err != nil {
		t.Fatalf("NewMihomoClient() error = %v", err)
	}

	err = client.SelectProxy(context.Background(), "PROXY Group", "HK-1")
	if err != nil {
		t.Fatalf("SelectProxy() error = %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Fatalf("method = %s, want %s", gotMethod, http.MethodPut)
	}
	if gotPath != "/proxies/PROXY%20Group" {
		t.Fatalf("path = %s, want escaped proxy group path", gotPath)
	}
	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
	if gotBody["name"] != "HK-1" {
		t.Fatalf("body name = %q, want HK-1", gotBody["name"])
	}
}

func TestGetProxies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies" {
			t.Fatalf("path = %s, want /proxies", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"proxies":{"PROXY":{"type":"Selector"}}}`))
	}))
	defer server.Close()

	client, err := NewMihomoClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("NewMihomoClient() error = %v", err)
	}

	proxies, err := client.GetProxies(context.Background())
	if err != nil {
		t.Fatalf("GetProxies() error = %v", err)
	}
	if _, ok := proxies.Proxies["PROXY"]; !ok {
		t.Fatalf("PROXY group missing from response: %#v", proxies.Proxies)
	}
}

func TestTestProxyDelayRequest(t *testing.T) {
	var gotPath string
	var gotURL string
	var gotTimeout string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotURL = r.URL.Query().Get("url")
		gotTimeout = r.URL.Query().Get("timeout")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"delay":123}`))
	}))
	defer server.Close()

	client, err := NewMihomoClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("NewMihomoClient() error = %v", err)
	}

	delay, err := client.TestProxyDelay(context.Background(), "HK 1", "https://example.test/204", 3000)
	if err != nil {
		t.Fatalf("TestProxyDelay() error = %v", err)
	}
	if delay.Delay != 123 {
		t.Fatalf("delay = %d, want 123", delay.Delay)
	}
	if gotPath != "/proxies/HK%201/delay" {
		t.Fatalf("path = %s, want escaped proxy delay path", gotPath)
	}
	if gotURL != "https://example.test/204" || gotTimeout != "3000" {
		t.Fatalf("query url=%q timeout=%q", gotURL, gotTimeout)
	}
}

func TestHTTPErrorIncludesStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	client, err := NewMihomoClient(server.URL, "", server.Client())
	if err != nil {
		t.Fatalf("NewMihomoClient() error = %v", err)
	}

	err = client.CloseConnections(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 400") {
		t.Fatalf("CloseConnections() error = %v, want status 400", err)
	}
}
