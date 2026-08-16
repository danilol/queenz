package neo4j_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"queenx/internal/lineage/domain"
	lineage_neo4j "queenx/internal/lineage/repository/neo4j"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tc_neo4j "github.com/testcontainers/testcontainers-go/modules/neo4j"
)

const (
	testTimeout = 3 * time.Minute
)

func clearDatabase(ctx context.Context, t *testing.T, driver neo4j.DriverWithContext) {
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer func() {
		_ = session.Close(ctx)
	}()

	_, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, "MATCH (n) DETACH DELETE n", nil)
		if err != nil {
			return nil, err
		}
		return nil, res.Err()
	})
	require.NoError(t, err)
}

func TestNeo4jRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	// 1. Spin up Neo4j test container
	const adminPassword = "integration-test-password"
	neo4jContainer, containerErr := tc_neo4j.Run(ctx, "neo4j:5-community",
		tc_neo4j.WithAdminPassword(adminPassword),
	)
	if containerErr != nil {
		errStr := containerErr.Error()
		isDockerMissing := strings.Contains(errStr, "docker") ||
			strings.Contains(errStr, "socket") ||
			strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "not found") ||
			strings.Contains(errStr, "daemon")
		if os.Getenv("CI") != "" || !isDockerMissing {
			t.Fatalf("Failed to start Neo4j container: %v", containerErr)
		}
		t.Skipf("Skipping integration test; Docker is likely not running: %v", containerErr)
	}
	defer func() {
		_ = neo4jContainer.Terminate(context.Background())
	}()

	// 2. Retrieve connection URL
	boltURL, urlErr := neo4jContainer.BoltUrl(ctx)
	require.NoError(t, urlErr)

	// 3. Create the Neo4j Driver
	driver, driverErr := neo4j.NewDriverWithContext(boltURL, neo4j.BasicAuth("neo4j", adminPassword, ""))
	require.NoError(t, driverErr)
	defer func() {
		_ = driver.Close(context.Background())
	}()

	// Verify connectivity
	connErr := driver.VerifyConnectivity(ctx)
	require.NoError(t, connErr)

	// Initialize Repository
	repo := lineage_neo4j.NewRepository(driver)

	t.Run("Create and Get Queen", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		q := &domain.Queen{
			ID:              "queen-1",
			DragName:        "Gigi Goode",
			RealName:        "Gigi",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen", "gothic queen"},
		}

		err := repo.CreateQueen(ctx, q)
		assert.NoError(t, err)

		retrieved, err := repo.GetQueenByID(ctx, "queen-1")
		require.NoError(t, err)
		assert.Equal(t, q.ID, retrieved.ID)
		assert.Equal(t, q.DragName, retrieved.DragName)
		assert.Equal(t, q.RealName, retrieved.RealName)
		assert.Equal(t, q.BirthPlace, retrieved.BirthPlace)
		assert.ElementsMatch(t, q.Classifications, retrieved.Classifications)
	})

	t.Run("Get Non-Existent Queen Returns NotFound", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		retrieved, err := repo.GetQueenByID(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
	})

	t.Run("Create and Get House", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		h := &domain.House{
			ID:   "house-1",
			Name: "House of Avalon",
		}

		err := repo.CreateHouse(ctx, h)
		assert.NoError(t, err)

		retrieved, err := repo.GetHouseByID(ctx, "house-1")
		require.NoError(t, err)
		assert.Equal(t, h.ID, retrieved.ID)
		assert.Equal(t, h.Name, retrieved.Name)
	})

	t.Run("Create and Get Season", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		s := &domain.Season{
			ID:          "season-1",
			Name:        "Season 12",
			FranchiseID: "franchise-1",
		}

		err := repo.CreateSeason(ctx, s)
		assert.NoError(t, err)

		retrieved, err := repo.GetSeasonByID(ctx, "season-1")
		require.NoError(t, err)
		assert.Equal(t, s.ID, retrieved.ID)
		assert.Equal(t, s.Name, retrieved.Name)
		assert.Equal(t, s.FranchiseID, retrieved.FranchiseID)
	})

	t.Run("Add Relationships and Verify ErrNotFound on Invalid Node IDs", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		// Create only one queen to make some IDs invalid and some valid
		q := &domain.Queen{
			ID:              "queen-1",
			DragName:        "Gigi Goode",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, q))

		// Mother-Daughter
		err := repo.AddDragMother(ctx, "non-existent-mother", "queen-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		err = repo.AddDragMother(ctx, "queen-1", "queen-1")
		assert.ErrorIs(t, err, domain.ErrInvalidRelationship)

		// Sister
		err = repo.AddSister(ctx, "non-existent-sister", "queen-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		err = repo.AddSister(ctx, "queen-1", "queen-1")
		assert.ErrorIs(t, err, domain.ErrInvalidRelationship)

		// House Member
		err = repo.AddHouseMember(ctx, "non-existent-queen", "house-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// Participation
		err = repo.AddParticipation(ctx, "queen-1", "non-existent-season", "Runner-up", 4)
		assert.ErrorIs(t, err, domain.ErrNotFound)

		// Lip Sync
		err = repo.AddLipSync(ctx, "queen-1", "non-existent-opponent", "Starships", "ep-1", "queen-1")
		assert.ErrorIs(t, err, domain.ErrNotFound)

		err = repo.AddLipSync(ctx, "queen-1", "queen-1", "Starships", "ep-1", "queen-1")
		assert.ErrorIs(t, err, domain.ErrInvalidRelationship)
	})

	t.Run("Add Valid Relationships Successfully", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		// Setup nodes
		q1 := &domain.Queen{
			ID:              "queen-1",
			DragName:        "Gigi Goode",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, q1))

		q2 := &domain.Queen{
			ID:              "queen-2",
			DragName:        "Symone",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, q2))

		h := &domain.House{ID: "house-1", Name: "House of Avalon"}
		require.NoError(t, repo.CreateHouse(ctx, h))

		s := &domain.Season{ID: "season-1", Name: "Season 12", FranchiseID: "franchise-1"}
		require.NoError(t, repo.CreateSeason(ctx, s))

		// Mother-daughter
		err := repo.AddDragMother(ctx, "queen-1", "queen-2")
		assert.NoError(t, err)

		// Sister
		err = repo.AddSister(ctx, "queen-1", "queen-2")
		assert.NoError(t, err)

		// House member
		err = repo.AddHouseMember(ctx, "queen-1", "house-1")
		assert.NoError(t, err)
		err = repo.AddHouseMember(ctx, "queen-2", "house-1")
		assert.NoError(t, err)

		// Participation
		err = repo.AddParticipation(ctx, "queen-1", "season-1", "Runner-up", 4)
		assert.NoError(t, err)

		// Lip Sync
		err = repo.AddLipSync(ctx, "queen-1", "queen-2", "Starships", "ep-1", "queen-2")
		assert.NoError(t, err)
	})

	t.Run("FindAestheticSiblings Complexity Test", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		// Clean / Setup Graph to represent exact specifications:
		// Target Queen: Gigi Goode ("gigi-id")
		// - House: House of Avalon ("house-avalon")
		// - Season: Season 12 ("s12")
		// - BirthPlace: "Los Angeles"
		// - Classifications: ["fashion queen", "gothic queen"]
		gigi := &domain.Queen{
			ID:              "gigi-id",
			DragName:        "Gigi Goode",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen", "gothic queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, gigi))

		avalon := &domain.House{ID: "house-avalon", Name: "House of Avalon"}
		require.NoError(t, repo.CreateHouse(ctx, avalon))
		require.NoError(t, repo.AddHouseMember(ctx, "gigi-id", "house-avalon"))

		s12 := &domain.Season{ID: "s12", Name: "Season 12", FranchiseID: "us"}
		require.NoError(t, repo.CreateSeason(ctx, s12))
		require.NoError(t, repo.AddParticipation(ctx, "gigi-id", "s12", "Runner-up", 4))

		s13 := &domain.Season{ID: "s13", Name: "Season 13", FranchiseID: "us"}
		require.NoError(t, repo.CreateSeason(ctx, s13))

		// Sibling 1: Symone ("symone-id")
		// - House: House of Avalon (10 pts)
		// - Classifications: ["fashion queen"] (1 shared * 3 = 3 pts)
		// - BirthPlace: "Los Angeles" (same = 2 pts)
		// - Season: Season 13 (no shared season = 0 pts)
		// - Total expected score: 15 pts
		symone := &domain.Queen{
			ID:              "symone-id",
			DragName:        "Symone",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"fashion queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, symone))
		require.NoError(t, repo.AddHouseMember(ctx, "symone-id", "house-avalon"))
		require.NoError(t, repo.AddParticipation(ctx, "symone-id", "s13", "Winner", 4))

		// Sibling 2: Gottmik ("gottmik-id")
		// - House: none (0 pts)
		// - Classifications: ["gothic queen", "fashion queen"] (2 shared * 3 = 6 pts)
		// - BirthPlace: "Los Angeles" (same = 2 pts)
		// - Season: Season 13 (0 pts)
		// - Total expected score: 8 pts
		gottmik := &domain.Queen{
			ID:              "gottmik-id",
			DragName:        "Gottmik",
			BirthPlace:      "Los Angeles",
			Classifications: []string{"gothic queen", "fashion queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, gottmik))
		require.NoError(t, repo.AddParticipation(ctx, "gottmik-id", "s13", "3rd/4th", 2))

		// Sibling 3: Jaida Essence Hall ("jaida-id")
		// - House: none (0 pts)
		// - Classifications: ["pageant queen"] (0 pts)
		// - BirthPlace: "Milwaukee" (0 pts)
		// - Season: Season 12 (shared season = 5 pts)
		// - Total expected score: 5 pts
		jaida := &domain.Queen{
			ID:              "jaida-id",
			DragName:        "Jaida Essence Hall",
			BirthPlace:      "Milwaukee",
			Classifications: []string{"pageant queen"},
		}
		require.NoError(t, repo.CreateQueen(ctx, jaida))
		require.NoError(t, repo.AddParticipation(ctx, "jaida-id", "s12", "Winner", 3))

		// Execute FindAestheticSiblings for Gigi Goode
		siblings, err := repo.FindAestheticSiblings(ctx, "gigi-id", 5)
		require.NoError(t, err)

		// Assertions on results (Symone should be first, Gottmik second, Jaida third)
		require.True(t, len(siblings) >= 3, "expected at least 3 siblings")

		// Verify order and correct score calculations
		var symoneResult, gottmikResult, jaidaResult *domain.SiblingQueryResult
		for _, s := range siblings {
			switch s.Queen.ID {
			case "symone-id":
				symoneResult = s
			case "gottmik-id":
				gottmikResult = s
			case "jaida-id":
				jaidaResult = s
			}
		}

		require.NotNil(t, symoneResult)
		require.NotNil(t, gottmikResult)
		require.NotNil(t, jaidaResult)

		assert.Equal(t, 15, symoneResult.Score, "Symone should have 15 points (10 House, 3 class, 2 birthPlace)")
		assert.Equal(t, 8, gottmikResult.Score, "Gottmik should have 8 points (6 class, 2 birthPlace)")
		assert.Equal(t, 5, jaidaResult.Score, "Jaida should have 5 points (5 Season)")

		// Verify the list order matches high score to low score
		assert.Equal(t, "symone-id", siblings[0].Queen.ID, "Symone should be the top aesthetic sibling")
		assert.Equal(t, "gottmik-id", siblings[1].Queen.ID, "Gottmik should be the second aesthetic sibling")
		assert.Equal(t, "jaida-id", siblings[2].Queen.ID, "Jaida should be the third aesthetic sibling")
	})

	t.Run("FindAestheticSiblings Multiple Houses Test", func(t *testing.T) {
		clearDatabase(ctx, t, driver)

		// Setup target queen and sibling queen
		target := &domain.Queen{
			ID:              "target-id",
			DragName:        "Aria",
			BirthPlace:      "New York",
			Classifications: []string{"pageant"},
		}
		require.NoError(t, repo.CreateQueen(ctx, target))

		sibling := &domain.Queen{
			ID:              "sibling-id",
			DragName:        "Bella",
			BirthPlace:      "New York",
			Classifications: []string{"pageant"},
		}
		require.NoError(t, repo.CreateQueen(ctx, sibling))

		// Create 2 houses and link BOTH queens to both houses (Shares multiple houses)
		h1 := &domain.House{ID: "house-1", Name: "House of Aria"}
		require.NoError(t, repo.CreateHouse(ctx, h1))
		require.NoError(t, repo.AddHouseMember(ctx, "target-id", "house-1"))
		require.NoError(t, repo.AddHouseMember(ctx, "sibling-id", "house-1"))

		h2 := &domain.House{ID: "house-2", Name: "House of Sparkle"}
		require.NoError(t, repo.CreateHouse(ctx, h2))
		require.NoError(t, repo.AddHouseMember(ctx, "target-id", "house-2"))
		require.NoError(t, repo.AddHouseMember(ctx, "sibling-id", "house-2"))

		// Create 1 season and link BOTH queens to it (Shares 1 season)
		s1 := &domain.Season{ID: "season-1", Name: "Season 1", FranchiseID: "us"}
		require.NoError(t, repo.CreateSeason(ctx, s1))
		require.NoError(t, repo.AddParticipation(ctx, "target-id", "season-1", "3rd", 1))
		require.NoError(t, repo.AddParticipation(ctx, "sibling-id", "season-1", "3rd", 1))

		// Run aesthetic siblings query
		siblings, err := repo.FindAestheticSiblings(ctx, "target-id", 5)
		require.NoError(t, err)

		// Verification:
		// Bella should be returned EXACTLY once (no duplication).
		require.Equal(t, 1, len(siblings), "Expected exactly 1 sibling returned")
		assert.Equal(t, "sibling-id", siblings[0].Queen.ID)

		// Correct score calculations (Grouped aggregation on house prevents multiplication/duplication):
		// - Shares at least one House: +10 pts
		// - Shares Season (1 season): +5 pts (NOT duplicated/multiplied to 10 by the 2 house matches)
		// - Same Birth Place: +2 pts
		// - Shares Classifications (1 tag): +3 pts
		// Expected Score: 10 + 5 + 2 + 3 = 20 pts
		assert.Equal(t, 20, siblings[0].Score, "Score should be exactly 20 points, with no duplicates or multiplying season scores")
	})

	t.Run("Driver Connection Failure Errors", func(t *testing.T) {
		failDriver, err := neo4j.NewDriverWithContext(boltURL, neo4j.BasicAuth("neo4j", adminPassword, ""))
		require.NoError(t, err)

		// Close it immediately to force connection failures on all queries
		_ = failDriver.Close(ctx)

		failRepo := lineage_neo4j.NewRepository(failDriver)

		err = failRepo.CreateQueen(ctx, &domain.Queen{ID: "fail"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create queen")

		_, err = failRepo.GetQueenByID(ctx, "fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get queen")

		err = failRepo.CreateHouse(ctx, &domain.House{ID: "fail"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create house")

		_, err = failRepo.GetHouseByID(ctx, "fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get house")

		err = failRepo.CreateSeason(ctx, &domain.Season{ID: "fail"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create season")

		_, err = failRepo.GetSeasonByID(ctx, "fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get season")

		err = failRepo.AddDragMother(ctx, "fail-mother", "fail-daughter")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add drag mother relationship")

		err = failRepo.AddSister(ctx, "fail-sister1", "fail-sister2")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add sister relationship")

		err = failRepo.AddHouseMember(ctx, "fail", "fail")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add house member relationship")

		err = failRepo.AddParticipation(ctx, "fail", "fail", "Placement", 0)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add participation relationship")

		err = failRepo.AddLipSync(ctx, "fail-1", "fail-2", "Song", "ep", "fail-1")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to add lip sync relationship")

		_, err = failRepo.FindAestheticSiblings(ctx, "fail", 5)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to find aesthetic siblings")
	})
}
