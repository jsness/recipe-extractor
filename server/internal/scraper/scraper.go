package scraper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const maxBodyBytes int64 = 2 * 1024 * 1024

type Scraper struct {
	httpClient *http.Client
}

type Result struct {
	SourceURL string
	HTML      string
	JSONLD    []string
	Text      string
}

func New(timeout time.Duration) *Scraper {
	return &Scraper{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (s *Scraper) Fetch(ctx context.Context, sourceURL string) (Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", "recipe-extractor/0.1 (+https://localhost)")

	res, err := s.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Result{}, fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	limited := io.LimitReader(res.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}

	html := string(bodyBytes)
	jsonld := extractJSONLD(html)
	text := collapseWhitespace(stripTags(html))
	if len(text) > 20000 {
		text = text[:20000]
	}

	return Result{
		SourceURL: sourceURL,
		HTML:      html,
		JSONLD:    jsonld,
		Text:      text,
	}, nil
}

var jsonLDRegex = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
var tagRegex = regexp.MustCompile(`(?s)<[^>]+>`)

func extractJSONLD(html string) []string {
	matches := jsonLDRegex.FindAllStringSubmatch(html, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		value := strings.TrimSpace(m[1])
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func stripTags(html string) string {
	return tagRegex.ReplaceAllString(html, " ")
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
