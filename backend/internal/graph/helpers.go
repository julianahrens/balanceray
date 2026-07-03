package graph

import "database/sql"

// nullStringToPointer converts a database sql.NullString into a clean *string for GraphQL
func nullStringToPointer(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}
