package resolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hostodo/hostodo-cli/pkg/api"
)

// testInstances returns a set of canned instances for testing
func testInstances() []api.Instance {
	return []api.Instance{
		{InstanceID: "inst-001", Hostname: "web-server-alpha"},
		{InstanceID: "inst-002", Hostname: "web-server-beta"},
		{InstanceID: "inst-003", Hostname: "db-primary"},
		{InstanceID: "inst-004", Hostname: "cache-node"},
	}
}

// newTestServer creates an httptest.Server that serves canned instances at
// /client/instances/ and tracks request count via the provided counter.
func newTestServer(instances []api.Instance, requestCount *atomic.Int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/client/instances/") {
			http.NotFound(w, r)
			return
		}
		if requestCount != nil {
			requestCount.Add(1)
		}
		resp := api.InstancesResponse{
			Count:   len(instances),
			Results: instances,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// newTestClient creates an api.Client pointed at the given test server URL
func newTestClient(serverURL string) *api.Client {
	return &api.Client{
		BaseURL:    serverURL,
		HTTPClient: &http.Client{},
		TokenFunc:  func() (string, error) { return "test-token", nil },
	}
}

func TestResolveInstance_ExactMatch(t *testing.T) {
	InvalidateCache()
	server := newTestServer(testInstances(), nil)
	defer server.Close()
	client := newTestClient(server.URL)

	result, err := ResolveInstance(client, "db-primary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != MatchExact {
		t.Errorf("expected MatchExact, got %s", result.MatchType)
	}
	if result.Instance.Hostname != "db-primary" {
		t.Errorf("expected hostname 'db-primary', got %q", result.Instance.Hostname)
	}
	if result.Instance.InstanceID != "inst-003" {
		t.Errorf("expected instance ID 'inst-003', got %q", result.Instance.InstanceID)
	}
}

func TestResolveInstance_PrefixMatch(t *testing.T) {
	InvalidateCache()
	server := newTestServer(testInstances(), nil)
	defer server.Close()
	client := newTestClient(server.URL)

	result, err := ResolveInstance(client, "db-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != MatchPrefix {
		t.Errorf("expected MatchPrefix, got %s", result.MatchType)
	}
	if result.Instance.Hostname != "db-primary" {
		t.Errorf("expected hostname 'db-primary', got %q", result.Instance.Hostname)
	}
}

func TestResolveInstance_AmbiguousPrefix(t *testing.T) {
	InvalidateCache()
	server := newTestServer(testInstances(), nil)
	defer server.Close()
	client := newTestClient(server.URL)

	_, err := ResolveInstance(client, "web-server")
	if err == nil {
		t.Fatal("expected error for ambiguous prefix, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous hostname prefix") {
		t.Errorf("expected 'ambiguous hostname prefix' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "web-server-alpha") || !strings.Contains(err.Error(), "web-server-beta") {
		t.Errorf("expected both matching hostnames in error, got: %v", err)
	}
}

func TestResolveInstance_IDFallback(t *testing.T) {
	InvalidateCache()
	server := newTestServer(testInstances(), nil)
	defer server.Close()
	client := newTestClient(server.URL)

	result, err := ResolveInstance(client, "inst-004")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MatchType != MatchID {
		t.Errorf("expected MatchID, got %s", result.MatchType)
	}
	if result.Instance.Hostname != "cache-node" {
		t.Errorf("expected hostname 'cache-node', got %q", result.Instance.Hostname)
	}
}

func TestResolveInstance_NoMatch(t *testing.T) {
	InvalidateCache()
	server := newTestServer(testInstances(), nil)
	defer server.Close()
	client := newTestClient(server.URL)

	_, err := ResolveInstance(client, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no match, got nil")
	}
	if !strings.Contains(err.Error(), "no instance found matching") {
		t.Errorf("expected 'no instance found matching' in error, got: %v", err)
	}
}

func TestGetInstancesCached(t *testing.T) {
	InvalidateCache()
	var requestCount atomic.Int64
	server := newTestServer(testInstances(), &requestCount)
	defer server.Close()
	client := newTestClient(server.URL)

	// First call should hit the server
	instances1, err := GetInstancesCached(client)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if len(instances1) != 4 {
		t.Fatalf("expected 4 instances, got %d", len(instances1))
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected 1 request after first call, got %d", requestCount.Load())
	}

	// Second call should use cache (no additional server hit)
	instances2, err := GetInstancesCached(client)
	if err != nil {
		t.Fatalf("unexpected error on second call: %v", err)
	}
	if len(instances2) != 4 {
		t.Fatalf("expected 4 instances on second call, got %d", len(instances2))
	}
	if requestCount.Load() != 1 {
		t.Errorf("expected still 1 request after second call (cache hit), got %d", requestCount.Load())
	}
}

func TestInvalidateCache(t *testing.T) {
	InvalidateCache()
	var requestCount atomic.Int64
	server := newTestServer(testInstances(), &requestCount)
	defer server.Close()
	client := newTestClient(server.URL)

	// First call: populates cache
	_, err := GetInstancesCached(client)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}
	if requestCount.Load() != 1 {
		t.Fatalf("expected 1 request after first call, got %d", requestCount.Load())
	}

	// Invalidate cache
	InvalidateCache()

	// Next call should hit server again
	_, err = GetInstancesCached(client)
	if err != nil {
		t.Fatalf("unexpected error after invalidation: %v", err)
	}
	if requestCount.Load() != 2 {
		t.Errorf("expected 2 requests after cache invalidation, got %d", requestCount.Load())
	}
}
