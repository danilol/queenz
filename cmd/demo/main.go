package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"queenx/internal/ingestion"
)

func main() {
	if err := runDemo(); err != nil {
		fmt.Printf("❌ Demo failed with error: %v\n", err)
		os.Exit(1)
	}
}

func runDemo() error {
	fmt.Println("👑 =========================================== 👑")
	fmt.Println("👑       QUEENX CLI SCRAPER INGESTION DEMO     👑")
	fmt.Println("👑 =========================================== 👑")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize scraper pointing to the live wiki
	scraper := ingestion.NewScraper("https://rupaulsdragrace.fandom.com")

	// 1. Live Scrape Franchises
	fmt.Println("🔍 1. Scraping Franchise Adaptations from live Wiki...")
	startTime := time.Now()
	franchises, err := scraper.ScrapeFranchises(ctx)
	if err != nil {
		return fmt.Errorf("scraping franchises: %w", err)
	}
	fmt.Printf("✅ Scraped %d franchises in %v!\n\n", len(franchises), time.Since(startTime))

	// Display first 5 franchises
	fmt.Println("📋 Top Franchises found on Wiki:")
	displayLimit := 5
	if len(franchises) < displayLimit {
		displayLimit = len(franchises)
	}
	for i := 0; i < displayLimit; i++ {
		f := franchises[i]
		fmt.Printf("   [%d] Name: %-30s | Country: %-15s | Path: %s\n", i+1, f.Name, f.Country, f.WikiURL)
	}
	if len(franchises) > displayLimit {
		fmt.Printf("   ... and %d more!\n", len(franchises)-displayLimit)
	}
	fmt.Println()

	if len(franchises) == 0 {
		fmt.Println("⚠️ No franchises found. Ending demo.")
		return nil
	}

	// 2. Select a franchise to scrape its seasons
	selectedFranchise := franchises[0]
	for _, f := range franchises {
		if f.Name == "RuPaul's Drag Race" {
			selectedFranchise = f
			break
		}
	}

	fmt.Printf("🔍 2. Scraping Seasons for '%s' (%s)...\n", selectedFranchise.Name, selectedFranchise.WikiURL)
	startTime = time.Now()
	seasons, err := scraper.ScrapeSeasons(ctx, selectedFranchise)
	if err != nil {
		return fmt.Errorf("scraping seasons: %w", err)
	}
	fmt.Printf("✅ Scraped %d seasons in %v!\n\n", len(seasons), time.Since(startTime))

	// Display seasons
	displayLimit = 5
	if len(seasons) < displayLimit {
		displayLimit = len(seasons)
	}
	for i := 0; i < displayLimit; i++ {
		s := seasons[i]
		airDateStr := "TBA"
		if !s.AirDate.IsZero() {
			airDateStr = s.AirDate.Format("January 2, 2006")
		}
		fmt.Printf("   Season %-2d | Name: %-15s | Air Date: %-18s | Path: %s\n", s.Number, s.Name, airDateStr, s.WikiURL)
	}
	if len(seasons) > displayLimit {
		fmt.Printf("   ... and %d more!\n", len(seasons)-displayLimit)
	}
	fmt.Println()

	if len(seasons) == 0 {
		fmt.Println("⚠️ No seasons found for this franchise. Ending demo.")
		return nil
	}

	// 3. Select a season to scrape its episodes (e.g. Season 1 or Season 2)
	selectedSeason := seasons[0]
	for _, s := range seasons {
		if s.Number == 1 {
			selectedSeason = s
			break
		}
	}

	fmt.Printf("🔍 3. Scraping Episode guide for '%s > %s' (%s)...\n", selectedFranchise.Name, selectedSeason.Name, selectedSeason.WikiURL)
	startTime = time.Now()
	episodes, err := scraper.ScrapeEpisodes(ctx, selectedSeason)
	if err != nil {
		return fmt.Errorf("scraping episodes: %w", err)
	}
	fmt.Printf("✅ Scraped %d episodes in %v!\n\n", len(episodes), time.Since(startTime))

	// Display episodes
	fmt.Printf("🎬 Episode Guide for '%s':\n", selectedSeason.Name)
	for _, ep := range episodes {
		airDateStr := "TBA"
		if !ep.AirDate.IsZero() {
			airDateStr = ep.AirDate.Format("Jan 02, 2006")
		}
		fmt.Printf("   Episode %-2d | Title: %-30s | Air Date: %s\n", ep.Number, ep.Title, airDateStr)
	}
	fmt.Println()

	fmt.Println("👑 =========================================== 👑")
	fmt.Println("👑         DEMO COMPLETED SUCCESSFULLY!        👑")
	fmt.Println("👑 =========================================== 👑")
	return nil
}
