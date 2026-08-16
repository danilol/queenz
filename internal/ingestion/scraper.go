package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"queenx/internal/ingestion/parser"

	"github.com/gocolly/colly/v2"
	"golang.org/x/sync/errgroup"
)

// Scraper defines the interface for our concurrent wiki scraper.
type Scraper interface {
	ScrapeFranchises(ctx context.Context) ([]*parser.ScrapedFranchise, error)
	ScrapeSeasons(ctx context.Context, f *parser.ScrapedFranchise) ([]*parser.ScrapedSeason, error)
	ScrapeEpisodes(ctx context.Context, s *parser.ScrapedSeason) ([]*parser.ScrapedEpisode, error)
	Orchestrate(ctx context.Context) ([]*parser.ScrapedFranchise, []*parser.ScrapedSeason, []*parser.ScrapedEpisode, error)
}

type wikiScraper struct {
	baseURL       string
	mainPagePath  string
	baseCollector *colly.Collector
}

type wikiAPIResponse struct {
	Parse struct {
		Text struct {
			HTML string `json:"*"`
		} `json:"text"`
	} `json:"parse"`
}

// NewScraper creates a new instance of Colly-backed Scraper.
func NewScraper(baseURL string) Scraper {
	if baseURL == "" {
		baseURL = "https://rupaulsdragrace.fandom.com"
	}

	c := colly.NewCollector(
		colly.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"),
	)

	// Dynamically calculate and set allowed domains to support local test servers
	allowedDomains := []string{"rupaulsdragrace.fandom.com", "fandom.com"}
	if parsed, err := url.Parse(baseURL); err == nil && parsed.Host != "" {
		host := parsed.Host
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}
		allowedDomains = append(allowedDomains, host)
	}
	c.AllowedDomains = allowedDomains

	// Set respectful rate limiting
	_ = c.Limit(&colly.LimitRule{
		DomainRegexp: `rupaulsdragrace\.fandom\.com`,
		Delay:        100 * time.Millisecond, // fast but polite
		RandomDelay:  50 * time.Millisecond,
		Parallelism:  4, // moderate concurrency
	})

	return &wikiScraper{
		baseURL:       baseURL,
		mainPagePath:  "/wiki/Drag_Race_(Franchise)",
		baseCollector: c,
	}
}

// fetchHTML fetches the HTML of a page. It uses the MediaWiki Parse API for live Fandom scraping
// to bypass Cloudflare 403 Forbidden blocks, and falls back to Colly for local mock server tests.
func (ws *wikiScraper) fetchHTML(ctx context.Context, wikiPath string) (string, error) {
	if strings.Contains(ws.baseURL, "rupaulsdragrace.fandom.com") {
		pageName := strings.TrimPrefix(wikiPath, "/wiki/")
		apiURL := fmt.Sprintf("%s/api.php?action=parse&page=%s&format=json", ws.baseURL, pageName)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
		if err != nil {
			return "", fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("dispatching request: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("API response error: status %d", resp.StatusCode)
		}

		var apiResp wikiAPIResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
			return "", fmt.Errorf("parsing API response JSON: %w", err)
		}

		return apiResp.Parse.Text.HTML, nil
	}

	// Fallback to Colly for local test servers
	c := ws.baseCollector.Clone()
	var htmlContent string
	var fetchErr error

	c.OnResponse(func(r *colly.Response) {
		htmlContent = string(r.Body)
	})

	c.OnError(func(r *colly.Response, err error) {
		fetchErr = err
	})

	targetURL := ws.baseURL + wikiPath
	err := c.Visit(targetURL)
	if err != nil {
		return "", fmt.Errorf("visiting page: %w", err)
	}

	if fetchErr != nil {
		return "", fmt.Errorf("fetching page: %w", fetchErr)
	}

	return htmlContent, nil
}

func (ws *wikiScraper) ScrapeFranchises(ctx context.Context) ([]*parser.ScrapedFranchise, error) {
	htmlContent, err := ws.fetchHTML(ctx, ws.mainPagePath)
	if err != nil {
		return nil, fmt.Errorf("scraping franchises: %w", err)
	}
	return parser.ParseFranchises(htmlContent)
}

func (ws *wikiScraper) ScrapeSeasons(ctx context.Context, f *parser.ScrapedFranchise) ([]*parser.ScrapedSeason, error) {
	htmlContent, err := ws.fetchHTML(ctx, f.WikiURL)
	if err != nil {
		return nil, fmt.Errorf("scraping seasons for %s: %w", f.Name, err)
	}
	return parser.ParseSeasons(f.Name, htmlContent)
}

func (ws *wikiScraper) ScrapeEpisodes(ctx context.Context, s *parser.ScrapedSeason) ([]*parser.ScrapedEpisode, error) {
	htmlContent, err := ws.fetchHTML(ctx, s.WikiURL)
	if err != nil {
		return nil, fmt.Errorf("scraping episodes for %s: %w", s.Name, err)
	}
	return parser.ParseEpisodes(s.Name, htmlContent)
}

func (ws *wikiScraper) Orchestrate(ctx context.Context) ([]*parser.ScrapedFranchise, []*parser.ScrapedSeason, []*parser.ScrapedEpisode, error) {
	// 1. Scrape all franchises
	franchises, err := ws.ScrapeFranchises(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("orchestrating franchises: %w", err)
	}

	// 2. Concurrently scrape seasons for each franchise using errgroup
	g, gCtx := errgroup.WithContext(ctx)
	var seasonsMu sync.Mutex
	var allSeasons []*parser.ScrapedSeason

	// Limit simultaneous active HTTP scrapes to prevent overwhelming the wiki or memory
	sem := make(chan struct{}, 4)

	for _, f := range franchises {
		fCopy := f // capture loop variable
		g.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-gCtx.Done():
				return gCtx.Err()
			}
			defer func() { <-sem }()

			seasons, err := ws.ScrapeSeasons(gCtx, fCopy)
			if err != nil {
				return nil
			}

			seasonsMu.Lock()
			allSeasons = append(allSeasons, seasons...)
			seasonsMu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, nil, nil, fmt.Errorf("orchestrating seasons: %w", err)
	}

	// 3. Concurrently scrape episodes for each season
	g2, g2Ctx := errgroup.WithContext(ctx)
	var episodesMu sync.Mutex
	var allEpisodes []*parser.ScrapedEpisode

	for _, s := range allSeasons {
		sCopy := s // capture loop variable
		g2.Go(func() error {
			select {
			case sem <- struct{}{}:
			case <-g2Ctx.Done():
				return g2Ctx.Err()
			}
			defer func() { <-sem }()

			episodes, err := ws.ScrapeEpisodes(g2Ctx, sCopy)
			if err != nil {
				return nil
			}

			episodesMu.Lock()
			allEpisodes = append(allEpisodes, episodes...)
			episodesMu.Unlock()
			return nil
		})
	}

	if err := g2.Wait(); err != nil {
		return nil, nil, nil, fmt.Errorf("orchestrating episodes: %w", err)
	}

	return franchises, allSeasons, allEpisodes, nil
}
