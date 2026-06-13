// Package semanticscholar is the library behind the s2 command: the HTTP
// client, request shaping, and the typed data models for Semantic Scholar.
//
// The public Graph API at api.semanticscholar.org/graph/v1 is free and open;
// no API key is required. Rate limit is approximately 100 requests per 5
// minutes for unauthenticated access. This client defaults to 200ms between
// requests to stay well within that budget.
package semanticscholar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	graphPath = "/graph/v1"
	recoPath  = "/recommendations/v1"
)

// DefaultUserAgent identifies the client to the Semantic Scholar API.
const DefaultUserAgent = "s2/dev (+https://github.com/tamnd/semanticscholar-cli)"

// ErrNotFound is returned when the API returns no result for a given ID.
var ErrNotFound = errors.New("not found")

var arXivRe = regexp.MustCompile(`^\d{4}\.\d{4,5}$`)

// parseID maps a user-supplied identifier to the form the API expects.
// DOI (starts with "10.") gets "DOI:" prefix; arXiv IDs (NNNN.NNNNN) get
// "ARXIV:" prefix; anything else is used as-is (Semantic Scholar paperId).
func parseID(s string) string {
	if strings.HasPrefix(s, "10.") {
		return "DOI:" + s
	}
	if arXivRe.MatchString(s) {
		return "ARXIV:" + s
	}
	return s
}

