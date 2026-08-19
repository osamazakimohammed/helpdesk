package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
)

type queryCounterKey struct{}

// QueryTracker tracks database query execution count within a request
type QueryTracker struct {
	count atomic.Int64
}

func (q *QueryTracker) Inc() {
	q.count.Add(1)
}

func (q *QueryTracker) Count() int64 {
	return q.count.Load()
}

// WithQueryTracker attaches a query tracker to the request context
func WithQueryTracker(ctx context.Context) (context.Context, *QueryTracker) {
	tracker := &QueryTracker{}
	return context.WithValue(ctx, queryCounterKey{}, tracker), tracker
}

// GetQueryTracker retrieves the tracker from context
func GetQueryTracker(ctx context.Context) *QueryTracker {
	if tracker, ok := ctx.Value(queryCounterKey{}).(*QueryTracker); ok {
		return tracker
	}
	return nil
}

type queryResponseWriter struct {
	http.ResponseWriter
	tracker     *QueryTracker
	maxAllowed  int64
	failOnError bool
	wroteHeader bool
}

func (w *queryResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		queryCount := w.tracker.Count()
		w.Header().Set("X-DB-Query-Count", fmt.Sprintf("%d", queryCount))

		if w.maxAllowed > 0 && queryCount > w.maxAllowed {
			w.Header().Set("X-DB-Query-Budget-Violated", "true")
			if w.failOnError {
				panic(fmt.Sprintf("Query budget exceeded: %d queries executed (limit: %d)", queryCount, w.maxAllowed))
			}
		}
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *queryResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// QueryBudgetMiddleware logs query counts and sets X-DB-Query-Count header
func QueryBudgetMiddleware(maxAllowed int64, failOnError bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, tracker := WithQueryTracker(r.Context())
			
			rw := &queryResponseWriter{
				ResponseWriter: w,
				tracker:        tracker,
				maxAllowed:     maxAllowed,
				failOnError:    failOnError,
			}

			next.ServeHTTP(rw, r.WithContext(ctx))

			// Ensure header is written if not flushed yet
			if !rw.wroteHeader {
				rw.WriteHeader(http.StatusOK)
			}
		})
	}
}
