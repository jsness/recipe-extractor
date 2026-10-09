package wayback

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestLookupRetriesAndPreservesSourceQuery(t *testing.T) {
	attempts := 0
	source := "https://example.com/recipe?x=1&y=2"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("url") != source {
			t.Errorf("source URL = %q", r.URL.Query().Get("url"))
		}
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"archived_snapshots":{"closest":{"available":true,"url":"https://web.archive.org/web/20260101120000/https://example.com/recipe?x=1&y=2","timestamp":"20260101120000"}}}`))
	}))
	defer server.Close()
	c := New(time.Second)
	c.availabilityURL = server.URL
	snapshot, err := c.Lookup(context.Background(), source)
	if err != nil || snapshot == nil || snapshot.Timestamp != "20260101120000" || attempts != 2 {
		t.Fatalf("Lookup() = %#v, %v; attempts=%d", snapshot, err, attempts)
	}
}

func TestLookupMissingAndInvalidSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		fail bool
	}{
		{"missing", `{"archived_snapshots":{}}`, false},
		{"invalid host", `{"archived_snapshots":{"closest":{"available":true,"url":"https://example.com/recipe"}}}`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			c := New(time.Second)
			c.availabilityURL = server.URL
			snapshot, err := c.Lookup(context.Background(), "https://example.com/recipe")
			if snapshot != nil || (err != nil) != tc.fail {
				t.Fatalf("Lookup() = %#v, %v", snapshot, err)
			}
		})
	}
}

func TestListSnapshotsFiltersAndOrdersCandidates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("url") != "https://example.com/recipe" || q.Get("limit") != "-3" || q.Get("collapse") != "digest" ||
			!reflect.DeepEqual(q["filter"], []string{"statuscode:200", "mimetype:text/html"}) {
			t.Errorf("unexpected query: %v", q)
		}
		_, _ = w.Write([]byte(`[
			["timestamp","original","statuscode","mimetype"],
			["20240101120000","https://example.com/recipe","200","text/html"],
			["20260101120000","https://example.com/recipe","200","text/html"],
			["20250101120000","http://example.com:80/recipe","200","text/html"],
			["20260101120000","https://example.com/recipe","200","text/html"],
			["20260201120000","https://example.com/other-recipe","200","text/html"],
			["20260301120000","https://example.com/recipe","403","text/html"],
			["invalid","https://example.com/recipe","200","text/html"],
			["20260401120000","https://example.com/recipe","200","application/json"]
		]`))
	}))
	defer server.Close()
	c := New(time.Second)
	c.cdxURL = server.URL
	snapshots, err := c.ListSnapshots(context.Background(), "https://example.com/recipe")
	if err != nil || len(snapshots) != 3 {
		t.Fatalf("ListSnapshots() = %#v, %v", snapshots, err)
	}
	for i, timestamp := range []string{"20260101120000", "20250101120000", "20240101120000"} {
		if snapshots[i].Timestamp != timestamp {
			t.Fatalf("ListSnapshots() = %#v, want newest first", snapshots)
		}
	}
}

func TestOriginalURLAndCaptureIdentity(t *testing.T) {
	for _, tc := range []struct {
		url  string
		want string
	}{
		{"https://web.archive.org/web/20260101120000/https://example.com/a%20b?x=1&y=2", "https://example.com/a%20b?x=1&y=2"},
		{"http://web.archive.org/web/20260101120000id_/https://example.com/recipe", "https://example.com/recipe"},
		{"https://example.com/recipe", ""},
		{"https://web.archive.org/web/latest/https://example.com/recipe", ""},
		{"https://web.archive.org/web/20260101120000/file:///recipe", ""},
		{"https://web.archive.org/web/20260101120000/https://web.archive.org/recipe", ""},
	} {
		got, ok := OriginalURL(tc.url)
		if got != tc.want || ok != (tc.want != "") {
			t.Errorf("OriginalURL(%q) = %q, %v; want %q", tc.url, got, ok, tc.want)
		}
	}
	a := CaptureKey("https://web.archive.org/web/20260101120000/https://example.com/recipe")
	b := CaptureKey("http://web.archive.org/web/20260101120000id_/http://example.com:80/recipe")
	if a == "" || a != b {
		t.Fatalf("capture identities differ: %q, %q", a, b)
	}
}

func TestSameSourceURLDoesNotMixRecipes(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want bool
	}{
		{"https://example.com/recipe", "http://example.com:80/recipe", true},
		{"https://EXAMPLE.com:443/recipe", "https://example.com/recipe", true},
		{"https://example.com/recipe", "https://example.com/other", false},
		{"https://example.com/recipe?a=1", "https://example.com/recipe?a=2", false},
		{"https://example.com/recipe", "https://other.com/recipe", false},
		{"https://example.com/recipe", "https://example.com:8443/recipe", false},
		{"", "", false},
	} {
		if got := SameSourceURL(tc.a, tc.b); got != tc.want {
			t.Errorf("SameSourceURL(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}
