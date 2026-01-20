package tempo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
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

// ResponseError captures Tempo response errors with status codes.
type ResponseError struct {
	StatusCode int
	Body       string
}

func (e ResponseError) Error() string {
	return fmt.Sprintf("tempo error: %d %s", e.StatusCode, e.Body)
}

// IsTimeout reports whether the error represents a timeout.
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
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

	params := BuildQueryParams(lookbackSeconds)
	params.Set("limit", "500") // Limit results per query (increased from 100 to 500)
	return c.searchTraces(ctx, params)
}

// SearchTraces pulls traces matching service and endpoint within a time range.
func (c *Client) SearchTraces(ctx context.Context, service, endpoint string, start, end int64, limit int) ([]domain.TraceEvent, error) {
	if c == nil {
		return nil, fmt.Errorf("tempo client is nil")
	}

	params := BuildRangeParams(start, end)
	params.Set("tags", BuildSearchTags(service, endpoint))
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	return c.searchTraces(ctx, params)
}

// GetTraceSpans fetches a trace by ID and returns normalized span data.
func (c *Client) GetTraceSpans(ctx context.Context, traceID string) ([]SpanData, error) {
	if c == nil {
		return nil, fmt.Errorf("tempo client is nil")
	}
	if strings.TrimSpace(traceID) == "" {
		return nil, fmt.Errorf("trace id is required")
	}

	endpoint := c.baseURL + "/api/traces/" + url.PathEscape(traceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tempo request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, ResponseError{StatusCode: resp.StatusCode, Body: string(b)}
	}

	var traceResp TraceByIDResponse
	if err := json.NewDecoder(resp.Body).Decode(&traceResp); err != nil {
		return nil, fmt.Errorf("decode tempo trace response: %w", err)
	}

	spans := extractSpanData(traceResp)
	return spans, nil
}

func extractSpanData(traceResp TraceByIDResponse) []SpanData {
	resourceSpans := traceResp.ResourceSpans
	if len(resourceSpans) == 0 {
		resourceSpans = traceResp.Batches
	}

	spans := make([]SpanData, 0)
	for _, rs := range resourceSpans {
		service := findServiceName(rs.Resource.Attributes)
		scopes := rs.ScopeSpans
		if len(scopes) == 0 {
			scopes = rs.InstrumentationLibrarySpans
		}
		for _, scope := range scopes {
			for _, span := range scope.Spans {
				spans = append(spans, SpanData{
					TraceID:           span.TraceID,
					SpanID:            span.SpanID,
					ParentSpanID:      span.ParentSpanID,
					Name:              span.Name,
					ServiceName:       service,
					StartTimeUnixNano: span.StartTimeUnixNano,
					EndTimeUnixNano:   span.EndTimeUnixNano,
				})
			}
		}
	}

	return spans
}

func findServiceName(attrs []KeyValue) string {
	for _, attr := range attrs {
		if attr.Key == "service.name" {
			return attr.Value.StringValue
		}
	}
	return ""
}

func (c *Client) searchTraces(ctx context.Context, params url.Values) ([]domain.TraceEvent, error) {
	endpoint := c.baseURL + "/api/search"
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
