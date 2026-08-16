# 👑 QueenX Reboot

QueenX is a modern, highly concurrent, AI-driven platform ported from Ruby on Rails to a Go-centric stack. This repository is built as a **Domain-Driven Design (DDD) modular monolith** with high coverage tests, strict static analysis, and optimized data layers.

---

## 🏗️ Architecture & Technology Stack

The project follows **Clean Architecture** and **SOLID** engineering principles. By isolating domain logic from infrastructure details, we ensure robust testability and simple, future extraction of sub-services.

### Technology Stack
*   **Backend:** Go (1.26+)
*   **Relational Storage:** PostgreSQL 16 (via `pgx/v5` driver)
*   **Graph Storage:** Neo4j 5community (social lineages)
*   **State & Caching:** Redis 7
*   **AI Integration:** Google Gemini API & Fal.ai (Flux image gen)
*   **Frontend:** Next.js (TypeScript, TailwindCSS, Shadcn/ui)

### Project Layout
```text
queenx-go/
├── cmd/
│   ├── api/            # HTTP API Entrypoint
│   └── scraper/        # Wiki Background Scraper CLI
├── internal/
│   ├── core/           # Core Drag Context (Postgres relational data)
│   │   ├── domain/     # Pure Entities, Repository Interfaces & Domain Errors
│   │   └── repository/ # pgx Concrete Repository Implementations
│   ├── ingestion/      # Wiki Scraper & HTML Parsers (Colly-based)
│   ├── lineage/        # Graph Lineages & Social Network (Neo4j)
│   └── persona/        # AI Persona Generation (Gemini SSE & Fal.ai)
├── pkg/                # Shared internal utilities (db pools, logging, redis)
└── web/                # Next.js Presentation Layer
```

---

## 🛡️ Best Practices & Guardrails

1.  **Interfaces First:** Interfaces are defined where they are consumed (not where they are implemented), enabling clean mocking.
2.  **No Heavy ORMs:** Raw SQL is preferred via the `pgx/v5` driver to maintain query optimization, simplicity, and compile-time traceability.
3.  **Context Propagations:** `context.Context` is passed as the first parameter to every network, database, or async boundary.
4.  **No Warnings/Hacks:** Strict compiler warnings, static analysis, and strict linting (via `golangci-lint`) are strictly enforced.

---

## 🚀 Running Executables & Demos

This project contains runnable executables and CLI tools located under the `cmd/` directory.

### 1. CLI Scraper Ingestion Demo
We provide a real-time, non-interactive command-line scraper demo that fetches live data from the RuPaul's Drag Race Fandom Wiki. It showcases our asynchronous fetching pipeline, the production-ready MediaWiki JSON API bypass mechanism, and HTML parsers.

To run the scraper ingestion demo, run:
```bash
go run cmd/demo/main.go
```

### 2. Infrastructure & Tests
To spin up local PostgreSQL, Neo4j, and Redis databases or run our suite of tests and linters, please consult the detailed instructions in **[SETUP.md](./SETUP.md)**.

*(Note: As we progress to subsequent phases, instructions for starting the backend HTTP API server and Next.js frontend application will be documented here).*

---

## 📈 Development Status

- [x] **Phase 1: Foundation, Infrastructure, & Core Models**
    - Go project setup with strict v2 linting rules.
    - PostgreSQL dockerized environment.
    - Pure domain model definitions for `Franchise`, `Season`, `Episode`, and `Person`.
    - PostgreSQL-backed database repository implementations for CRUD on franchises and seasons.
    - Comprehensive unit tests (using `pgxmock` for mocking) reaching **97.7% code coverage**.
- [x] **Phase 2: Ingestion Engine (Scraping)**
    - Designed and implemented highly concurrent wiki scrapers in Go using the `Colly` framework.
    - Built decoupled, robust HTML parsers (`Franchise`, `Season`, `Episode`) under `internal/ingestion/parser` to isolate parsing from networking.
    - Implemented respectful rate limits, user-agent headers, and concurrent fan-out fetching via `errgroup`.
    - Wrote extensive table-driven unit and integration tests (using local mock `httptest` HTTP servers) achieving **96.8% coverage** in the parser package and **85.6% coverage** in the scraper package with 0 linter warnings.
- [ ] **Phase 3: Social Graph (Neo4j Integration)**
- [ ] **Phase 4: API Layer & BFF (Backend-for-Frontend)**
- [ ] **Phase 5: AI Persona Engine (Gemini & Fal.ai)**
- [ ] **Phase 6: Next.js Frontend Application**
