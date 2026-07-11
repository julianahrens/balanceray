# Market Data Sidecar (market-data-yf)

This microservice acts as an internal gRPC provider for retrieving stock market quotes, historical OHLC candles, metadata, and dividend histories.

## Legal & Data Source Notice
- **Data Source:** This service utilizes the open-source Python library [`yfinance`](https://github.com/ranaroussi/yfinance).
- **Trademarks:** "Yahoo!" and "Yahoo Finance" are registered trademarks of Yahoo Inc. This project is not affiliated, endorsed, or sponsored by Yahoo Inc.
- **Usage:** Strictly intended for personal/educational portfolio analysis.

## Development Setup

```bash
# Generate Protobuf stubs
uv run python -m grpc_tools.protoc -I../../proto --python_out=. --grpc_python_out=. ../../proto/market_data.proto

# Run gRPC server
uv run python server.py
```
