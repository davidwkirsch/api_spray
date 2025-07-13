package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"api_spray/pkg/types"
)

// Client wraps HTTP client functionality
type Client struct {
	client    *http.Client
	userAgent string
	retries   int
}

// NewClient creates a new HTTP client with the given configuration
func NewClient(config *types.Config) *Client {
	transport := &http.Transport{
		MaxIdleConns:        config.Threads * 2,
		MaxIdleConnsPerHost: config.Threads,
		IdleConnTimeout:     30 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives: false,
	}

	client := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	if !config.FollowRedirs {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return &Client{
		client:    client,
		userAgent: config.UserAgent,
		retries:   config.MaxRetries,
	}
}

// MakeRequest makes HTTP request with retries
func (hc *Client) MakeRequest(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", hc.userAgent)

	var resp *http.Response
	for attempt := 0; attempt <= hc.retries; attempt++ {
		resp, err = hc.client.Do(req)
		if err == nil {
			return resp, nil
		}

		if attempt < hc.retries {
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return nil, err
}

// ExtractTitle extracts title from HTML content
func ExtractTitle(content string) string {
	re := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches := re.FindStringSubmatch(content)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// GenerateURL generates URL based on scan mode
func GenerateURL(target, word string, mode types.ScanMode) string {
	target = strings.TrimSuffix(target, "/")
	word = strings.TrimPrefix(word, "/")

	switch mode {
	case types.ModeWildcards:
		return strings.ReplaceAll(target, "*", word)
	case types.ModeDirectories:
		return fmt.Sprintf("%s/%s", target, word)
	case types.ModeSubdomains:
		// Handle both domain.com and http://domain.com inputs
		domain := strings.TrimPrefix(target, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.Split(domain, "/")[0]
		return fmt.Sprintf("https://%s.%s", word, domain)
	default:
		return target
	}
}

// TestURL tests a single URL and returns the result
func TestURL(ctx context.Context, httpClient *Client, target, word, url string, statusCodes []int, disableHTTP bool) types.Result {
	start := time.Now()
	result := types.Result{
		Target: target,
		Word:   word,
		URL:    url,
	}

	// Try HTTPS first
	httpsURL := url
	if !strings.HasPrefix(url, "http") {
		httpsURL = "https://" + url
	} else if strings.HasPrefix(url, "http://") {
		httpsURL = strings.Replace(url, "http://", "https://", 1)
	}

	resp, err := httpClient.MakeRequest(ctx, httpsURL)
	if err != nil && !disableHTTP {
		// Fallback to HTTP
		httpURL := strings.Replace(httpsURL, "https://", "http://", 1)
		if httpURL != httpsURL {
			resp, err = httpClient.MakeRequest(ctx, httpURL)
			if err == nil {
				result.URL = httpURL
			}
		}
	}

	result.ResponseTime = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ContentLength = resp.ContentLength

	// Check if status code is in allowed list
	allowed := false
	for _, code := range statusCodes {
		if resp.StatusCode == code {
			allowed = true
			break
		}
	}

	if allowed {
		// Read response body for title extraction
		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
		if err == nil {
			result.Title = ExtractTitle(string(body))
		}
	}

	return result
}
