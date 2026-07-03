-- name: CreateBaseAsset :one
INSERT INTO assets (
    symbol, name, currency, asset_class, live_price
) VALUES (
             $1, $2, $3, $4, $5
         )
    RETURNING id, symbol, name, currency, asset_class, live_price, updated_at;

-- name: CreateStockExtension :exec
INSERT INTO assets_stock (
    asset_id, isin, wkn, issuer, country_code
) VALUES (
             $1, $2, $3, $4, $5
         );

-- name: CreateEtfExtension :exec
INSERT INTO assets_etf (
    asset_id, isin, wkn, issuer, provider_product_id
) VALUES (
             $1, $2, $3, $4, $5
         );

-- name: ListAllAssets :many
-- Fetches the core fields of all assets. The extensions (Stock/ETF details)
-- should be loaded lazily or via batching to keep this initial query fast.
SELECT id, symbol, name, currency, asset_class, live_price, updated_at
FROM assets
ORDER BY name ASC;

-- name: GetStockExtensionByAssetID :one
SELECT asset_id, isin, wkn, issuer, country_code
FROM assets_stock
WHERE asset_id = $1;

-- name: GetEtfExtensionByAssetID :one
SELECT asset_id, isin, wkn, issuer, provider_product_id
FROM assets_etf
WHERE asset_id = $1;

-- name: GetHistoricalPrices :many
SELECT asset_id, price_date, open_price, high_price, low_price, close_price
FROM historical_prices
WHERE asset_id = $1 AND price_date >= $2
ORDER BY price_date DESC;

-- name: ListAllAssetsWithExtensions :many
SELECT
    a.id, a.symbol, a.name, a.currency, a.asset_class, a.live_price, a.updated_at,
    s.isin AS stock_isin, s.wkn AS stock_wkn, s.issuer AS stock_issuer, s.country_code,
    e.isin AS etf_isin, e.wkn AS etf_wkn, e.issuer AS etf_issuer, e.provider_product_id
FROM assets a
         LEFT JOIN assets_stock s ON a.id = s.asset_id AND a.asset_class = 'STOCK'
         LEFT JOIN assets_etf e ON a.id = e.asset_id AND a.asset_class = 'ETF'
ORDER BY a.name ASC;

-- name: UpdateAssetPrice :exec
UPDATE assets
SET live_price = $2, updated_at = NOW()
WHERE id = $1;
