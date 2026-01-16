package tempo

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "time"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
)

// Client is a minimal HTTP client for querying Tempo search API.
type Client struct {
    httpClient *http.Client
    baseURL    string
    authToken  string
}

// NewClient creates a new Tempo client using the provided configuration.
// It uses a sensible default timeout suitable for polling.
func NewClient(cfg config.TempoConfig) *Client {
    // default timeout for Tempo polling operations
    httpClient := &http.Client{Timeout: 15 * time.Second}
    return &Client{
        httpClient: httpClient,
        baseURL:    strings.TrimRight(cfg.URL, "/"),
        authToken:  strings.TrimSpace(cfg.AuthToken),
    }
}

// QueryTraces pulls recent traces within the last N seconds and converts them
// into domain.TraceEvent values.
func (c *Client) QueryTraces(ctx context.Context, lookbackSeconds int) ([]domain.TraceEvent, error) {
    if c == nil {
        return nil, fmt.Errorf("tempo client is nil")
    }

    endpoint := c.baseURL + "/api/search"
    params := BuildQueryParams(lookbackSeconds)
    params.Set("limit", "500") // Limit results per query (increased from 100 to 500)

    u, err := url.Parse(endpoint)
    if err != nil {
        return nil, fmt.Errorf("parse tempo base url: %w", err)
    }
    u.RawQuery = params.Encode()

    const maxAttempts = 3
    var lastErr error

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
        if err != nil {
            return nil, fmt.Errorf("build request: %w", err)
        }
        req.Header.Set("Accept", "application/json")
        if c.authToken != "" {
            req.Header.Set("Authorization", "Bearer "+c.authToken)
        }

        resp, err := c.httpClient.Do(req)
        if err != nil {
            lastErr = fmt.Errorf("tempo request failed (attempt %d/%d): %w", attempt, maxAttempts, err)
        } else {
            // Handle response
            if resp.StatusCode != http.StatusOK {
                b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
                resp.Body.Close()
                if resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests {
                    lastErr = fmt.Errorf("tempo non-200 (attempt %d/%d): %d %s", attempt, maxAttempts, resp.StatusCode, string(b))
                } else {
                    return nil, fmt.Errorf("tempo error: %d %s", resp.StatusCode, string(b))
                }
            } else {
                var tr TempoResponse
                err = json.NewDecoder(resp.Body).Decode(&tr)
                resp.Body.Close()
                if err != nil {
                    lastErr = fmt.Errorf("decode tempo response: %w", err)
                } else {
                    events := make([]domain.TraceEvent, 0, len(tr.Traces))
                    for _, t := range tr.Traces {
                        events = append(events, domain.TraceEvent{
                            TraceID:           t.TraceID,
                            RootServiceName:   t.RootServiceName,
                            RootTraceName:     t.RootTraceName,
                            StartTimeUnixNano: t.StartTimeUnixNano,
                            DurationMs:        t.DurationMs,
                        })
                    }
                    return events, nil
                }
            }
        }

        if attempt < maxAttempts {
            time.Sleep(time.Duration(1<<uint(attempt-1)) * 300 * time.Millisecond)
        }
    }

    if lastErr != nil {
        return nil, lastErr
    }
    return nil, fmt.Errorf("tempo query failed for unknown reasons")
}
