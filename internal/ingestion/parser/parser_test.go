package parser

import (
	"testing"
	"time"
)

func TestParseFranchises(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="wikitable">
				<tr>
					<th>Franchise</th>
					<th>Country</th>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race">RuPaul's Drag Race</a></td>
					<td>United States</td>
				</tr>
				<tr>
					<td><a href="/wiki/Drag_Race_Espa%C3%B1a">Drag Race España</a></td>
					<td>Spain</td>
				</tr>
				<tr>
					<td>Invalid Row (no link)</td>
					<td>Nowhere</td>
				</tr>
				<tr>
					<td><a href="/external/link">External</a></td>
					<td>Other</td>
				</tr>
			</table>
		</body>
		</html>
	`

	t.Run("success", func(t *testing.T) {
		franchises, err := ParseFranchises(htmlContent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(franchises) != 2 {
			t.Errorf("expected 2 franchises, got %d", len(franchises))
		}

		if franchises[0].Name != "RuPaul's Drag Race" || franchises[0].Country != "United States" || franchises[0].WikiURL != "/wiki/RuPaul%27s_Drag_Race" {
			t.Errorf("unexpected first franchise: %+v", franchises[0])
		}

		if franchises[1].Name != "Drag Race España" || franchises[1].Country != "Spain" || franchises[1].WikiURL != "/wiki/Drag_Race_Espa%C3%B1a" {
			t.Errorf("unexpected second franchise: %+v", franchises[1])
		}
	})

	t.Run("invalid html", func(t *testing.T) {
		franchises, err := ParseFranchises("not html")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(franchises) != 0 {
			t.Errorf("expected 0 franchises, got %d", len(franchises))
		}
	})
}

func TestParseSeasons(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="wikitable">
				<tr>
					<th>Season</th>
					<th>Premiere Date</th>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race_(season_1)">Season 1</a></td>
					<td>February 2, 2009[1]</td>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race_(season_2)">Season 2</a></td>
					<td>February 1, 2010</td>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race_All_Stars">All Stars 1</a></td>
					<td>October 22, 2012</td>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race_(season_invalid)">First Season</a></td>
					<td>2009-02-02</td>
				</tr>
				<tr>
					<td><a href="/wiki/RuPaul%27s_Drag_Race_(Season_3)">The Third Season</a></td>
					<td>3 January 2011</td>
				</tr>
			</table>
		</body>
		</html>
	`

	t.Run("success", func(t *testing.T) {
		seasons, err := ParseSeasons("RuPaul's Drag Race", htmlContent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(seasons) != 3 {
			t.Errorf("expected 3 seasons, got %d", len(seasons))
		}

		expectedDate1, _ := time.Parse("2006-01-02", "2009-02-02")
		if seasons[0].Name != "Season 1" || seasons[0].Number != 1 || !seasons[0].AirDate.Equal(expectedDate1) || seasons[0].WikiURL != "/wiki/RuPaul%27s_Drag_Race_(season_1)" {
			t.Errorf("unexpected first season: %+v", seasons[0])
		}

		expectedDate2, _ := time.Parse("2006-01-02", "2010-02-01")
		if seasons[1].Name != "Season 2" || seasons[1].Number != 2 || !seasons[1].AirDate.Equal(expectedDate2) || seasons[1].WikiURL != "/wiki/RuPaul%27s_Drag_Race_(season_2)" {
			t.Errorf("unexpected second season: %+v", seasons[1])
		}

		expectedDate3, _ := time.Parse("2006-01-02", "2011-01-03")
		if seasons[2].Name != "The Third Season" || seasons[2].Number != 3 || !seasons[2].AirDate.Equal(expectedDate3) || seasons[2].WikiURL != "/wiki/RuPaul%27s_Drag_Race_(Season_3)" {
			t.Errorf("unexpected third season (href fallback): %+v", seasons[2])
		}
	})

	t.Run("invalid html", func(t *testing.T) {
		seasons, err := ParseSeasons("RuPaul's Drag Race", "not html")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(seasons) != 0 {
			t.Errorf("expected 0 seasons, got %d", len(seasons))
		}
	})
}

func TestParseEpisodes(t *testing.T) {
	htmlContent := `
		<html>
		<body>
			<table class="wikitable">
				<tr>
					<th>No. in season</th>
					<th>Title</th>
					<th>Air date</th>
				</tr>
				<tr>
					<td>1</td>
					<td>"Drag on a Dime"</td>
					<td>February 2, 2009</td>
				</tr>
				<tr>
					<td>2</td>
					<td>"Girl Groups"</td>
					<td>February 9, 2009</td>
				</tr>
				<tr>
					<td>overall</td>
					<td>3</td>
					<td>"Queens of Comedy"</td>
					<td>February 16, 2009</td>
				</tr>
				<tr>
					<td>4</td>
					<td>Episode 4</td>
					<td>February 23, 2009</td>
				</tr>
				<tr>
					<td>Invalid</td>
					<td>"No Airdate"</td>
					<td>TBA</td>
				</tr>
			</table>
		</body>
		</html>
	`

	t.Run("success", func(t *testing.T) {
		episodes, err := ParseEpisodes("Season 1", htmlContent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(episodes) != 4 {
			t.Errorf("expected 4 valid episodes, got %d", len(episodes))
		}

		expectedDate1, _ := time.Parse("2006-01-02", "2009-02-02")
		if episodes[0].Number != 1 || episodes[0].Title != "Drag on a Dime" || !episodes[0].AirDate.Equal(expectedDate1) {
			t.Errorf("unexpected first episode: %+v", episodes[0])
		}

		expectedDate2, _ := time.Parse("2006-01-02", "2009-02-09")
		if episodes[1].Number != 2 || episodes[1].Title != "Girl Groups" || !episodes[1].AirDate.Equal(expectedDate2) {
			t.Errorf("unexpected second episode: %+v", episodes[1])
		}

		expectedDate3, _ := time.Parse("2006-01-02", "2009-02-16")
		if episodes[2].Number != 3 || episodes[2].Title != "Queens of Comedy" || !episodes[2].AirDate.Equal(expectedDate3) {
			t.Errorf("unexpected third episode (two-column offset): %+v", episodes[2])
		}

		expectedDate4, _ := time.Parse("2006-01-02", "2009-02-23")
		if episodes[3].Number != 4 || episodes[3].Title != "Episode 4" || !episodes[3].AirDate.Equal(expectedDate4) {
			t.Errorf("unexpected fourth episode (generic label/title): %+v", episodes[3])
		}
	})

	t.Run("invalid html", func(t *testing.T) {
		episodes, err := ParseEpisodes("Season 1", "not html")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(episodes) != 0 {
			t.Errorf("expected 0 episodes, got %d", len(episodes))
		}
	})
}
