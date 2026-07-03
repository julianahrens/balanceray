-- 1. Drop Indexes (Optional, as dropping tables removes associated indexes, but good practice)
DROP INDEX IF EXISTS idx_etf_holdings_percentage;
DROP INDEX IF EXISTS idx_historical_prices_date;

-- 2. Drop Tables (Child tables first to avoid foreign key violations)
DROP TABLE IF EXISTS historical_prices;
DROP TABLE IF EXISTS etf_country_allocations;
DROP TABLE IF EXISTS etf_asset_holdings;
DROP TABLE IF EXISTS assets_etf;
DROP TABLE IF EXISTS assets_stock;
DROP TABLE IF EXISTS assets;

-- 3. Drop Custom Enums
DROP TYPE IF EXISTS asset_class;
