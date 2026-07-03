package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/julianahrens/balanceray/backend/internal/graph"
	"github.com/julianahrens/balanceray/backend/internal/repository/db"
	"github.com/julianahrens/balanceray/backend/internal/services"
	"github.com/vektah/gqlparser/v2/ast"

	_ "github.com/lib/pq"
)

const defaultPort = "8080"

func main() {
	connStr := "postgres://balanceray_user:balanceray_password@localhost:5432/balanceray_dev?sslmode=disable"
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer conn.Close()

	if err := db.RunMigrations(conn); err != nil {
		log.Fatalf("Could not run database migrations: %v", err)
	}

	// 1. Initialize store
	store := db.NewStore(conn)

	// 2. Initialize domain services
	assetService := services.NewAssetService(conn, store)

	// 3. Inject services into the central GraphQL Resolver struct
	resolver := &graph.Resolver{
		AssetService: assetService,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: resolver}))

	srv.AddTransport(transport.Options{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.POST{})

	srv.SetQueryCache(lru.New[*ast.QueryDocument](1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](100),
	})

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
