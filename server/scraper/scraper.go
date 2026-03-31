package scraper

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/temoto/robotstxt"

	"recipe-extractor/server/extractor"
)

const maxBodyBytes int64 = 2 * 1024 * 1024
const userAgent = "recipe-extractor/0.1 (+https://localhost)"
const blockedAccessMessage = "site blocked automated access and requires a browser challenge"

type Scraper struct {
	httpClient  *http.Client
	robotsMu    sync.RWMutex
	robotsCache map[string]*robotstxt.RobotsData
}

type Result struct {
	SourceURL   string
	HTML        string
	JSONLD      []string
	Text        string
	Links       []string // deduplicated "link text [url]" annotations from every <a> in the page
	Ingredients []extractor.IngredientGroup
}

type FetchErrorKind string

const (
	FetchErrorKindUnexpectedStatus FetchErrorKind = "unexpected_status"
	FetchErrorKindBlockedAccess    FetchErrorKind = "blocked_access"
)

type FetchError struct {
	Kind       FetchErrorKind
	StatusCode int
}

func (e *FetchError) Error() string {
	if e.Kind == FetchErrorKindBlockedAccess {
		return fmt.Sprintf("%s (HTTP %d)", blockedAccessMessage, e.StatusCode)
	}
	return fmt.Sprintf("unexpected status code: %d", e.StatusCode)
}

func New(timeout time.Duration) *Scraper {
	return &Scraper{
		httpClient:  &http.Client{Timeout: timeout},
		robotsCache: make(map[string]*robotstxt.RobotsData),
	}
}

func (s *Scraper) Fetch(ctx context.Context, sourceURL string) (Result, error) {
	allowed, err := s.robotsAllowed(ctx, sourceURL)
	if err != nil {
		return Result{}, fmt.Errorf("robots.txt check: %w", err)
	}
	if !allowed {
		return Result{}, fmt.Errorf("robots.txt disallows scraping %s", sourceURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()

	limited := io.LimitReader(res.Body, maxBodyBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return Result{}, err
	}

	html := string(bodyBytes)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if isBlockedAccessResponse(res, html) {
			return Result{}, &FetchError{Kind: FetchErrorKindBlockedAccess, StatusCode: res.StatusCode}
		}
		return Result{}, &FetchError{Kind: FetchErrorKindUnexpectedStatus, StatusCode: res.StatusCode}
	}

	jsonld := extractJSONLD(html)
	text := collapseWhitespace(stripTags(annotateLinks(html)))
	if len(text) > 20000 {
		text = text[:20000]
	}
	links := extractLinks(html)
	ingredients := extractIngredientGroups(html)

	return Result{
		SourceURL:   sourceURL,
		HTML:        html,
		JSONLD:      jsonld,
		Text:        text,
		Links:       links,
		Ingredients: ingredients,
	}, nil
}

func IsBlockedAccessError(err error) bool {
	var fetchErr *FetchError
	return errors.As(err, &fetchErr) && fetchErr.Kind == FetchErrorKindBlockedAccess
}

func SupportsArchivedFallback(errorMessage string) bool {
	return strings.Contains(errorMessage, blockedAccessMessage)
}

func isBlockedAccessResponse(res *http.Response, body string) bool {
	if strings.EqualFold(strings.TrimSpace(res.Header.Get("cf-mitigated")), "challenge") {
		return true
	}

	if res.StatusCode != http.StatusForbidden && res.StatusCode != http.StatusTooManyRequests {
		return false
	}

	lowerBody := strings.ToLower(body)
	return strings.Contains(lowerBody, "enable javascript and cookies to continue") ||
		strings.Contains(lowerBody, "just a moment") ||
		strings.Contains(lowerBody, "captcha")
}

func IsArchivedURL(rawURL string) bool {
	parsed, err := neturl.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Hostname(), "web.archive.org")
}

var jsonLDRegex = regexp.MustCompile(`(?is)<script[^>]*type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
var tagRegex = regexp.MustCompile(`(?s)<[^>]+>`)
var linkRegex = regexp.MustCompile(`(?is)<a\s+[^>]*href=["']([^"'#][^"']*)["'][^>]*>(.*?)</a>`)
var wprmIngredientGroupRegex = regexp.MustCompile(`(?is)<div[^>]*class=["'][^"']*\bwprm-recipe-ingredient-group\b[^"']*["'][^>]*>(.*?)</div>`)
var wprmIngredientGroupNameRegex = regexp.MustCompile(`(?is)<h4[^>]*class=["'][^"']*\bwprm-recipe-ingredient-group-name\b[^"']*["'][^>]*>(.*?)</h4>`)
var wprmIngredientItemRegex = regexp.MustCompile(`(?is)<li[^>]*class=["'][^"']*\bwprm-recipe-ingredient\b[^"']*["'][^>]*>(.*?)</li>`)

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

func annotateLinks(html string) string {
	return linkRegex.ReplaceAllString(html, "$2 [$1]")
}

func extractLinks(html string) []string {
	matches := linkRegex.FindAllStringSubmatch(html, -1)
	seen := make(map[string]struct{})
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) < 3 {
			continue
		}
		href := strings.TrimSpace(m[1])
		text := collapseWhitespace(stripTags(m[2]))
		if href == "" || len(text) < 4 {
			continue
		}
		if _, ok := seen[href]; ok {
			continue
		}
		seen[href] = struct{}{}
		out = append(out, text+" ["+href+"]")
	}
	return out
}

func extractIngredientGroups(htmlText string) []extractor.IngredientGroup {
	matches := wprmIngredientGroupRegex.FindAllStringSubmatch(htmlText, -1)
	if len(matches) == 0 {
		return nil
	}

	groups := make([]extractor.IngredientGroup, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		block := match[1]
		groupName := ""
		if nameMatch := wprmIngredientGroupNameRegex.FindStringSubmatch(block); len(nameMatch) == 2 {
			groupName = cleanHTMLText(nameMatch[1])
		}

		itemMatches := wprmIngredientItemRegex.FindAllStringSubmatch(block, -1)
		items := make([]string, 0, len(itemMatches))
		for _, itemMatch := range itemMatches {
			if len(itemMatch) < 2 {
				continue
			}
			item := cleanHTMLText(itemMatch[1])
			if item != "" {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}

		groups = append(groups, extractor.IngredientGroup{
			Group: groupName,
			Items: items,
		})
	}

	if len(groups) == 0 {
		return nil
	}
	return groups
}

func stripTags(html string) string {
	return tagRegex.ReplaceAllString(html, " ")
}

func collapseWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func cleanHTMLText(s string) string {
	return collapseWhitespace(strings.TrimSpace(html.UnescapeString(stripTags(s))))
}
