package domain

// Queen represents a drag queen node within the lineage social graph.
type Queen struct {
	ID              string   `json:"id"`
	DragName        string   `json:"dragName"`
	RealName        string   `json:"realName"`
	BirthPlace      string   `json:"birthPlace"`
	Classifications []string `json:"classifications"` // e.g. ["lip sync assassin", "fashion queen"]
}

// House represents a drag house or family organization node (e.g., "House of Edwards").
type House struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Season represents a Season node in the graph, used to connect contestants to their seasons.
type Season struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FranchiseID string `json:"franchiseId"`
}

// SiblingQueryResult represents a match from the aesthetic sibling query.
type SiblingQueryResult struct {
	Queen       *Queen `json:"queen"`
	SharedHouse string `json:"sharedHouse"`
	Score       int    `json:"score"`
}
