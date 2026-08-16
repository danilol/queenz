package domain

import "context"

// LineageRepository defines the graph-store operations for managing
// nodes (Queens, Houses, Seasons) and their relationships in the Lineage Context.
type LineageRepository interface {
	// Node Management
	CreateQueen(ctx context.Context, q *Queen) error
	GetQueenByID(ctx context.Context, id string) (*Queen, error)
	CreateHouse(ctx context.Context, h *House) error
	GetHouseByID(ctx context.Context, id string) (*House, error)
	CreateSeason(ctx context.Context, s *Season) error
	GetSeasonByID(ctx context.Context, id string) (*Season, error)

	// Relationship Management
	AddDragMother(ctx context.Context, motherID, daughterID string) error
	AddSister(ctx context.Context, queenID1, queenID2 string) error
	AddHouseMember(ctx context.Context, queenID, houseID string) error
	AddParticipation(ctx context.Context, queenID, seasonID string, placement string, wins int) error
	AddLipSync(ctx context.Context, queenID1, queenID2 string, song, episodeID, winnerID string) error

	// Graph Traversals
	// FindAestheticSiblings scores siblings based on shared Houses, Seasons, birthPlace, and classifications.
	FindAestheticSiblings(ctx context.Context, queenID string) ([]*SiblingQueryResult, error)
}
