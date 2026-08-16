# 🕸️ Lineage Context: Graph Databases & Cypher Social Traversal

The **Lineage Context** (`internal/lineage/`) is a high-performance graph database engine powered by **Neo4j 5** and written in Go using the official driver. It is designed to track complex drag family genealogies, drag house memberships, television season participations, lip-sync performance histories, and evaluate contestant aesthetic similarity.

---

## 🏗️ Architectural Overview

Following **Domain-Driven Design (DDD)** and **Clean Architecture** principles, the Lineage Context is completely decoupled into:
- **`domain/`**: Technology-agnostic entities, custom graph-related errors, and the repository interface definition.
- **`repository/neo4j`**: A concrete database adapter implementing the `LineageRepository` using Neo4j and Cypher queries.

### 📊 Graph Nodes & Properties

1. **`Queen` Node**
   - Label: `:Queen`
   - Properties:
     - `id` (String, Unique)
     - `dragName` (String)
     - `realName` (String)
     - `birthPlace` (String)
     - `classifications` (List of Strings, e.g. `["fashion queen", "comedy queen"]`)

2. **`House` Node**
   - Label: `:House`
   - Properties:
     - `id` (String, Unique)
     - `name` (String)

3. **`Season` Node**
   - Label: `:Season`
   - Properties:
     - `id` (String, Unique)
     - `name` (String)
     - `franchiseId` (String)

---

## 🔗 Graph Relationships (Edges)

1. **`DRAG_MOTHER_OF`** (Directed)
   - Connection: `(:Queen)-[:DRAG_MOTHER_OF]->(:Queen)`
   - Meaning: Mentorship, genealogical heritage (e.g., Alyssa Edwards to Shangela).

2. **`SISTER_OF`** (Bi-directional)
   - Connection: `(:Queen)-[:SISTER_OF]-(:Queen)`
   - Meaning: Sisters within the same generation.

3. **`MEMBER_OF`** (Directed)
   - Connection: `(:Queen)-[:MEMBER_OF]->(:House)`
   - Meaning: Performance House affiliation (e.g., Symone to House of Avalon).

4. **`PARTICIPATED_IN`** (Directed, with Properties)
   - Connection: `(:Queen)-[p:PARTICIPATED_IN]->(:Season)`
   - Properties:
     - `placement` (String, e.g., `"Runner-up"`, `"Winner"`)
     - `wins` (Integer)

5. **`LIP_SYNCED_AGAINST`** (Bi-directional, with Properties)
   - Connection: `(:Queen)-[l:LIP_SYNCED_AGAINST]-(:Queen)`
   - Properties:
     - `song` (String)
     - `episodeId` (String)
     - `winnerId` (String)

---

## 🧮 Aesthetic Similarity scoring (`FindAestheticSiblings`)

A primary feature of the Lineage Context is finding "Aesthetic Siblings" utilizing real-time scoring in Cypher. The scoring engine evaluates potential siblings using four distinct dimensions:

| Dimension | Rule | Points |
| :--- | :--- | :--- |
| **Shared House** | If they are members of the exact same `:House` | **+10 pts** |
| **Shared Season** | For each television `:Season` they competed in together | **+5 pts** |
| **Shared Birth Place** | If their birthplace string matches exactly (and is non-empty) | **+2 pts** |
| **Shared Classifications** | For each overlapping category tag they share (e.g. `"fashion queen"`) | **+3 pts per tag** |

### Optimized Cypher Traversal Query

```cypher
MATCH (q:Queen {id: $queenID})
MATCH (other:Queen) WHERE other.id <> q.id

// 1. Shared House Score
OPTIONAL MATCH (q)-[:MEMBER_OF]->(h:House)<-[:MEMBER_OF]-(other)
WITH q, other, CASE WHEN h IS NOT NULL THEN 10 ELSE 0 END AS houseScore, h.name AS sharedHouse

// 2. Shared Season Score
OPTIONAL MATCH (q)-[:PARTICIPATED_IN]->(s:Season)<-[:PARTICIPATED_IN]-(other)
WITH q, other, houseScore, sharedHouse, count(s) * 5 AS seasonScore

// 3. Shared Birth Place Score
WITH q, other, houseScore, sharedHouse, seasonScore,
     CASE WHEN q.birthPlace = other.birthPlace AND q.birthPlace <> "" THEN 2 ELSE 0 END AS birthPlaceScore

// 4. Shared Classifications Score
WITH q, other, houseScore, sharedHouse, seasonScore, birthPlaceScore,
     size([x IN coalesce(q.classifications, []) WHERE x IN coalesce(other.classifications, [])]) * 3 AS classificationScore

// 5. Total Score Evaluation
WITH other, sharedHouse, (houseScore + seasonScore + birthPlaceScore + classificationScore) AS totalScore
WHERE totalScore > 0

RETURN other, coalesce(sharedHouse, "") AS sharedHouse, totalScore
ORDER BY totalScore DESC, other.dragName ASC
```

---

## 🧪 Integration Testing with `testcontainers-go`

Integration testing requires a real Neo4j instance to validate Cypher constraints and query structures. We use **`testcontainers-go`** to run programmatic containers automatically.

### Running the Suite
```bash
go test ./internal/lineage/... -v
```

- **Database Cleanup Isolation**: The suite features an automatic `clearDatabase` helper running `MATCH (n) DETACH DELETE n` before each sub-test to ensure pristine state isolation.
- **Verification Coverage**: The suite validates the scoring weight equations explicitly, ensuring that the descending order perfectly aligns with mathematical scoring.
