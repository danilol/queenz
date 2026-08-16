package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ParseEpisodes parses a list of episodes for a specific season from its HTML page.
func ParseEpisodes(seasonName, htmlContent string) ([]*ScrapedEpisode, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("loading HTML document: %w", err)
	}

	var episodes []*ScrapedEpisode

	// Fandom wiki episode tables typically have columns: [No. overall/in season, Title, Air date, etc.]
	doc.Find("table.wikitable tr").Each(func(i int, s *goquery.Selection) {
		// Skip header rows
		if s.Find("th").Length() > 0 {
			return
		}

		cells := s.Find("td")
		if cells.Length() >= 3 {
			// Extract episode number
			epNumIdx := 0
			rawNum := cleanString(cells.Eq(0).Text())
			num, err := strconv.Atoi(rawNum)
			if err != nil {
				// Sometimes there is a column for "No. overall" and "No. in season", let's try the second column
				epNumIdx = 1
				rawNum = cleanString(cells.Eq(1).Text())
				num, err = strconv.Atoi(rawNum)
				if err != nil {
					return // Not a valid episode row
				}
			}

			// Extract title (usually cell 1 or 2, wrapped in quotes)
			var title string
			for idx := 1; idx < cells.Length(); idx++ {
				txt := cleanString(cells.Eq(idx).Text())
				if strings.HasPrefix(txt, "\"") && strings.HasSuffix(txt, "\"") {
					title = strings.Trim(txt, "\"")
					break
				}
			}

			// Fallback: if no quoted string, use the next cell text after the episode number cell
			if title == "" {
				title = cleanString(cells.Eq(epNumIdx + 1).Text())
			}

			if title == "" || (strings.Contains(strings.ToLower(title), "episode")) {
				// Skip generic episode labels if they don't have titles yet
				title = fmt.Sprintf("Episode %d", num)
			}

			// Extract air date (look through cells starting from column 2)
			var airDate *time.Time
			var foundDate bool
			for idx := 2; idx < cells.Length(); idx++ {
				txt := cleanString(cells.Eq(idx).Text())
				if parsedDate, ok := tryParseDate(txt); ok {
					airDate = &parsedDate
					foundDate = true
					break
				}
			}

			if foundDate {
				episodes = append(episodes, &ScrapedEpisode{
					SeasonName: seasonName,
					Title:      title,
					Number:     num,
					AirDate:    airDate,
				})
			}
		}
	})

	return episodes, nil
}
