package parser

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var seasonNumRegex = regexp.MustCompile(`(?i)Season[\s_-]*(\d+)`)
var footnoteRegex = regexp.MustCompile(`\[\d+\]`)

// ParseSeasons parses seasons for a specific franchise from its HTML page.
func ParseSeasons(franchiseName, htmlContent string) ([]*ScrapedSeason, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("loading HTML document: %w", err)
	}

	var seasons []*ScrapedSeason

	// Scan all wikitable rows for season info
	doc.Find("table.wikitable tr").Each(func(i int, s *goquery.Selection) {
		if s.Find("th").Length() > 0 {
			return
		}

		cells := s.Find("td")
		if cells.Length() >= 2 {
			link := cells.Eq(0).Find("a").First()
			name := strings.TrimSpace(link.Text())
			href, exists := link.Attr("href")

			// Check if the link contains Season information
			if exists && strings.HasPrefix(href, "/wiki/") && (strings.Contains(strings.ToLower(href), "season") || strings.Contains(strings.ToLower(name), "season")) {
				// Try to extract season number
				number := extractSeasonNumber(name, href)
				if number == 0 {
					return
				}

				// Find air date (usually in subsequent columns, let's look for first parseable date)
				var airDate *time.Time
				var foundDate bool
				for idx := 1; idx < cells.Length(); idx++ {
					cleanedText := cleanString(cells.Eq(idx).Text())
					if parsedDate, ok := tryParseDate(cleanedText); ok {
						airDate = &parsedDate
						foundDate = true
						break
					}
				}

				if foundDate {
					seasons = append(seasons, &ScrapedSeason{
						FranchiseName: franchiseName,
						Name:          name,
						Number:        number,
						AirDate:       airDate,
						WikiURL:       href,
					})
				}
			}
		}
	})

	return seasons, nil
}

func extractSeasonNumber(name, href string) int {
	// Try name first (e.g. "Season 1")
	matches := seasonNumRegex.FindStringSubmatch(name)
	if len(matches) == 2 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return val
		}
	}

	// Try href (e.g. "/wiki/RuPaul%27s_Drag_Race_(season_1)")
	decoded, err := url.PathUnescape(href)
	if err != nil {
		return 0
	}
	matches = seasonNumRegex.FindStringSubmatch(decoded)
	if len(matches) == 2 {
		if val, err := strconv.Atoi(matches[1]); err == nil {
			return val
		}
	}

	return 0
}

func cleanString(val string) string {
	// Remove footnotes like [1] or references, trailing newlines
	cleaned := footnoteRegex.ReplaceAllString(val, "")
	return strings.TrimSpace(cleaned)
}

func tryParseDate(raw string) (time.Time, bool) {
	formats := []string{
		"January 2, 2006",
		"2 January 2006",
		"2006-01-02",
		"January 2006", // fallback
	}
	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
