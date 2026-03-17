package scraper

import (
	"context"
	"io"
	"net/http"
	"net/url"

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
			s.cacheRobots(host, nil)
			return true, nil
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := s.httpClient.Do(req)
		if err != nil || resp.StatusCode == http.StatusNotFound {
			s.cacheRobots(host, nil)
			return true, nil
		}
		defer resp.Body.Close()
		limited := io.LimitReader(resp.Body, 512*1024)
		bodyBytes, err := io.ReadAll(limited)
		if err != nil {
			s.cacheRobots(host, nil)
			return true, nil
		}
		data, err = robotstxt.FromStatusAndBytes(resp.StatusCode, bodyBytes)
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
