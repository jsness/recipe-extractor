package wayback

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jsness/recipe-extractor/server/internal/httpfetch"
	"github.com/jsness/recipe-extractor/server/scraper"
)

const availabilityEndpoint = "https://archive.org/wayback/available"
const cdxEndpoint = "https://web.archive.org/cdx/search/cdx"

type Snapshot struct {
	URL       string
	Timestamp string
}

type Client struct {
	httpClient      *http.Client
	availabilityURL string
	cdxURL          string
	logger          *log.Logger
}

func New(timeout time.Duration) *Client {
	return NewWithLogger(timeout, nil)
}

func NewWithLogger(timeout time.Duration, logger *log.Logger) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		httpClient:      &http.Client{Timeout: timeout},
		availabilityURL: availabilityEndpoint,
		cdxURL:          cdxEndpoint,
		logger:          logger,
	}
}

func (c *Client) Lookup(ctx context.Context, sourceURL string) (*Snapshot, error) {
	if scraper.IsArchivedURL(sourceURL) {
		return nil, nil
	}

	reqURL, err := neturl.Parse(c.availabilityURL)
	if err != nil {
		return nil, err
	}

	query := reqURL.Query()
	query.Set("url", sourceURL)
	reqURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := httpfetch.Fetch(c.httpClient, req, httpfetch.Options{MaxBytes: 1024 * 1024, Logger: c.logger, Stage: "archive_lookup"})
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wayback availability check failed (%d)", res.StatusCode)
	}

	var body availabilityResponse
	if err := json.Unmarshal(res.Body, &body); err != nil {
		return nil, err
	}

	closest := body.ArchivedSnapshots.Closest
	if !closest.Available || closest.URL == "" {
		return nil, nil
	}
	if _, ok := OriginalURL(closest.URL); !ok {
		return nil, fmt.Errorf("wayback returned an invalid snapshot URL")
	}

	return &Snapshot{
		URL:       closest.URL,
		Timestamp: closest.Timestamp,
	}, nil
}

// ListSnapshots returns at most three recent, distinct successful HTML captures.
// This is only called after an archived extraction fails; the availability API
// remains the fast path for the existing archive lookup endpoint.
func (c *Client) ListSnapshots(ctx context.Context, sourceURL string) ([]Snapshot, error) {
	if scraper.IsArchivedURL(sourceURL) {
		return nil, fmt.Errorf("snapshot search requires an original URL")
	}
	reqURL, err := neturl.Parse(c.cdxURL)
	if err != nil {
		return nil, err
	}
	query := reqURL.Query()
	query.Set("url", sourceURL)
	query.Set("output", "json")
	query.Set("fl", "timestamp,original,statuscode,mimetype")
	query.Add("filter", "statuscode:200")
	query.Add("filter", "mimetype:text/html")
	query.Set("collapse", "digest")
	query.Set("limit", "-3")
	reqURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := httpfetch.Fetch(c.httpClient, req, httpfetch.Options{MaxBytes: 1024 * 1024, Logger: c.logger, Stage: "archive_candidates"})
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wayback snapshot search failed (%d)", res.StatusCode)
	}
	var rows [][]string
	if err := json.Unmarshal(res.Body, &rows); err != nil {
		return nil, fmt.Errorf("decode wayback snapshots: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	if strings.Join(rows[0], ",") != "timestamp,original,statuscode,mimetype" {
		return nil, fmt.Errorf("unexpected wayback snapshot columns")
	}
	var snapshots []Snapshot
	seen := make(map[string]bool)
	for _, row := range rows[1:] {
		if len(row) != 4 || row[2] != "200" || row[3] != "text/html" {
			continue
		}
		if _, err := time.Parse("20060102150405", row[0]); err != nil {
			continue
		}
		// Validate the replay URL as well as the index metadata before fetching.
		u := "https://web.archive.org/web/" + row[0] + "/" + row[1]
		if original, ok := OriginalURL(u); !ok || !SameSourceURL(original, sourceURL) || seen[u] {
			continue
		}
		seen[u] = true
		snapshots = append(snapshots, Snapshot{URL: u, Timestamp: row[0]})
	}
	sort.Slice(snapshots, func(i, j int) bool { return snapshots[i].Timestamp > snapshots[j].Timestamp })
	if len(snapshots) > 3 {
		snapshots = snapshots[:3]
	}
	return snapshots, nil
}

var replayTimestamp = regexp.MustCompile(`^[0-9]{1,14}(?:[a-z]{2}_)?$`)

// OriginalURL recognizes timestamped Wayback replay URLs, including raw replay
// modifiers. It preserves the original URL's query and escaped path.
func OriginalURL(rawURL string) (string, bool) {
	u, err := neturl.Parse(rawURL)
	if err != nil || !scraper.IsArchivedURL(rawURL) || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return "", false
	}
	path, ok := strings.CutPrefix(u.EscapedPath(), "/web/")
	if !ok {
		return "", false
	}
	timestamp, original, ok := strings.Cut(path, "/")
	if !ok || !replayTimestamp.MatchString(timestamp) {
		return "", false
	}
	if u.RawQuery != "" {
		original += "?" + u.RawQuery
	}
	source, err := neturl.Parse(original)
	if err != nil || source.Hostname() == "" || source.User != nil || (source.Scheme != "https" && source.Scheme != "http") || scraper.IsArchivedURL(original) {
		return "", false
	}
	return original, true
}

// SameSourceURL allows older HTTP captures of an HTTPS source, but requires the
// same host, path and query so another recipe cannot be substituted.
func SameSourceURL(a, b string) bool {
	x, y := sourceKey(a), sourceKey(b)
	return x != "" && x == y
}

func sourceKey(rawURL string) string {
	u, err := neturl.Parse(rawURL)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	port := u.Port()
	if (u.Scheme == "http" && port == "80") || (u.Scheme == "https" && port == "443") {
		port = ""
	}
	host := net.JoinHostPort(strings.ToLower(u.Hostname()), port)
	return host + "/" + u.EscapedPath() + "?" + u.RawQuery
}

// CaptureKey ignores the archive's scheme and replay modifier when identifying
// a capture, so the same snapshot is not fetched twice via different wrappers.
func CaptureKey(rawURL string) string {
	original, ok := OriginalURL(rawURL)
	if !ok {
		return ""
	}
	u, _ := neturl.Parse(rawURL)
	timestamp, _, _ := strings.Cut(strings.TrimPrefix(u.EscapedPath(), "/web/"), "/")
	timestamp = strings.TrimRight(timestamp, "abcdefghijklmnopqrstuvwxyz_")
	return timestamp + "/" + sourceKey(original)
}

type availabilityResponse struct {
	ArchivedSnapshots struct {
		Closest struct {
			Available bool   `json:"available"`
			URL       string `json:"url"`
			Timestamp string `json:"timestamp"`
		} `json:"closest"`
	} `json:"archived_snapshots"`
}
