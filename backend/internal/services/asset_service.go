package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/jilio/gqlgen-scalars/scalar"
	"github.com/julianahrens/balanceray/backend/internal/graph/model"
	"github.com/julianahrens/balanceray/backend/internal/repository/db"
	"github.com/pariz/gountries"
	"github.com/shopspring/decimal"
)

// FetchJob holds the metadata needed by the background worker
type FetchJob struct {
	AssetID    uuid.UUID
	Symbol     string
	AssetClass db.AssetClass
}

type YahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// startPriceFetchWorker runs indefinitely in a separate goroutine
func (s *AssetService) startPriceFetchWorker() {
	log.Println("🚀 Asset Price Fetch Worker started successfully...")

	for job := range s.jobQueue {
		ctx := context.Background()

		log.Printf("📥 [Worker] Fetching prices for %s (%s)...", job.Symbol, job.AssetClass)

		livePrice, err := s.fetchExternalLivePrice(job.Symbol)
		if err != nil {
			log.Printf("❌ [Worker] Failed to fetch live price for %s: %v", job.Symbol, err)
			continue
		}

		err = s.store.UpdateAssetPrice(ctx, db.UpdateAssetPriceParams{
			ID:        job.AssetID,
			LivePrice: livePrice,
		})
		if err != nil {
			log.Printf("❌ [Worker] Failed to save live price to DB: %v", err)
		}

		err = s.fetchAndStoreHistoricalPrices(ctx, job.AssetID, job.Symbol)
		if err != nil {
			log.Printf("❌ [Worker] Failed to sync history for %s: %v", job.Symbol, err)
		}

		log.Printf("✅ [Worker] Successfully updated pricing data for %s", job.Symbol)
	}
}

func (s *AssetService) fetchExternalLivePrice(symbol string) (scalar.Decimal, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1m&range=1d", symbol)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return scalar.Decimal{}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return scalar.Decimal{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return scalar.Decimal{}, fmt.Errorf("yahoo api returned status code %d", resp.StatusCode)
	}

	var yResp YahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&yResp); err != nil {
		return scalar.Decimal{}, fmt.Errorf("failed to decode yahoo response: %w", err)
	}

	if len(yResp.Chart.Result) == 0 {
		return scalar.Decimal{}, fmt.Errorf("no results returned for symbol %s", symbol)
	}

	meta := yResp.Chart.Result[0].Meta
	if meta.RegularMarketPrice == 0 {
		return scalar.Decimal{}, fmt.Errorf("received zero price for symbol %s", symbol)
	}

	priceDecimal := decimal.NewFromFloat(meta.RegularMarketPrice)
	log.Printf("📊 [Yahoo HTTP] Fetched price for %s: %s %s", symbol, priceDecimal.String(), meta.Currency)

	return scalar.NewDecimal(priceDecimal), nil
}

func (s *AssetService) fetchAndStoreHistoricalPrices(ctx context.Context, assetID uuid.UUID, symbol string) error {
	return nil
}

type AssetService struct {
	store    db.Store
	rawDB    *sql.DB
	query    *gountries.Query // ISO 3166 Query Client
	jobQueue chan FetchJob
}

func NewAssetService(rawDB *sql.DB, store db.Store) *AssetService {
	s := &AssetService{
		store:    store,
		rawDB:    rawDB,
		query:    gountries.New(),
		jobQueue: make(chan FetchJob, 100),
	}
	go s.startPriceFetchWorker()

	return s
}

// Custom validation rule to verify real ISO 3166-1 alpha-2 countries
func (s *AssetService) isRealISOCountry(value interface{}) error {
	var str string

	// Handle both direct strings and string pointers safely
	switch v := value.(type) {
	case string:
		str = v
	case *string:
		if v != nil {
			str = *v
		}
	default:
		return nil // Field is not a string type, let other rules handle it
	}

	if str == "" {
		return nil // Let 'validation.Required' handle blank checks
	}

	// Now we have the actual string value to check against the official ISO data
	_, err := s.query.FindCountryByAlpha(str)
	if err != nil {
		return fmt.Errorf("must be a valid ISO 3166-1 alpha-2 country code")
	}
	return nil
}

// ValidateInput uses type-safe code rules instead of magic string struct tags
func (s *AssetService) ValidateInput(input model.CreateAssetInput) error {
	err := validation.ValidateStruct(&input,
		validation.Field(&input.Symbol, validation.Required, validation.Length(1, 10)),
		validation.Field(&input.Name, validation.Required, validation.Length(2, 100)),
		validation.Field(&input.Currency, validation.Required, validation.Length(3, 3), is.Alpha),
		validation.Field(&input.Isin, validation.NilOrNotEmpty, validation.Length(12, 12)),
		validation.Field(&input.Wkn, validation.NilOrNotEmpty, validation.Length(6, 6)),
	)
	if err != nil {
		return fmt.Errorf("base asset validation failed: %w", err)
	}

	switch input.AssetClass {
	case model.AssetClassStock:
		return validation.ValidateStruct(&input,
			validation.Field(&input.CountryCode, validation.Required, validation.Length(2, 2), is.Alpha, validation.By(s.isRealISOCountry)),
		)
	case model.AssetClassEtf:
		if input.CountryCode != nil && *input.CountryCode != "" {
			return fmt.Errorf("ETFs cannot be assigned a single country code")
		}
	}

	return nil
}

// CreateAsset executes a polymorphic database transaction to insert an asset and its subclass extension
func (s *AssetService) CreateAsset(ctx context.Context, input model.CreateAssetInput) (db.Asset, error) {
	var createdAsset db.Asset

	if err := s.ValidateInput(input); err != nil {
		return createdAsset, err
	}

	tx, err := s.rawDB.BeginTx(ctx, nil)
	if err != nil {
		return createdAsset, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	qtx := s.store.WithTx(tx)

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

	if err := tx.Commit(); err != nil {
		return createdAsset, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.jobQueue <- FetchJob{
		AssetID:    createdAsset.ID,
		Symbol:     createdAsset.Symbol,
		AssetClass: dbAssetClass,
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
