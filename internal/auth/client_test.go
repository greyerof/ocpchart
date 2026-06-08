package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromManual(t *testing.T) {
	creds := FromManual("https://thanos.example.com", "mytoken", true)

	if creds.ThanosURL != "https://thanos.example.com" {
		t.Fatalf("expected ThanosURL 'https://thanos.example.com', got %q", creds.ThanosURL)
	}

	if creds.Token != "mytoken" {
		t.Fatalf("expected Token 'mytoken', got %q", creds.Token)
	}

	if !creds.InsecureTLS {
		t.Fatal("expected InsecureTLS true")
	}
}

func TestFromManual_InsecureFalse(t *testing.T) {
	creds := FromManual("https://thanos.example.com", "tok", false)
	if creds.InsecureTLS {
		t.Fatal("expected InsecureTLS false")
	}
}

func TestHTTPClient_InjectsToken(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	creds := FromManual(server.URL, "test-token-123", false)
	client := creds.HTTPClient()

	resp, err := client.Get(server.URL + "/test")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	expected := "Bearer test-token-123"
	if receivedAuth != expected {
		t.Fatalf("expected Authorization %q, got %q", expected, receivedAuth)
	}
}

func TestTokenTransport_PreservesOtherHeaders(t *testing.T) {
	var receivedCustom string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedCustom = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	creds := FromManual(server.URL, "tok", false)
	client := creds.HTTPClient()

	req, _ := http.NewRequest("GET", server.URL+"/test", nil)
	req.Header.Set("X-Custom", "hello")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if receivedCustom != "hello" {
		t.Fatalf("expected X-Custom 'hello', got %q", receivedCustom)
	}
}
