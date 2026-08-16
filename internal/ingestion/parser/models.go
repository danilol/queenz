package parser

import "time"

// ScrapedFranchise represents raw scraped data for a franchise.
type ScrapedFranchise struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	WikiURL string `json:"wiki_url"`
}

// ScrapedSeason represents raw scraped data for a season under a franchise.
type ScrapedSeason struct {
	FranchiseName string     `json:"franchise_name"`
	Name          string     `json:"name"`
	Number        int        `json:"number"`
	AirDate       *time.Time `json:"air_date"`
	WikiURL       string     `json:"wiki_url"`
}

// ScrapedEpisode represents raw scraped data for an episode.
type ScrapedEpisode struct {
	SeasonName string     `json:"season_name"`
	Title      string     `json:"title"`
	Number     int        `json:"number"`
	AirDate    *time.Time `json:"air_date"`
}
