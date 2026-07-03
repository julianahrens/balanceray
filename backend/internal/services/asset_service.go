package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jilio/gqlgen-scalars/scalar"
	"github.com/julianahrens/balanceray/backend/internal/graph/model"
	"github.com/julianahrens/balanceray/backend/internal/repository/db"
	"github.com/shopspring/decimal"
)

type AssetService struct {
	store db.Store // Das von sqlc generierte DB-Interface (erfordert 'emit_interface: true')
	rawDB *sql.DB  // Benötigt für das Initiieren von Transaktionen
}

func NewAssetService(rawDB *sql.DB, store db.Store) *AssetService {
	return &AssetService{
		store: store,
		rawDB: rawDB,
	}
}

// CreateAsset executes a polymorphic database transaction to insert an asset and its subclass extension
func (s *AssetService) CreateAsset(ctx context.Context, input model.CreateAssetInput) (db.Asset, error) {
	var createdAsset db.Asset

	// 1. Start explicit SQL transaction
	tx, err := s.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return createdAsset, fmt.Errorf("failed to begin transaction: %w", err)
	}
	// Ensure rollback happens if any subsequent step panics or returns an error
	defer tx.Rollback()

	// 2. Bind the sqlc Queries to our active transaction instance
	qtx := s.store.WithTx(tx)

	// 3. Map values to sqlc Base Asset params
	var dbAssetClass db.AssetClass
	switch input.AssetClass {
	case model.AssetClassStock:
		dbAssetClass = db.AssetClassSTOCK
	case model.AssetClassEtf:
		dbAssetClass = db.AssetClassETF
	default:
		return createdAsset, fmt.Errorf("unsupported asset class: %s", input.AssetClass)
	}

	dec, _ := decimal.NewFromString("0.0000")
	// 4. Insert into core 'assets' table
	createdAsset, err = qtx.CreateBaseAsset(ctx, db.CreateBaseAssetParams{
		Symbol:     input.Symbol,
		Name:       input.Name,
		Currency:   input.Currency,
		AssetClass: dbAssetClass,
		LivePrice:  scalar.NewDecimal(dec), // Default baseline for new manual assets
	})
	if err != nil {
		return createdAsset, fmt.Errorf("failed to create base asset: %w", err)
	}

	// 5. Handle Polymorphic Polymorph / Child Extensions
	switch dbAssetClass {
	case db.AssetClassSTOCK:
		if input.CountryCode == nil || *input.CountryCode == "" {
			return createdAsset, fmt.Errorf("countryCode is strictly required for stock assets")
		}

		err = qtx.CreateStockExtension(ctx, db.CreateStockExtensionParams{
			AssetID:     createdAsset.ID,
			Isin:        sql.NullString{String: derefString(input.Isin), Valid: input.Isin != nil},
			Wkn:         sql.NullString{String: derefString(input.Wkn), Valid: input.Wkn != nil},
			Issuer:      sql.NullString{String: derefString(input.Issuer), Valid: input.Issuer != nil},
			CountryCode: *input.CountryCode,
		})

	case db.AssetClassETF:
		err = qtx.CreateEtfExtension(ctx, db.CreateEtfExtensionParams{
			AssetID:           createdAsset.ID,
			Isin:              sql.NullString{String: derefString(input.Isin), Valid: input.Isin != nil},
			Wkn:               sql.NullString{String: derefString(input.Wkn), Valid: input.Wkn != nil},
			Issuer:            sql.NullString{String: derefString(input.Issuer), Valid: input.Issuer != nil},
			ProviderProductID: sql.NullString{String: derefString(input.ProviderProductID), Valid: input.ProviderProductID != nil},
		})
	}

	if err != nil {
		return createdAsset, fmt.Errorf("failed to create asset subclass extension: %w", err)
	}

	// 6. Commit the entire transaction atomically
	if err := tx.Commit(); err != nil {
		return createdAsset, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return createdAsset, nil
}

func (s *AssetService) GetStockExtension(ctx context.Context, id uuid.UUID) (db.AssetsStock, error) {
	return s.store.GetStockExtensionByAssetID(ctx, id)
}

func (s *AssetService) GetEtfExtension(ctx context.Context, id uuid.UUID) (db.AssetsEtf, error) {
	return s.store.GetEtfExtensionByAssetID(ctx, id)
}

func (s *AssetService) ListAllAssets(ctx context.Context) ([]db.ListAllAssetsWithExtensionsRow, error) {
	return s.store.ListAllAssetsWithExtensions(ctx)
}

// Small inline helper to safely handle optional GraphQL strings (*string -> string)
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
