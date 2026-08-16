package neo4j

import (
	"context"
	"errors"
	"fmt"
	"queenx/internal/lineage/domain"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	queenIDParam         = "queenID"
	birthPlaceParam      = "birthPlace"
	classificationsParam = "classifications"
)

// Repository implements domain.LineageRepository using Neo4j as backend.
type Repository struct {
	driver neo4j.DriverWithContext
}

// NewRepository creates a new instances of the Neo4j Lineage repository.
func NewRepository(driver neo4j.DriverWithContext) *Repository {
	return &Repository{driver: driver}
}

// executeRelationshipWrite is a reusable helper that runs write queries creating graph relationships,
// managing sessions, transactions, and translating missing target nodes into domain.ErrNotFound.
func (r *Repository) executeRelationshipWrite(ctx context.Context, query string, params map[string]any, errCtx string) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return true, nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrInvalidRelationship) {
			return err
		}
		return fmt.Errorf("%s: %w", errCtx, err)
	}
	return nil
}

// CreateQueen persists or merges a Queen node in the graph database.
func (r *Repository) CreateQueen(ctx context.Context, q *domain.Queen) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (q:Queen {id: $id})
			SET q.dragName = $dragName,
			    q.realName = $realName,
			    q.birthPlace = $birthPlace,
			    q.classifications = $classifications
			RETURN q
		`
		params := map[string]any{
			"id":                 q.ID,
			"dragName":           q.DragName,
			"realName":           q.RealName,
			birthPlaceParam:      q.BirthPlace,
			classificationsParam: q.Classifications,
		}
		res, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return fmt.Errorf("failed to create queen: %w", err)
	}
	return nil
}

// GetQueenByID retrieves a single Queen node from the graph.
func (r *Repository) GetQueenByID(ctx context.Context, id string) (*domain.Queen, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	recordVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (q:Queen {id: $id})
			RETURN q
		`
		res, err := tx.Run(ctx, query, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get queen: %w", err)
	}

	record, ok := recordVal.(*neo4j.Record)
	if !ok || record == nil {
		return nil, domain.ErrNotFound
	}

	nodeVal, found := record.Get("q")
	if !found {
		return nil, domain.ErrNotFound
	}
	node, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node, got %T", nodeVal)
	}

	props := node.GetProperties()
	return &domain.Queen{
		ID:              mapStringProp(props, "id"),
		DragName:        mapStringProp(props, "dragName"),
		RealName:        mapStringProp(props, "realName"),
		BirthPlace:      mapStringProp(props, birthPlaceParam),
		Classifications: mapStringSliceProp(props, classificationsParam),
	}, nil
}

// CreateHouse persists or merges a House node in the graph database.
func (r *Repository) CreateHouse(ctx context.Context, h *domain.House) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (h:House {id: $id})
			SET h.name = $name
			RETURN h
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"id":   h.ID,
			"name": h.Name,
		})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return fmt.Errorf("failed to create house: %w", err)
	}
	return nil
}

// GetHouseByID retrieves a single House node from the graph.
func (r *Repository) GetHouseByID(ctx context.Context, id string) (*domain.House, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	recordVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (h:House {id: $id})
			RETURN h
		`
		res, err := tx.Run(ctx, query, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get house: %w", err)
	}

	record, ok := recordVal.(*neo4j.Record)
	if !ok || record == nil {
		return nil, domain.ErrNotFound
	}

	nodeVal, found := record.Get("h")
	if !found {
		return nil, domain.ErrNotFound
	}
	node, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node, got %T", nodeVal)
	}

	props := node.GetProperties()
	return &domain.House{
		ID:   mapStringProp(props, "id"),
		Name: mapStringProp(props, "name"),
	}, nil
}

// CreateSeason persists or merges a Season node in the graph database.
func (r *Repository) CreateSeason(ctx context.Context, s *domain.Season) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MERGE (s:Season {id: $id})
			SET s.name = $name,
			    s.franchiseId = $franchiseId
			RETURN s
		`
		res, err := tx.Run(ctx, query, map[string]any{
			"id":          s.ID,
			"name":        s.Name,
			"franchiseId": s.FranchiseID,
		})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return fmt.Errorf("failed to create season: %w", err)
	}
	return nil
}

