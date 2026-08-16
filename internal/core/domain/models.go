package domain

import "time"

// Franchise represents a Drag Race franchise (e.g., RuPaul's Drag Race, Drag Race España).
type Franchise struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Country   string    `json:"country"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Season represents a specific season of a Franchise.
type Season struct {
	ID          string    `json:"id"`
	FranchiseID string    `json:"franchise_id"`
	Name        string    `json:"name"`
	Number      int       `json:"number"`
	AirDate     time.Time `json:"air_date"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Episode represents a specific episode within a Season.
type Episode struct {
	ID        string    `json:"id"`
	SeasonID  string    `json:"season_id"`
	Title     string    `json:"title"`
	Number    int       `json:"number"`
	AirDate   time.Time `json:"air_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Person represents a contestant / drag queen.
type Person struct {
	ID         string    `json:"id"`
	DragName   string    `json:"drag_name"`
	RealName   string    `json:"real_name"`
	BirthPlace string    `json:"birth_place"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
