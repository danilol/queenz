package domain

import "time"

// Franchise represents a high-level Drag Race franchise within the Core Drag Context
// (e.g., "RuPaul's Drag Race", "Drag Race España"). It acts as the root organization unit
// for seasons and contestants within a given country or region.
type Franchise struct {
	// ID is the unique UUID string identifying the franchise.
	ID string `json:"id"`
	// Name is the official title of the franchise.
	Name string `json:"name"`
	// Country represents the sovereign nation or territory hosting the franchise (e.g., "United States", "Spain").
	Country string `json:"country"`
	// CreatedAt marks when the franchise was first created in the relational repository.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt marks the last time the franchise's metadata was synchronized or edited.
	UpdatedAt time.Time `json:"updated_at"`
}

// Season represents a specific collection of episodes and contestants grouped under a Franchise.
// Each season is sequentially numbered and tracks when it first started airing.
type Season struct {
	// ID is the unique UUID identifying the season.
	ID string `json:"id"`
	// FranchiseID is the foreign key associating this season with its parent Franchise.
	FranchiseID string `json:"franchise_id"`
	// Name represents the common moniker of the season (e.g., "Season 1", "All Stars 5").
	Name string `json:"name"`
	// Number represents the chronological index of this season within its franchise.
	Number int `json:"number"`
	// AirDate indicates when the first episode of this season premiered.
	AirDate time.Time `json:"air_date"`
	// CreatedAt marks when this season was first persisted.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt marks when this season's attributes were last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// Episode represents an individual broadcasted show belonging to a specific Season.
// It tracks its ordinal position (number), standard title, and precise airdate.
type Episode struct {
	// ID is the unique UUID identifying the episode.
	ID string `json:"id"`
	// SeasonID is the foreign key associating this episode with its parent Season.
	SeasonID string `json:"season_id"`
	// Title is the official name of the episode (e.g., "The Grand Finale").
	Title string `json:"title"`
	// Number is the sequence number of this episode within its parent season.
	Number int `json:"number"`
	// AirDate is the specific calendar date the episode first broadcasted.
	AirDate time.Time `json:"air_date"`
	// CreatedAt marks when this episode was first cataloged.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt marks when this episode was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}

// Person represents an individual performer, artist, or drag queen associated with the franchise.
// It serves as the primary entity for tracking contestants, drag families, and contestant statistics.
type Person struct {
	// ID is the unique UUID identifying the person.
	ID string `json:"id"`
	// DragName is the professional stage/performing name of the artist (e.g., "Jinkx Monsoon").
	DragName string `json:"drag_name"`
	// RealName is the legal name of the artist, if publicly documented and available.
	RealName string `json:"real_name"`
	// BirthPlace indicates the city/state/country where the artist was born or resides.
	BirthPlace string `json:"birth_place"`
	// CreatedAt marks when this performer was first recorded in our system.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt marks when this performer's profile was last updated.
	UpdatedAt time.Time `json:"updated_at"`
}
