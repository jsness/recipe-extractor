package wayback

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"time"

	"recipe-extractor/server/scraper"
)

const availabilityEndpoint = "https://archive.org/wayback/available"

type Snapshot struct {
	URL       string
	Timestamp string
}

type Client struct {
	httpClient *http.Client
}

func New(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *Client) Lookup(ctx context.Context, sourceURL string) (*Snapshot, error) {
	if scraper.IsArchivedURL(sourceURL) {
		return nil, nil
	}

	reqURL, err := neturl.Parse(availabilityEndpoint)
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

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wayback availability check failed (%d)", res.StatusCode)
	}

	var body availabilityResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}

	closest := body.ArchivedSnapshots.Closest
	if !closest.Available || closest.URL == "" {
		return nil, nil
	}

	return &Snapshot{
		URL:       closest.URL,
		Timestamp: closest.Timestamp,
	}, nil
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
