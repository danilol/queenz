package parser

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ParseFranchises parses a list of franchises from a Wiki HTML page.
// It scans tables with class "wikitable" for adaptation names, countries, and Wiki links.
func ParseFranchises(htmlContent string) ([]*ScrapedFranchise, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("loading HTML document: %w", err)
	}

	var franchises []*ScrapedFranchise

	doc.Find("table.wikitable tr").Each(func(i int, s *goquery.Selection) {
		// Skip header rows
		if s.Find("th").Length() > 0 {
			return
		}

		cells := s.Find("td")
		if cells.Length() >= 2 {
			link := cells.Eq(0).Find("a").First()
			name := strings.TrimSpace(link.Text())
			href, exists := link.Attr("href")
			country := strings.TrimSpace(cells.Eq(1).Text())

			// Only ingest if we have a name, country, and valid URL path
			if name != "" && exists && country != "" && strings.HasPrefix(href, "/wiki/") {
				franchises = append(franchises, &ScrapedFranchise{
					Name:    name,
					Country: country,
					WikiURL: href,
				})
			}
		}
	})

	return franchises, nil
}
