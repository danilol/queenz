# ⚙️ Local Development Setup Guide

Follow this guide to spin up database containers, run tests, and check strict static analysis on your local machine.

---

## 📋 Prerequisites

Before starting, make sure you have the following installed on your operating system:
*   **Go** (version 1.26 or higher)
*   **Docker** & **Docker Compose**
*   **golangci-lint** (version 2.12 or higher)

---

## 🚀 1. Infrastructure Services (Docker)

To spin up PostgreSQL, Neo4j, and Redis databases in the background, run:

```bash
docker compose up -d
```

### Database Credentials & Port Mappings

| Service | Host Port | Username / Auth | Password / Secret | Default DB / Volume |
| :--- | :--- | :--- | :--- | :--- |
| **PostgreSQL 16** | `5432` | `postgres` | `postgres_password` | `queenx_dev` |
| **Neo4j 5** | `7474` (HTTP) <br> `7687` (Bolt) | `neo4j` | `neo4j_password` | Single default graph |
| **Redis 7** | `6379` | *None* | *None* | DB 0 |

To shut down the services and preserve volume data, run:
```bash
docker compose down
```

To wipe all data volumes and reset, run:
```bash
docker compose down -v
```

---

## 🧪 2. Running the Test Suite

All data repositories and domain operations have mocked units to ensure rapid, zero-dependency testing. You do **not** need Docker running to run unit tests.

### Run All Unit Tests with Coverage
```bash
go test ./... -v -cover
```

### Generate a Visual HTML Coverage Report
If you want to view which exact lines are covered or missed:
```bash
# Generate the cover.out file
go test ./... -coverprofile=cover.out

# View the visual HTML report in your browser
go tool cover -html=cover.out
```

---

## 🛡️ 3. Linting and Formatting

We enforce extreme structural discipline using **golangci-lint (v2)**.

### Run Linting
Check your code for potential code smells, duplicate code blocks, missing error checks, and security concerns:
```bash
golangci-lint run
```

### Auto-format Imports and Files
To let the formatter automatically fix issues like import order, blank formatting, and file structures:
```bash
golangci-lint fmt
```
Alternatively, you can run the standard Go tool:
```bash
gofmt -w .
```
