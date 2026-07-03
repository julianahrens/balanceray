package db

import "database/sql"

// Store defines all database interactions, allowing easy swapping for mock implementations during testing
type Store interface {
	Querier
	WithTx(tx *sql.Tx) *Queries
}

// SQLStore implements the Store interface directly using sqlc generated code
type SQLStore struct {
	*Queries
}

func NewStore(db *sql.DB) Store {
	return &SQLStore{
		Queries: New(db),
	}
}
