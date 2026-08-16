# 👑 QUEENX: The Ultimate Reboot Execution Plan

This is the master blueprint and execution guide for transitioning QueenX from Ruby on Rails to a modern, highly concurrent, AI-driven stack: **Go (Backend), Next.js (Frontend), PostgreSQL (Relational Data), Neo4j (Graph Data), and Redis (State & Caching)**.

This plan enforces **Domain-Driven Design (DDD)**, **Clean Architecture**, **Single Responsibility Principle (SRP)**, and **Test-Driven Development (TDD) / High Coverage**.

---

## 🏗️ 1. Architecture & Design Principles

### Domain-Driven Design (DDD) Boundaries
We will separate the system into distinct Bounded Contexts. Sub-services will be logically isolated within the Go monolith (modular monolith approach) to allow future extraction into microservices if needed.

1.  **Ingestion Context (Scraper):** Responsible solely for fetching HTML/API data from Fandom and saving raw/parsed payloads.
2.  **Core Drag Context (Relational):** Manages Franchises, Seasons, Episodes, and Contestants (Postgres).
3.  **Lineage Context (Graph):** Manages the drag family trees, house affiliations, and rivalries (Neo4j).
4.  **AI Persona Context:** Handles the "Create Your Drag Queen" logic, orchestrating LLM streams (Gemini) and Image Generation (Fal.ai).

### Project Layout (Go Modular Monolith)
```text
queenx-go/
├── cmd/
│   ├── api/            # Main HTTP API entrypoint
│   └── scraper/        # CLI entrypoint for background scraping tasks
├── internal/
│   ├── ingestion/      # Colly scrapers, parsing logic
│   ├── core/           # Postgres models, repositories, business logic
│   ├── lineage/        # Neo4j graph operations, Cypher queries
│   └── persona/        # AI logic, Gemini API, Fal.ai integration
├── pkg/                # Shared utilities (logger, db clients, redis)
└── web/                # Next.js Frontend application
```

---

## 🚀 2. Phase-by-Phase Execution Plan & AI Prompts

Use the following phases and prompts to instruct an AI coding assistant (like Gemini, Claude, or Devin) to build out the project incrementally.

### 🟡 Phase 1: Foundation, Infrastructure, & Core Models
**Goal:** Setup the Go environment, Docker Compose for the databases, and implement the PostgreSQL repository layer for the Core Drag Context.

*   **Setup:** Initialize `go mod`, configure `.golangci.yml` for strict linting, setup `docker-compose.yml` (Postgres, Neo4j, Redis).
*   **Domain:** Define Entities (Franchise, Season, Episode, Person) in Go structs.
*   **Data Access:** Implement the Repository pattern using `pgx` (or `sqlc` for type-safe SQL).

> **🤖 AI Prompt 1: Project Setup & Core Context**
> "Act as a Senior Go Systems Architect. Initialize a new Go project named `queenx-go`. Create a `docker-compose.yml` containing PostgreSQL 16, Neo4j 5, and Redis 7. Then, following Domain-Driven Design (DDD), create the 'Core Drag Context' inside `internal/core`. Define the Domain Entities (Franchise, Season, Episode, Person) as pure Go structs. Finally, implement a PostgreSQL repository interface and its concrete implementation using `pgx` to perform basic CRUD operations for Franchises and Seasons. Ensure separation of concerns (no DB logic in the domain models) and write 100% coverage unit tests using Go's standard `testing` package and dependency injection for mocking."

### 🟠 Phase 2: The Ingestion Engine (Scraping)
**Goal:** Port the Ruby/Capybara scraper to a high-concurrency Go scraper using `colly`.

*   **Sub-service:** `internal/ingestion`
*   **Logic:** Implement dual-source fetching (MediaWiki API and raw HTML).
*   **Concurrency:** Use goroutines and Colly's asynchronous features to fan out Franchise -> Season -> Episode scrapes.

> **🤖 AI Prompt 2: Highly Concurrent Scraping Engine**
> "Act as a Senior Go Data Engineer. We are building the 'Ingestion Context' inside `internal/ingestion` using the `gocolly/colly` framework. Create a scraper orchestrator that fetches data from the RuPaul's Drag Race Fandom wiki. Implement the Single Responsibility Principle: create separate parsers for Franchises, Seasons, and Episodes. The scraper should utilize Go's concurrency to fan out requests while respecting a rate limit (using Colly's LimitRule). It must return parsed Domain Entities. Do not tightly couple this to the database; it should return data structures that the Core context can ingest. Write extensive table-driven tests for the HTML parsing logic."

### 🔴 Phase 3: The Social Graph (Neo4j Integration)
**Goal:** Model relationships, drag families, and track records in a Graph Database.

*   **Sub-service:** `internal/lineage`
*   **Logic:** Translate Core relational data into Graph Nodes (`Queen`, `Season`) and Edges (`LIP_SYNCED_AGAINST`, `DRAG_MOTHER_OF`, `SISTER_OF`).

