package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueryBudgetEnforcement(t *testing.T) {
	// Handler that executes 2 queries
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker := GetQueryTracker(r.Context())
		if tracker == nil {
			t.Fatal("expected query tracker in context")
		}
		tracker.Inc()
		tracker.Inc()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	middleware := QueryBudgetMiddleware(5, true)
	ts := httptest.NewServer(middleware(handler))
	defer ts.Close()

	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	queryCountHeader := resp.Header.Get("X-DB-Query-Count")
	if queryCountHeader != "2" {
		t.Errorf("expected X-DB-Query-Count header '2', got %q", queryCountHeader)
	}
}
