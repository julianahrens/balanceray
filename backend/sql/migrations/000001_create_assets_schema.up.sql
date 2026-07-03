-- 1. Create Enums
CREATE TYPE asset_class AS ENUM ('STOCK', 'ETF');

-- 2. Core Base Table (Shared by all assets)
CREATE TABLE assets (
                        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                        symbol VARCHAR(20) UNIQUE NOT NULL,
                        name VARCHAR(255) NOT NULL,
                        currency CHAR(3) NOT NULL,
                        asset_class asset_class NOT NULL,
                        live_price NUMERIC(16, 4) NOT NULL DEFAULT 0.0000,
                        updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- 3. Specific Sub-Table: Stocks
CREATE TABLE assets_stock (
                              asset_id UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
                              isin VARCHAR(12) UNIQUE,
                              wkn VARCHAR(6) UNIQUE,
                              issuer VARCHAR(100),
                              country_code CHAR(2) NOT NULL
);

-- 4. Specific Sub-Table: ETFs
CREATE TABLE assets_etf (
                            asset_id UUID PRIMARY KEY REFERENCES assets(id) ON DELETE CASCADE,
                            isin VARCHAR(12) UNIQUE,
                            wkn VARCHAR(6) UNIQUE,
                            issuer VARCHAR(100),
                            provider_product_id VARCHAR(100)
);

-- 5. ETF Look-Through: Holdings
CREATE TABLE etf_asset_holdings (
                                    etf_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
                                    holding_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
                                    percentage NUMERIC(7, 6) NOT NULL,
                                    PRIMARY KEY (etf_asset_id, holding_asset_id)
);

-- 6. ETF Look-Through: Geographical Allocation
CREATE TABLE etf_country_allocations (
                                         etf_asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
                                         country_code CHAR(2) NOT NULL,
                                         percentage NUMERIC(7, 6) NOT NULL,
                                         PRIMARY KEY (etf_asset_id, country_code)
);

-- 7. Time-Series: Historical Price Points
CREATE TABLE historical_prices (
                                   asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
                                   price_date DATE NOT NULL,
                                   open_price NUMERIC(16, 4) NOT NULL,
                                   high_price NUMERIC(16, 4) NOT NULL,
                                   low_price NUMERIC(16, 4) NOT NULL,
                                   close_price NUMERIC(16, 4) NOT NULL,
                                   PRIMARY KEY (asset_id, price_date)
);

-- 8. Performance Indexes
CREATE INDEX idx_historical_prices_date ON historical_prices(asset_id, price_date DESC);
CREATE INDEX idx_etf_holdings_percentage ON etf_asset_holdings(etf_asset_id, percentage DESC);