> **🤖 AI Prompt 3: Graph Database & Cypher Integration**
> "Act as a Senior Graph Database Expert and Go Developer. Implement the 'Lineage Context' in `internal/lineage` using the official Neo4j Go driver. We need to model the social graph of Drag Queens. Create repository methods that execute Cypher queries to create Nodes (Queen, House) and Relationships (DRAG_MOTHER_OF, MEMBER_OF, LIP_SYNCED_AGAINST). Write a specific function `FindAestheticSiblings(queenID string)` that traverses the graph to find queens in the same house or with overlapping traits. Ensure the driver is injected as a dependency and write integration tests using Testcontainers for Neo4j."

### 🟣 Phase 4: API Layer & BFF (Backend-for-Frontend)
**Goal:** Expose the data to the frontend via a fast HTTP router.

*   **Sub-service:** `cmd/api` and `internal/api`
*   **Framework:** `Echo` or `Gin`.
*   **Logic:** RESTful endpoints for exploring franchises, seasons, and queens.

> **🤖 AI Prompt 4: High-Performance HTTP API**
> "Act as a Backend API Specialist. Implement the HTTP presentation layer in `cmd/api` using the LabStack Echo framework. Create RESTful endpoints to serve data from the `core` and `lineage` contexts (e.g., `GET /api/v1/franchises`, `GET /api/v1/queens/:id/lineage`). Implement middleware for Redis-based rate limiting and structured JSON logging (using `slog`). Follow clean architecture: the HTTP handlers should only handle request/response parsing and delegate business logic to injected Domain Services. Write HTTP tests using Go's `httptest` package."

### 🔵 Phase 5: AI Persona Engine (The Crown Jewel)
**Goal:** Implement the "Create Your Drag Queen" feature utilizing Gemini for streaming text and Fal.ai (Flux) for image generation.

*   **Sub-service:** `internal/persona`
*   **Logic:**
    1. Receive user traits.
    2. Query Postgres/Neo4j for drag context.
    3. Construct LLM prompt.
    4. Stream Gemini response via Server-Sent Events (SSE).
    5. Trigger Fal.ai image generation asynchronously and push URL via SSE.

> **🤖 AI Prompt 5: AI Orchestration & Streaming (SSE)**
> "Act as a Lead AI Integration Engineer. Implement the 'Persona Context' in `internal/persona`. We need to build the 'Create Your Drag Queen' feature. Create a service that takes user input traits, queries the `lineage` context for similar queens to inject as context, and builds a system prompt for the Google Gemini API. Implement an HTTP handler in Echo that opens a Server-Sent Events (SSE) connection to the client. Stream the Gemini generated bio, drag name, and stats back to the client in real-time. Once the text stream is finished, use the LLM's 'image_generation_prompt' to make a POST request to the Fal.ai Flux API, wait for the image URL, and push that final URL down the SSE stream. Handle contexts, timeouts, and cancellations gracefully."

### 🟢 Phase 6: Next.js Interactive Frontend
**Goal:** Build the sleek, interactive UI.

*   **Stack:** Next.js (App Router), React, TypeScript, TailwindCSS, Shadcn/ui.
*   **Logic:** Build a visually stunning dashboard to explore the wiki data, and a highly interactive "Audition" wizard to consume the AI SSE stream.

> **🤖 AI Prompt 6: Next.js Frontend & SSE Consumer**
> "Act as a Senior Frontend Architect. Initialize a Next.js (App Router) project with TypeScript, TailwindCSS, and Shadcn/ui in the `web/` directory. Build the 'Create Your Drag Queen' interactive audition flow. Create a beautiful, multi-step form for users to select their traits. Then, build a 'Reveal' component that connects to the Go backend's Server-Sent Events (SSE) endpoint using the native EventSource API. As the JSON chunks stream in, animate the typing of the drag queen's bio and stats. When the final event containing the image URL arrives, transition the UI to reveal the high-glamour drag portrait."

---

## 🛡️ Best Practices & Guardrails for the Reboot

1.  **Interfaces First:** In Go, define interfaces where they are consumed, not where they are implemented. This allows for trivial mocking of the Neo4j, Postgres, and Gemini APIs during testing.
2.  **No ORMs:** Avoid heavy ORMs like GORM. Use standard SQL with `pgx` or `sqlc`. For Neo4j, use raw Cypher queries. It keeps the abstraction clean and queries optimized.
3.  **Strict Error Handling:** Do not ignore errors. Wrap errors with context (`fmt.Errorf("fetching queen: %w", err)`) to ensure trace logs are actually useful.
4.  **Graceful Shutdown:** The Go API and Scraper must implement context-based graceful shutdowns to prevent orphaned background jobs or corrupted database states.
5.  **Context Passing:** `context.Context` must be the first argument of every function that crosses a network or database boundary.

This plan gives you a deterministic, phased approach to rewriting QueenX into a modern, scalable, AI-powered platform.
