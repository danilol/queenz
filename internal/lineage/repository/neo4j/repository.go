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

// Ensure Repository implements the domain.LineageRepository interface at compile time.
var _ domain.LineageRepository = (*Repository)(nil)

// Repository implements domain.LineageRepository using Neo4j as backend.
type Repository struct {
	driver neo4j.DriverWithContext
}

// NewRepository creates a new instances of the Neo4j Lineage repository.
func NewRepository(driver neo4j.DriverWithContext) *Repository {
	return &Repository{driver: driver}
}

// EnsureConstraints idempotently creates uniqueness constraints for the ID property of Queen, House, and Season labels in Neo4j.
func (r *Repository) EnsureConstraints(ctx context.Context) error {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	constraints := []string{
		"CREATE CONSTRAINT queen_id_unique IF NOT EXISTS FOR (q:Queen) REQUIRE q.id IS UNIQUE",
		"CREATE CONSTRAINT house_id_unique IF NOT EXISTS FOR (h:House) REQUIRE h.id IS UNIQUE",
		"CREATE CONSTRAINT season_id_unique IF NOT EXISTS FOR (s:Season) REQUIRE s.id IS UNIQUE",
	}

	for _, stmt := range constraints {
		_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
			return tx.Run(ctx, stmt, nil)
		})
		if err != nil {
			return fmt.Errorf("failed to create constraint: %w", err)
		}
	}
	return nil
}

// executeNodeRead is a generic helper that matching a single node by ID, maps its properties,
// and centralizes session handling, record/node validations, and not-found behavior.
func (r *Repository) executeNodeRead(ctx context.Context, id, label string, mapper func(map[string]any) any, errCtx string) (any, error) {
	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	recordVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := fmt.Sprintf("MATCH (n:%s {id: $id}) RETURN n", label)
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
		if errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", errCtx, err)
	}

	record, ok := recordVal.(*neo4j.Record)
	if !ok || record == nil {
		return nil, domain.ErrNotFound
	}

	nodeVal, found := record.Get("n")
	if !found {
		return nil, domain.ErrNotFound
	}
	node, ok := nodeVal.(neo4j.Node)
	if !ok {
		return nil, fmt.Errorf("expected neo4j.Node, got %T", nodeVal)
	}

	return mapper(node.GetProperties()), nil
}

// executeNodeCreate is a generic helper that creates a single node in the graph,
// centralizing write session management, transactions, constraint validation mapping, and error handling.
func (r *Repository) executeNodeCreate(ctx context.Context, query string, params map[string]any, errCtx string) error {
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
			return res.Record(), nil
		}
		if err := res.Err(); err != nil {
			return nil, err
		}
		return nil, domain.ErrNotFound
	})
	if err != nil {
		var neoErr *neo4j.Neo4jError
		if errors.As(err, &neoErr) && neoErr.Code == "Neo.ClientError.Schema.ConstraintValidationFailed" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("%s: %w", errCtx, err)
	}
	return nil
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
	query := `
		CREATE (q:Queen {
			id: $id,
			dragName: $dragName,
			realName: $realName,
			birthPlace: $birthPlace,
			classifications: $classifications
		})
		RETURN q
	`
	params := map[string]any{
		"id":                 q.ID,
		"dragName":           q.DragName,
		"realName":           q.RealName,
		birthPlaceParam:      q.BirthPlace,
		classificationsParam: q.Classifications,
	}
	return r.executeNodeCreate(ctx, query, params, "failed to create queen")
}

// GetQueenByID retrieves a single Queen node from the graph.
func (r *Repository) GetQueenByID(ctx context.Context, id string) (*domain.Queen, error) {
	mapper := func(props map[string]any) any {
		return &domain.Queen{
			ID:              mapStringProp(props, "id"),
			DragName:        mapStringProp(props, "dragName"),
			RealName:        mapStringProp(props, "realName"),
			BirthPlace:      mapStringProp(props, birthPlaceParam),
			Classifications: mapStringSliceProp(props, classificationsParam),
		}
	}
	val, err := r.executeNodeRead(ctx, id, "Queen", mapper, "failed to get queen")
	if err != nil {
		return nil, err
	}
	return val.(*domain.Queen), nil
}

// CreateHouse persists or merges a House node in the graph database.
func (r *Repository) CreateHouse(ctx context.Context, h *domain.House) error {
	query := `
		CREATE (h:House {id: $id, name: $name})
		RETURN h
	`
	params := map[string]any{
		"id":   h.ID,
		"name": h.Name,
	}
	return r.executeNodeCreate(ctx, query, params, "failed to create house")
}