// GetSeasonByID retrieves a single Season node from the graph.
func (r *Repository) GetSeasonByID(ctx context.Context, id string) (*domain.Season, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	recordVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (s:Season {id: $id})
			RETURN s
		`
		res, err := tx.Run(ctx, query, map[string]any{"id": id})
		if err != nil {
			return nil, err
		}
		if res.Next(ctx) {
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get season: %w", err)
	}

	record, ok := recordVal.(*neo4j.Record)
	if !ok || record == nil {
		return nil, domain.ErrNotFound
	}

	nodeVal, found := record.Get("s")
	if !found {
		return nil, domain.ErrNotFound
	}
	node, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node, got %T", nodeVal)
	}

	props := node.GetProperties()
	return &domain.Season{
		ID:          mapStringProp(props, "id"),
		Name:        mapStringProp(props, "name"),
		FranchiseID: mapStringProp(props, "franchiseId"),
	}, nil
}

// AddDragMother links a mother Queen node to a daughter Queen node.
func (r *Repository) AddDragMother(ctx context.Context, motherID, daughterID string) error {
	if motherID == daughterID {
		return fmt.Errorf("%w: a queen cannot be her own mother", domain.ErrInvalidRelationship)
	}

	query := `
		MATCH (mother:Queen {id: $motherID})
		MATCH (daughter:Queen {id: $daughterID})
		MERGE (mother)-[:DRAG_MOTHER_OF]->(daughter)
		RETURN true
	`
	params := map[string]any{
		"motherID":   motherID,
		"daughterID": daughterID,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add drag mother relationship")
}

// AddSister creates a bi-directional SISTER_OF link between two Queen nodes.
func (r *Repository) AddSister(ctx context.Context, queenID1, queenID2 string) error {
	if queenID1 == queenID2 {
		return fmt.Errorf("%w: a queen cannot be her own sister", domain.ErrInvalidRelationship)
	}

	query := `
		MATCH (s1:Queen {id: $id1})
		MATCH (s2:Queen {id: $id2})
		MERGE (s1)-[:SISTER_OF]->(s2)
		MERGE (s2)-[:SISTER_OF]->(s1)
		RETURN true
	`
	params := map[string]any{
		"id1": queenID1,
		"id2": queenID2,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add sister relationship")
}

// AddHouseMember links a Queen node to a House node.
func (r *Repository) AddHouseMember(ctx context.Context, queenID, houseID string) error {
	query := `
		MATCH (q:Queen {id: $queenID})
		MATCH (h:House {id: $houseID})
		MERGE (q)-[:MEMBER_OF]->(h)
		RETURN true
	`
	params := map[string]any{
		queenIDParam: queenID,
		"houseID":    houseID,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add house member relationship")
}

// AddParticipation links a Queen node to a Season node with metadata.
func (r *Repository) AddParticipation(ctx context.Context, queenID, seasonID, placement string, wins int) error {
	query := `
		MATCH (q:Queen {id: $queenID})
		MATCH (s:Season {id: $seasonID})
		MERGE (q)-[p:PARTICIPATED_IN]->(s)
		SET p.placement = $placement,
		    p.wins = $wins
		RETURN true
	`
	params := map[string]any{
		queenIDParam: queenID,
		"seasonID":   seasonID,
		"placement":  placement,
		"wins":       wins,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add participation relationship")
}

// AddLipSync creates a bi-directional lip sync relationship between two Queen nodes.
func (r *Repository) AddLipSync(ctx context.Context, queenID1, queenID2, song, episodeID, winnerID string) error {
	if queenID1 == queenID2 {
		return fmt.Errorf("%w: a queen cannot lip sync against herself", domain.ErrInvalidRelationship)
	}

	query := `
		MATCH (q1:Queen {id: $id1})
		MATCH (q2:Queen {id: $id2})
		MERGE (q1)-[l1:LIP_SYNCED_AGAINST {song: $song, episodeId: $episodeID, winnerId: $winnerID}]->(q2)
		MERGE (q2)-[l2:LIP_SYNCED_AGAINST {song: $song, episodeId: $episodeID, winnerId: $winnerID}]->(q1)
		RETURN true
	`
	params := map[string]any{
		"id1":       queenID1,
		"id2":       queenID2,
		"song":      song,
		"episodeID": episodeID,
		"winnerID":  winnerID,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add lip sync relationship")
}

// FindAestheticSiblings finds similar queens using shared House, shared Season, shared birthPlace, and shared classifications.
func (r *Repository) FindAestheticSiblings(ctx context.Context, queenID string) ([]*domain.SiblingQueryResult, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	resultsVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (q:Queen {id: $queenID})
			MATCH (other:Queen) WHERE other.id <> q.id

			// Shared House Score
			OPTIONAL MATCH (q)-[:MEMBER_OF]->(h:House)<-[:MEMBER_OF]-(other)
			WITH q, other, CASE WHEN h IS NOT NULL THEN 10 ELSE 0 END AS houseScore, h.name AS sharedHouse

			// Shared Season Score
			OPTIONAL MATCH (q)-[:PARTICIPATED_IN]->(s:Season)<-[:PARTICIPATED_IN]-(other)
			WITH q, other, houseScore, sharedHouse, count(s) * 5 AS seasonScore

			// Shared Birth Place Score
			WITH q, other, houseScore, sharedHouse, seasonScore,
			     CASE WHEN q.birthPlace = other.birthPlace AND q.birthPlace <> "" THEN 2 ELSE 0 END AS birthPlaceScore

			// Shared Classifications Score
			WITH q, other, houseScore, sharedHouse, seasonScore, birthPlaceScore,
			     size([x IN coalesce(q.classifications, []) WHERE x IN coalesce(other.classifications, [])]) * 3 AS classificationScore

			// Total Score
			WITH other, sharedHouse, (houseScore + seasonScore + birthPlaceScore + classificationScore) AS totalScore
			WHERE totalScore > 0

			RETURN other, coalesce(sharedHouse, "") AS sharedHouse, totalScore
			ORDER BY totalScore DESC, other.dragName ASC
		`
		res, err := tx.Run(ctx, query, map[string]any{queenIDParam: queenID})
		if err != nil {
			return nil, err
		}

		var siblings []*domain.SiblingQueryResult
		for res.Next(ctx) {
			rec := res.Record()

			nodeVal, found := rec.Get("other")
			if !found {
				continue
			}
			node, ok := nodeVal.(neo4j.Node)
			if !ok {
				continue
			}

			shVal, _ := rec.Get("sharedHouse")
			sharedHouse, _ := shVal.(string)

			tsVal, _ := rec.Get("totalScore")
			totalScore, _ := tsVal.(int64)

			props := node.GetProperties()
			queen := &domain.Queen{
				ID:              mapStringProp(props, "id"),
				DragName:        mapStringProp(props, "dragName"),
				RealName:        mapStringProp(props, "realName"),
				BirthPlace:      mapStringProp(props, birthPlaceParam),
				Classifications: mapStringSliceProp(props, classificationsParam),
			}

			siblings = append(siblings, &domain.SiblingQueryResult{
				Queen:       queen,
				SharedHouse: sharedHouse,
				Score:       int(totalScore),
			})
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return siblings, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find aesthetic siblings: %w", err)
	}

	siblings, ok := resultsVal.([]*domain.SiblingQueryResult)
	if !ok {
		return nil, fmt.Errorf("unexpected results type: %T", resultsVal)
	}
	return siblings, nil
}

// Property mapping helpers

func mapStringProp(props map[string]any, key string) string {
	if val, ok := props[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func mapStringSliceProp(props map[string]any, key string) []string {
	if val, ok := props[key]; ok {
		if list, ok := val.([]any); ok {
			res := make([]string, 0, len(list))
			for _, item := range list {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		}
		if list, ok := val.([]string); ok {
			return list
		}
	}
	return nil
}