// Config holds constructor parameters for Client.
type Config struct {
	UserAgent string
	BaseURL   string // override graph API base for testing; defaults to https://api.semanticscholar.org
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns conservative defaults that respect the public rate limit.
func DefaultConfig() Config {
	return Config{
		UserAgent: DefaultUserAgent,
		Rate:      200 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the Semantic Scholar Graph API.
// Rate and Retries may be set directly after construction (e.g. in tests).
type Client struct {
	httpClient *http.Client
	userAgent  string
	baseURL    string
	Rate       time.Duration
	Retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured from cfg. Use DefaultConfig() to get
// conservative defaults and override individual fields as needed.
func NewClient(cfg Config) *Client {
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.semanticscholar.org"
	}
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		userAgent:  cfg.UserAgent,
		baseURL:    base,
		Rate:       cfg.Rate,
		Retries:    cfg.Retries,
	}
}

// Get fetches a URL with pacing and retries. It is exported so tests can
// exercise transport-level behaviour (User-Agent, retry logic) without
// needing a real API response.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Rate <= 0 {
		return
	}
	if wait := c.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL string, v any) error {
	body, err := c.Get(ctx, rawURL)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── paper fields ─────────────────────────────────────────────────────────────

const paperFields = "paperId,title,abstract,year,authors,externalIds,citationCount," +
	"referenceCount,influentialCitationCount,isOpenAccess,openAccessPdf,venue,publicationDate"

const paperListFields = "paperId,title,year,authors,citationCount"

// ─── SearchPapers ─────────────────────────────────────────────────────────────

// SearchPapers searches Semantic Scholar for papers matching query.
// It paginates through results until limit is reached.
func (c *Client) SearchPapers(ctx context.Context, query string, limit int) ([]Paper, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("fields", paperFields)
	params.Set("limit", strconv.Itoa(pageSize(limit)))

	var out []Paper
	offset := 0
	for {
		params.Set("offset", strconv.Itoa(offset))
		rawURL := c.baseURL + graphPath + "/paper/search?" + params.Encode()
		var resp paperSearchResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		for _, p := range resp.Data {
			out = append(out, apiPaperToRecord(p, len(out)+1))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if resp.Next == nil || len(resp.Data) == 0 {
			break
		}
		offset = *resp.Next
	}
	return out, nil
}

// Paper fetches a single paper by ID. id may be an S2 paperId, DOI, or arXiv ID.
func (c *Client) Paper(ctx context.Context, id string) (Paper, error) {
	apiID := parseID(id)
	params := url.Values{}
	params.Set("fields", paperFields)
	rawURL := c.baseURL + graphPath + "/paper/" + url.PathEscape(apiID) + "?" + params.Encode()
	var p apiPaper
	if err := c.getJSON(ctx, rawURL, &p); err != nil {
		return Paper{}, fmt.Errorf("paper %q: %w", id, err)
	}
	if p.PaperID == "" {
		return Paper{}, fmt.Errorf("paper %q: %w", id, ErrNotFound)
	}
	return apiPaperToRecord(p, 0), nil
}

// Citations returns papers that cite the given paper.
func (c *Client) Citations(ctx context.Context, id string, limit int) ([]Paper, error) {
	apiID := parseID(id)
	params := url.Values{}
	params.Set("fields", paperListFields)
	params.Set("limit", strconv.Itoa(pageSize(limit)))

	var out []Paper
	offset := 0
	for {
		params.Set("offset", strconv.Itoa(offset))
		rawURL := c.baseURL + graphPath + "/paper/" + url.PathEscape(apiID) + "/citations?" + params.Encode()
		var resp citationsResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, fmt.Errorf("citations %q: %w", id, err)
		}
		for _, item := range resp.Data {
			if item.CitingPaper == nil {
				continue
			}
			out = append(out, apiPaperToRecord(*item.CitingPaper, len(out)+1))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if resp.Next == nil || len(resp.Data) == 0 {
			break
		}
		offset = *resp.Next
	}
	return out, nil
}

// References returns papers referenced by the given paper.
func (c *Client) References(ctx context.Context, id string, limit int) ([]Paper, error) {
	apiID := parseID(id)
	params := url.Values{}
	params.Set("fields", paperListFields)
	params.Set("limit", strconv.Itoa(pageSize(limit)))

	var out []Paper
	offset := 0
	for {
		params.Set("offset", strconv.Itoa(offset))
		rawURL := c.baseURL + graphPath + "/paper/" + url.PathEscape(apiID) + "/references?" + params.Encode()
		var resp citationsResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, fmt.Errorf("references %q: %w", id, err)
		}
		for _, item := range resp.Data {
			if item.CitedPaper == nil {
				continue
			}
			out = append(out, apiPaperToRecord(*item.CitedPaper, len(out)+1))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if resp.Next == nil || len(resp.Data) == 0 {
			break
		}
		offset = *resp.Next
	}
	return out, nil
}

// SearchAuthors searches Semantic Scholar for authors matching query.
func (c *Client) SearchAuthors(ctx context.Context, query string, limit int) ([]Author, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("fields", "authorId,name,hIndex,citationCount,paperCount,affiliations")
	params.Set("limit", strconv.Itoa(pageSize(limit)))

	var out []Author
	offset := 0
	for {
		params.Set("offset", strconv.Itoa(offset))
		rawURL := c.baseURL + graphPath + "/author/search?" + params.Encode()
		var resp authorSearchResp
		if err := c.getJSON(ctx, rawURL, &resp); err != nil {
			return out, err
		}
		for _, a := range resp.Data {
			out = append(out, apiAuthorToRecord(a, len(out)+1))
			if limit > 0 && len(out) >= limit {
				return out, nil
			}
		}
		if resp.Next == nil || len(resp.Data) == 0 {
			break
		}
		offset = *resp.Next
	}
	return out, nil
}

// AuthorProfile fetches an author profile by authorId.
func (c *Client) AuthorProfile(ctx context.Context, id string) (Author, []Paper, error) {
	params := url.Values{}
	params.Set("fields", "authorId,name,hIndex,citationCount,paperCount,affiliations,papers.paperId,papers.title,papers.year")
	rawURL := c.baseURL + graphPath + "/author/" + url.PathEscape(id) + "?" + params.Encode()
	var a apiAuthorFull
	if err := c.getJSON(ctx, rawURL, &a); err != nil {
		return Author{}, nil, fmt.Errorf("author %q: %w", id, err)
	}
	if a.AuthorID == "" {
		return Author{}, nil, fmt.Errorf("author %q: %w", id, ErrNotFound)
	}
	author := apiAuthorToRecord(a, 0)
	papers := make([]Paper, 0, len(a.Papers))
	for i, p := range a.Papers {
		papers = append(papers, apiPaperToRecord(p, i+1))
	}
	return author, papers, nil
}

// Recommend returns papers recommended as similar to the given paper.
func (c *Client) Recommend(ctx context.Context, id string, limit int) ([]Paper, error) {
	apiID := parseID(id)
	params := url.Values{}
	params.Set("fields", paperListFields)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}
	rawURL := c.baseURL + recoPath + "/papers/forpaper/" + url.PathEscape(apiID) + "?" + params.Encode()
	var resp recommendResp
	if err := c.getJSON(ctx, rawURL, &resp); err != nil {
		return nil, fmt.Errorf("recommend %q: %w", id, err)
	}
	out := make([]Paper, 0, len(resp.RecommendedPapers))
	for _, p := range resp.RecommendedPapers {
		out = append(out, apiPaperToRecord(p, len(out)+1))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

// pageSize returns the per-page fetch size: capped at 100 (API max), defaulting
// to 10 when limit is 0.
func pageSize(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}
