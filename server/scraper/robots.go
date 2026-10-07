package scraper

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jsness/recipe-extractor/server/internal/httpfetch"
	"github.com/temoto/robotstxt"
)

func (s *Scraper) robotsAllowed(ctx context.Context, targetURL string) (bool, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return false, err
	}
	host := parsed.Host

	s.robotsMu.RLock()
	data, ok := s.robotsCache[host]
	s.robotsMu.RUnlock()

	if !ok {
		robotsURL := parsed.Scheme + "://" + host + "/robots.txt"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
		if err != nil {
			return false, err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := httpfetch.Fetch(s.httpClient, req, httpfetch.Options{MaxBytes: 512 * 1024, Logger: s.logger, Stage: "robots", StopRetry: isBlockedFetchResponse})
		if err != nil {
			return false, err
		}
		if isBlockedFetchResponse(resp) {
			return false, &FetchError{Kind: FetchErrorKindBlockedAccess, StatusCode: resp.StatusCode}
		}
		if resp.StatusCode == http.StatusNotFound {
			s.cacheRobots(host, nil)
			return true, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 || resp.StatusCode == http.StatusRequestTimeout {
			return false, fmt.Errorf("temporary robots.txt failure (HTTP %d)", resp.StatusCode)
		}
		data, err = robotstxt.FromStatusAndBytes(resp.StatusCode, resp.Body)
		if err != nil {
			s.cacheRobots(host, nil)
			return true, nil
		}
		s.cacheRobots(host, data)
	}

	if data == nil {
		return true, nil
	}
	group := data.FindGroup(userAgent)
	return group.Test(parsed.Path), nil
}

func (s *Scraper) cacheRobots(host string, data *robotstxt.RobotsData) {
	s.robotsMu.Lock()
	s.robotsCache[host] = data
	s.robotsMu.Unlock()
}
