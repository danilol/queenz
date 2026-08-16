# 🌐 Ingestion Engine: Concurrency, Architecture, & Anti-Blocking

The **Ingestion Context** (`internal/ingestion/`) is a high-performance, asynchronous web crawling and HTML parsing engine designed to extract rich data from the Fandom Drag Race Wiki; the scraper returns parser.Scraped* DTOs, and a later layer maps those DTOs into the system's core domain models.

---

## 🏗️ Architectural Overview

The engine acts as the first layer of the **QueenX** data collection pipeline. It implements a decoupled architecture following the **Single Responsibility Principle (SRP)**:
- **`scraper.go`**: Handles HTTP networking, rate-limiting, anti-blocking, and asynchronous fan-out orchestration.
- **`parser/`**: Performs pure HTML/DOM queries, extracting structural data into domain-ready transfer objects (DTOs) without network side effects.

### Interfaces First

The orchestration is governed by a strict Go interface:

```go
type Scraper interface {
    ScrapeFranchises(ctx context.Context) ([]*parser.ScrapedFranchise, error)
    ScrapeSeasons(ctx context.Context, f *parser.ScrapedFranchise) ([]*parser.ScrapedSeason, error)
    ScrapeEpisodes(ctx context.Context, s *parser.ScrapedSeason) ([]*parser.ScrapedEpisode, error)
    Orchestrate(ctx context.Context) ([]*parser.ScrapedFranchise, []*parser.ScrapedSeason, []*parser.ScrapedEpisode, error)
}
```

This interface facilitates clean mocking during tests and allows the presentation layer or CLI utilities to trigger localized or global scrapers.

---

## 🛡️ Anti-Blocking Strategy (Fandom API vs. Colly)

Fandom Wiki utilizes advanced CDN/WAF rules (Cloudflare) that block raw Go User-Agents or atypical TLS handshakes with `403 Forbidden` responses. To bypass these constraints reliably in production while supporting fast, offline unit testing, the scraper implements a **dual-source fetching engine**:

### 1. Production API Route (MediaWiki Parse API)
When querying the live Fandom Wiki (`rupaulsdragrace.fandom.com`), the scraper completely bypasses standard HTML visits. Instead, it queries the public MediaWiki Parse JSON API:

```text
GET /api.php?action=parse&page={page_name}&format=json
```

- **Headers**: Injects a standard browser User-Agent (`Mozilla/5.0...`).
- **Response**: Decodes the structural JSON containing the rendered HTML content, which is then fed into the local parsers.
- **Benefit**: Avoids heavy page renders, is optimized for CDN edge delivery, and bypasses standard Cloudflare bot-detection hurdles.

### 2. Test Fallback (Colly HTML Visits)
During local unit and integration tests (such as verifying local `httptest.NewServer` outputs), the scraper automatically falls back to raw HTTP crawling using **Colly**:
- Prevents hitting the live Fandom API.
- Assures robust, fast testing against local mock HTML templates.

---

## ⚡ Concurrency & Rate Limiting

Scraping a high-volume wiki requires polite execution to avoid IP bans, server CPU spikes, or memory exhaustion.

### 1. Colly Limit Rules
The base collector enforces polite delays:
- **Delay**: `100ms` base delay.
- **RandomDelay**: `50ms` jitter to avoid pattern detection.
- **Parallelism**: Moderate pool size of `4` concurrent requests per domain.

### 2. Fan-Out Orchestration with `errgroup`
The orchestrator crawls three logical layers: **Franchises → Seasons → Episodes**. To achieve high throughput, it uses `golang.org/x/sync/errgroup` to fetch sub-resources in parallel:

```go
g, gCtx := errgroup.WithContext(ctx)
// ...
g.Go(func() error {
    // Concurrent fetch logic
})
```

### 3. Bounded Concurrency via Semaphores
To prevent launching thousands of simultaneous network requests for seasons and episodes, a semaphore pattern is implemented using buffered channels:

```go
// Limit active HTTP scrapes to prevent overloading the wiki or memory
sem := make(chan struct{}, 4)

for _, f := range franchises {
    fCopy := f
    g.Go(func() error {
        select {
        case sem <- struct{}{}:
        case <-gCtx.Done():
            return gCtx.Err()
        }
        defer func() { <-sem }()
        
        // Crawl operations...
    })
}
```

---

## 📝 Usage Example

A concrete instance of the scraper can be initialized and executed from any command-line tool or background job scheduler:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "queenx/internal/ingestion"
)

func main() {
    ctx := context.Background()
    scraper := ingestion.NewScraper("https://rupaulsdragrace.fandom.com")
    
    franchises, seasons, episodes, err := scraper.Orchestrate(ctx)
    if err != nil {
        log.Fatalf("Scrape failed: %v", err)
    }
    
    fmt.Printf("Successfully scraped: %d franchises, %d seasons, %d episodes\n",
        len(franchises), len(seasons), len(episodes))
}
```
