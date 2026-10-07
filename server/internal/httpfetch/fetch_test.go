package httpfetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchRetryPolicy(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		retryAfter string
		want       int
	}{
		{"rate limit", 429, "0", 3},
		{"server unavailable", 503, "0", 3},
		{"forbidden", 403, "0", 1},
		{"missing", 404, "0", 1},
		{"long retry after", 429, "3600", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				w.Header().Set("Retry-After", tc.retryAfter)
				w.WriteHeader(tc.status)
			}))
			defer server.Close()
			req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
			res, err := Fetch(server.Client(), req, Options{MaxBytes: 1024})
			if err != nil || res.StatusCode != tc.status || attempts != tc.want {
				t.Fatalf("Fetch() = %#v, %v; attempts=%d, want %d", res, err, attempts, tc.want)
			}
		})
	}
}

func TestFetchCancelsBackoff(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	_, err := Fetch(server.Client(), req, Options{MaxBytes: 1024})
	if !errors.Is(err, context.DeadlineExceeded) || attempts != 1 {
		t.Fatalf("Fetch() error=%v, attempts=%d", err, attempts)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestFetchRetriesInterruptedBody(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		var body io.ReadCloser = io.NopCloser(strings.NewReader("recipe"))
		if attempts == 1 {
			body = io.NopCloser(&interruptedBody{})
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Retry-After": {"0"}}, Body: body}, nil
	})}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	res, err := Fetch(client, req, Options{MaxBytes: 1024})
	if err != nil || attempts != 2 || string(res.Body) != "recipe" {
		t.Fatalf("Fetch() = %#v, %v; attempts=%d", res, err, attempts)
	}
}

type interruptedBody struct{}

func (*interruptedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestFetchRejectsOversizedResponse(t *testing.T) {
	attempts := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		attempts++
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("too large"))}, nil
	})}
	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	_, err := Fetch(client, req, Options{MaxBytes: 3})
	if err == nil || !strings.Contains(err.Error(), "byte limit") || attempts != 1 {
		t.Fatalf("Fetch() error=%v, attempts=%d", err, attempts)
	}
}

func TestRetryAfter(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		value string
		want  time.Duration
		valid bool
	}{
		{"0", 0, true},
		{"5", 5 * time.Second, true},
		{now.Add(10 * time.Second).Format(http.TimeFormat), 10 * time.Second, true},
		{now.Add(-time.Second).Format(http.TimeFormat), 0, true},
		{"9223372036854775807", maxRetryDelay + time.Second, true},
		{"99999999999999999999999999999", maxRetryDelay + time.Second, true},
		{"-1", 0, false},
		{"invalid", 0, false},
	} {
		if got, valid := retryAfter(tc.value, now); got != tc.want || valid != tc.valid {
			t.Errorf("retryAfter(%q) = %s, %v; want %s, %v", tc.value, got, valid, tc.want, tc.valid)
		}
	}
}
