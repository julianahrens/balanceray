package graph

import (
	"context"
	"database/sql"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// nullStringToPointer converts a database sql.NullString into a clean *string for GraphQL
func nullStringToPointer(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// PresentValidationError maps structured ozzo-validation errors into official GraphQL Error Extensions
func PresentValidationError(ctx context.Context, err error) error {
	var valErrors validation.Errors
	if errors.As(err, &valErrors) {
		// Create a map specifically for the frontend field-bindings
		fieldExtensions := make(map[string]interface{})
		for field, fieldErr := range valErrors {
			fieldExtensions[field] = fieldErr.Error()
		}

		return &gqlerror.Error{
			Message: "Input validation failed",
			Path:    graphql.GetPath(ctx),
			Extensions: map[string]interface{}{
				"code":   "VALIDATION_FAILED",
				"fields": fieldExtensions, // Contains exact mappings like {"currency": "the length must be exactly 3"}
			},
		}
	}

	// Fallback for regular database or system errors
	return gqlerror.Errorf("internal server or database error: %s", err.Error())
}