// GetHouseByID retrieves a single House node from the graph.
func (r *Repository) GetHouseByID(ctx context.Context, id string) (*domain.House, error) {
	mapper := func(props map[string]any) any {
		return &domain.House{
			ID:   mapStringProp(props, "id"),
			Name: mapStringProp(props, "name"),
		}
	}
	val, err := r.executeNodeRead(ctx, id, "House", mapper, "failed to get house")
	if err != nil {
		return nil, err
	}
	return val.(*domain.House), nil
}

// CreateSeason persists or merges a Season node in the graph database.
func (r *Repository) CreateSeason(ctx context.Context, s *domain.Season) error {
	query := `
		CREATE (s:Season {id: $id, name: $name, franchiseId: $franchiseId})
		RETURN s
	`
	params := map[string]any{
		"id":          s.ID,
		"name":        s.Name,
		"franchiseId": s.FranchiseID,
	}
	return r.executeNodeCreate(ctx, query, params, "failed to create season")
}

// GetSeasonByID retrieves a single Season node from the graph.
func (r *Repository) GetSeasonByID(ctx context.Context, id string) (*domain.Season, error) {
	mapper := func(props map[string]any) any {
		return &domain.Season{
			ID:          mapStringProp(props, "id"),
			Name:        mapStringProp(props, "name"),
			FranchiseID: mapStringProp(props, "franchiseId"),
		}
	}
	val, err := r.executeNodeRead(ctx, id, "Season", mapper, "failed to get season")
	if err != nil {
		return nil, err
	}
	return val.(*domain.Season), nil
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

// AddSister creates a single undirected SISTER_OF relationship represented canonically by sorted IDs.
func (r *Repository) AddSister(ctx context.Context, queenID1, queenID2 string) error {
	if queenID1 == queenID2 {
		return fmt.Errorf("%w: a queen cannot be her own sister", domain.ErrInvalidRelationship)
	}

	id1 := queenID1
	id2 := queenID2
	if id1 > id2 {
		id1, id2 = id2, id1
	}

	query := `
		MATCH (s1:Queen {id: $id1})
		MATCH (s2:Queen {id: $id2})
		MERGE (s1)-[:SISTER_OF]->(s2)
		RETURN true
	`
	params := map[string]any{
		"id1": id1,
		"id2": id2,
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

// AddLipSync creates a single undirected LIP_SYNCED_AGAINST relationship represented canonically by sorted IDs.
func (r *Repository) AddLipSync(ctx context.Context, queenID1, queenID2, song, episodeID, winnerID string) error {
	if queenID1 == queenID2 {
		return fmt.Errorf("%w: a queen cannot lip sync against herself", domain.ErrInvalidRelationship)
	}

	id1 := queenID1
	id2 := queenID2
	if id1 > id2 {
		id1, id2 = id2, id1
	}

	query := `
		MATCH (q1:Queen {id: $id1})
		MATCH (q2:Queen {id: $id2})
		MERGE (q1)-[:LIP_SYNCED_AGAINST {song: $song, episodeId: $episodeID, winnerId: $winnerID}]->(q2)
		RETURN true
	`
	params := map[string]any{
		"id1":       id1,
		"id2":       id2,
		"song":      song,
		"episodeID": episodeID,
		"winnerID":  winnerID,
	}
	return r.executeRelationshipWrite(ctx, query, params, "failed to add lip sync relationship")
}

// FindAestheticSiblings finds similar queens using shared House, shared Season, shared birthPlace, and shared classifications.
// This query groups house connections before optional matching seasons, preventing multiple houses from multiplying seasonScore.
func (r *Repository) FindAestheticSiblings(ctx context.Context, queenID string, limit int) ([]*domain.SiblingQueryResult, error) {
	if limit <= 0 {
		limit = 100 // default limit
	}

	session := r.driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer func() {
		_ = session.Close(ctx)
	}()

	resultsVal, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		query := `
			MATCH (q:Queen {id: $queenID})
			MATCH (other:Queen) WHERE other.id <> q.id

			// Shared House Score - aggregate h first to prevent season duplication / scoring bugs
			OPTIONAL MATCH (q)-[:MEMBER_OF]->(h:House)<-[:MEMBER_OF]-(other)
			WITH q, other, 
			     CASE WHEN count(h) > 0 THEN 10 ELSE 0 END AS houseScore, 
			     coalesce(collect(h.name)[0], "") AS sharedHouse

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
			LIMIT $limit
		`
		res, err := tx.Run(ctx, query, map[string]any{
			queenIDParam: queenID,
			"limit":      limit,
		})
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
