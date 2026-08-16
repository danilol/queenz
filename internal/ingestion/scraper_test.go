package ingestion

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWikiScraper_Orchestrate(t *testing.T) {
	// Setup a mock local HTTP server to stub the Wiki responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wiki/Drag_Race_(Franchise)":
			// Main wiki page listing franchises
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
				<body>
					<table class="wikitable">
						<tr><td><a href="/wiki/RuPaul%27s_Drag_Race">RuPaul's Drag Race</a></td><td>United States</td></tr>
					</table>
				</body>
				</html>
			`))

		case "/wiki/RuPaul's_Drag_Race":
			// Franchise page listing seasons
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
				<body>
					<table class="wikitable">
						<tr><td><a href="/wiki/RuPaul%27s_Drag_Race_(season_1)">Season 1</a></td><td>February 2, 2009</td></tr>
					</table>
				</body>
				</html>
			`))

		case "/wiki/RuPaul's_Drag_Race_(season_1)":
			// Season page listing episodes
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`
				<html>
				<body>
					<table class="wikitable">
						<tr><td>1</td><td>"Drag on a Dime"</td><td>February 2, 2009</td></tr>
					</table>
				</body>
				</html>
			`))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Initialize scraper using our mock server's URL as the base
	scraper := NewScraper(server.URL)
	ctx := context.Background()

	t.Run("full orchestration integration", func(t *testing.T) {
		franchises, seasons, episodes, err := scraper.Orchestrate(ctx)
		if err != nil {
			t.Fatalf("unexpected error during orchestration: %v", err)
		}

		// Verify franchises
		if len(franchises) != 1 {
			t.Fatalf("expected 1 franchise, got %d", len(franchises))
		}
		if franchises[0].Name != "RuPaul's Drag Race" || franchises[0].WikiURL != "/wiki/RuPaul%27s_Drag_Race" {
			t.Errorf("unexpected franchise: %+v", franchises[0])
		}

		// Verify seasons
		if len(seasons) != 1 {
			t.Fatalf("expected 1 season, got %d", len(seasons))
		}
		if seasons[0].Name != "Season 1" || seasons[0].Number != 1 {
			t.Errorf("unexpected season: %+v", seasons[0])
		}

		// Verify episodes
		if len(episodes) != 1 {
			t.Fatalf("expected 1 episode, got %d", len(episodes))
		}
		if episodes[0].Title != "Drag on a Dime" || episodes[0].Number != 1 {
			t.Errorf("unexpected episode: %+v", episodes[0])
		}
	})

	t.Run("individual scrape failures", func(t *testing.T) {
		// Test behavior with individual failing URLs
		badScraper := NewScraper("http://localhost:54321") // non-existent local port to force failure
		_, _, _, err := badScraper.Orchestrate(ctx)
		if err == nil {
			t.Error("expected orchestrator error on unreachable base URL, got nil")
		}
	})
}
