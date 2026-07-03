package services_test

import (
	"errors"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/julianahrens/balanceray/backend/internal/graph/model"
	"github.com/julianahrens/balanceray/backend/internal/services"
)

func ptr(s string) *string { return &s }

func TestAssetService_ValidateInput_Detailed(t *testing.T) {
	service := services.NewAssetService(nil, nil)

	tests := []struct {
		name          string
		input         model.CreateAssetInput
		wantErr       bool
		expectedField string
	}{
		{
			name: "Valid Stock Asset",
			input: model.CreateAssetInput{
				Symbol:      "AAPL",
				Name:        "Apple Inc.",
				Currency:    "USD",
				AssetClass:  model.AssetClassStock,
				Isin:        ptr("US0378331005"),
				Wkn:         ptr("865985"),
				CountryCode: ptr("US"),
			},
			wantErr: false,
		},
		{
			name: "Invalid Currency Length",
			input: model.CreateAssetInput{
				Symbol:     "AAPL",
				Name:       "Apple Inc.",
				Currency:   "US-DOLLAR", // too long
				AssetClass: model.AssetClassStock,
			},
			wantErr:       true,
			expectedField: "currency",
		},
		{
			name: "Invalid ISIN Length",
			input: model.CreateAssetInput{
				Symbol:     "AAPL",
				Name:       "Apple Inc.",
				Currency:   "USD",
				AssetClass: model.AssetClassStock,
				Isin:       ptr("US123"), // too short
			},
			wantErr:       true,
			expectedField: "isin",
		},
		{
			name: "Stock Missing Country Code",
			input: model.CreateAssetInput{
				Symbol:      "AAPL",
				Name:        "Apple Inc.",
				Currency:    "USD",
				AssetClass:  model.AssetClassStock,
				CountryCode: nil, // required at stock
			},
			wantErr:       true,
			expectedField: "countryCode",
		},
		{
			name: "Invalid ISO Country Code but Correct Length",
			input: model.CreateAssetInput{
				Symbol:      "BMW.DE",
				Name:        "BMW AG",
				Currency:    "EUR",
				AssetClass:  model.AssetClassStock,
				CountryCode: ptr("XX"), // 2 Letter, but ISO 3166-1 alpha-2 invalid
			},
			wantErr:       true,
			expectedField: "countryCode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateInput(tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateInput() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectedField != "" {
				var valErrors validation.Errors
				if !errors.As(err, &valErrors) {
					t.Fatalf("Expected ozzo-validation.Errors, but got: %v", err)
				}

				if _, fieldExists := valErrors[tt.expectedField]; !fieldExists {
					t.Errorf("Expected validation error on field %q, but got errors for: %v", tt.expectedField, valErrors)
				} else {
					t.Logf("✅ Successfully caught expected error on field %q: %v", tt.expectedField, valErrors[tt.expectedField])
				}
			}
		})
	}
}
