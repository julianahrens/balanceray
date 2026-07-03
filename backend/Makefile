# Detect operating system to set the correct volume mount path syntax
ifeq ($(OS),Windows_NT)
    CUR_DIR := $(shell cd)
else
    CUR_DIR := $(shell pwd)
endif

.PHONY: generate

generate:
	@echo "🔄 Generating database models with sqlc via Docker..."
	docker run --rm -v "$(CUR_DIR):/src" -w /src sqlc/sqlc generate
	@echo "🔄 Generating GraphQL models and resolvers with gqlgen..."
	go run github.com/99designs/gqlgen generate
	@echo "✅ All code generation complete!"
