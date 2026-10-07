// Package httpfetch performs bounded retries for read-only HTTP requests.
package httpfetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const maxAttempts = 3
const maxRetryDelay = 30 * time.Second

type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

type Options struct {
	MaxBytes int64
	Logger   *log.Logger
	Stage    string
	// StopRetry prevents retries of permanent failures such as challenge pages.
	StopRetry func(*Response) bool
}

// Fetch is for GET requests only. Client.Timeout bounds each attempt; waits are
// cancelable and capped at 30 seconds. Longer Retry-After values stop retries
// rather than sending a request earlier than the server permits.
func Fetch(client *http.Client, req *http.Request, opts Options) (*Response, error) {
	if req.Method != http.MethodGet || opts.MaxBytes <= 0 {
		return nil, fmt.Errorf("httpfetch requires GET and a positive body limit")
	}
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := req.Context().Err(); err != nil {
			return nil, err
		}
		started := time.Now()
		res, err := client.Do(req.Clone(req.Context()))
		var result *Response
		if res != nil {
			result = &Response{StatusCode: res.StatusCode, Header: res.Header}
			if err == nil {
				result.Body, err = io.ReadAll(io.LimitReader(res.Body, opts.MaxBytes+1))
			}
			res.Body.Close()
			if int64(len(result.Body)) > opts.MaxBytes {
				err = fmt.Errorf("response exceeds %d byte limit", opts.MaxBytes)
			}
		}
		status := 0
		if result != nil {
			status = result.StatusCode
		}
		if opts.Logger != nil {
			opts.Logger.Printf("http stage=%s host=%s attempt=%d status=%d duration=%s error=%v", opts.Stage, req.URL.Hostname(), attempt, status, time.Since(started).Round(time.Millisecond), err)
		}
		if req.Context().Err() != nil {
			return nil, req.Context().Err()
		}
		retry := transientError(err)
		if err == nil && result != nil {
			retry = RetryableStatus(result.StatusCode)
			if opts.StopRetry != nil && opts.StopRetry(result) {
				retry = false
			}
		}
		if !retry || attempt == maxAttempts {
			return result, err
		}
		delay := time.Duration(1<<(attempt-1)) * 500 * time.Millisecond
		delay += time.Duration(rand.Int64N(int64(delay / 2)))
		if result != nil {
			if specified, ok := retryAfter(result.Header.Get("Retry-After"), time.Now()); ok {
				delay = specified
			}
		}
		if delay > maxRetryDelay {
			return result, err
		}
		if opts.Logger != nil {
			opts.Logger.Printf("http stage=%s retry_after=%s", opts.Stage, delay)
		}
		if err := wait(req.Context(), delay); err != nil {
			return nil, err
		}
	}
	panic("unreachable")
}

func RetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, http.StatusTooManyRequests, http.StatusInternalServerError,
		http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func transientError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	var netErr net.Error
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) ||
		(errors.As(err, &netErr) && (netErr.Timeout() || netErr.Temporary()))
}

func retryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value != "" && strings.Trim(value, "0123456789") == "" {
		seconds, err := strconv.ParseUint(value, 10, 64)
		// Avoid overflow for untrusted header values; any value above the cap
		// only needs to signal that retrying within our budget is impossible.
		if err != nil || seconds > uint64(maxRetryDelay/time.Second) {
			return maxRetryDelay + time.Second, true
		}
		return time.Duration(seconds) * time.Second, true
	}
	if date, err := http.ParseTime(value); err == nil {
		return max(0, date.Sub(now)), true
	}
	return 0, false
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
