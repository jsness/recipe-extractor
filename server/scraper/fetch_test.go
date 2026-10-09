package scraper

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchRejectsSuccessfulChallengeResponses(t *testing.T) {
	for name, body := range map[string]string{
		"challenge header": "<html>Checking your browser</html>",
		"challenge page":   "<html><title>Just a moment...</title>Enable JavaScript and cookies to continue</html>",
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if name == "challenge header" {
					w.Header().Set("cf-mitigated", "challenge")
				}
				_, _ = w.Write([]byte(body))
			}))
			defer server.Close()
			_, err := New(time.Second).Fetch(context.Background(), server.URL+"/recipe")
			if !IsBlockedAccessError(err) {
				t.Fatalf("Fetch() error = %v, want blocked access", err)
			}
		})
	}
}

func TestFetchRetriesTransientStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`<script type="application/ld+json">{"@type":"Recipe"}</script>`))
	}))
	defer server.Close()
	result, err := New(time.Second).Fetch(context.Background(), server.URL+"/recipe")
	if err != nil || attempts != 2 || len(result.JSONLD) != 1 {
		t.Fatalf("Fetch() = %#v, %v; attempts = %d", result, err, attempts)
	}
}

func TestFetchDoesNotCacheTransientRobotsFailure(t *testing.T) {
	robotsAttempts, pageAttempts := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			robotsAttempts++
			if robotsAttempts <= 3 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
			return
		}
		pageAttempts++
	}))
	defer server.Close()
	s := New(time.Second)
	if _, err := s.Fetch(context.Background(), server.URL+"/recipe"); err == nil || pageAttempts != 0 {
		t.Fatalf("Fetch() error = %v, page attempts = %d; want robots failure", err, pageAttempts)
	}
	if _, err := s.Fetch(context.Background(), server.URL+"/recipe"); err != nil || robotsAttempts != 4 || pageAttempts != 1 {
		t.Fatalf("Fetch() error = %v, robots attempts = %d, page attempts = %d", err, robotsAttempts, pageAttempts)
	}
}

func TestFetchCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := New(time.Second).Fetch(ctx, "http://localhost/recipe")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want cancellation", err)
	}
}

func TestFetchDoesNotRetryChallengesOrRobotsDenials(t *testing.T) {
	for _, denial := range []bool{false, true} {
		t.Run(map[bool]string{false: "challenge", true: "robots"}[denial], func(t *testing.T) {
			pageAttempts := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/robots.txt" {
					if denial {
						_, _ = w.Write([]byte("User-agent: *\nDisallow: /\n"))
					} else {
						w.WriteHeader(http.StatusNotFound)
					}
					return
				}
				pageAttempts++
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte("Captcha required"))
			}))
			defer server.Close()
			_, err := New(time.Second).Fetch(context.Background(), server.URL+"/recipe")
			if denial {
				if !errors.Is(err, ErrRobotsDenied) || pageAttempts != 0 {
					t.Fatalf("error=%v, page attempts=%d", err, pageAttempts)
				}
			} else if !IsBlockedAccessError(err) || pageAttempts != 1 {
				t.Fatalf("error=%v, page attempts=%d", err, pageAttempts)
			}
		})
	}
}

func TestRobotsChallengeDoesNotBecomeCachedPermission(t *testing.T) {
	robotsAttempts, pageAttempts := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			robotsAttempts++
			if robotsAttempts == 1 {
				w.Header().Set("cf-mitigated", "challenge")
				_, _ = w.Write([]byte("Check your browser"))
				return
			}
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
			return
		}
		pageAttempts++
	}))
	defer server.Close()
	s := New(time.Second)
	_, err := s.Fetch(context.Background(), server.URL+"/recipe")
	if !errors.Is(err, ErrRobotsCheck) || !IsBlockedAccessError(err) || robotsAttempts != 1 || pageAttempts != 0 {
		t.Fatalf("error=%v, robots attempts=%d, page attempts=%d", err, robotsAttempts, pageAttempts)
	}
	_, err = s.Fetch(context.Background(), server.URL+"/recipe")
	if err != nil || robotsAttempts != 2 || pageAttempts != 1 {
		t.Fatalf("error=%v, robots attempts=%d, page attempts=%d", err, robotsAttempts, pageAttempts)
	}
}
