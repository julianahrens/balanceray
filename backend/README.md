# BalanceRay – Go Backend & Performance Engine

This directory contains the core backend server for BalanceRay. It is built using Go, powered by a PostgreSQL database, and exposes a strongly typed GraphQL API optimized for batch loading.

## Tech Stack & Architecture

* **Language:** Go (1.25+)
* **API Layer:** GraphQL via `gqlgen` (Schema-first approach)
* **Database Layer:** PostgreSQL with `sqlc` for compile-time safe, raw SQL query generation
* **Performance:** Custom batch DataLoaders to eliminate $N+1$ query issues across polymorph interfaces
* **Design Pattern:** Class-Table-Inheritance (CTI) separating base financial asset metrics from domain-specific data extensions (Securities, Crypto, Precious Metals)

---

## Directory Structure

```text
backend/
├── cmd/
│   └── server/             # Application entry point (main.go)
├── internal/
│   ├── dataloader/         # Batching mechanisms for efficient DB reads
│   ├── graph/              # GraphQL layer (Schema, generated code, manual resolvers)
│   │   ├── model/          # Automatically generated Go models from schema
│   │   ├── schema.graphqls # The single source of truth API Schema
│   │   └── *.resolvers.go  # Custom business logic implementation hooks
│   ├── repository/
│   │   └── db/             # sqlc generated database client and types
│   └── services/           # Core domain logic (Calculations, scrapers, data sync)
├── sql/
│   ├── migrations/         # DDL: schema.sql (Tables, Enums, Constraints)
│   └── queries/            # DML: query.sql (Raw SQL queries mapped to Go functions)
├── Dockerfile              # Multi-stage production container build
├── go.mod                  # Go module definition
└── sqlc.yaml               # Compile configuration for sqlc