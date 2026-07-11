import os
import sys

# Set cache directory to writable location
os.environ["YFINANCE_CACHE_DIR"] = "/tmp/py-yfinance"

from concurrent import futures
import signal
import time
import re
import logging
import grpc
from curl_cffi import requests as curl_requests
import yfinance as yf
from yfinance import set_tz_cache_location
from grpc_reflection.v1alpha import reflection

import market_data_pb2
import market_data_pb2_grpc

try:
    set_tz_cache_location("/tmp/py-yfinance")
except Exception:
    pass

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[logging.StreamHandler(sys.stdout)]
)


def get_authenticated_session():
    """Create a curl_cffi session configured with browser impersonation to bypass Yahoo connection limits."""
    session = curl_requests.Session(impersonate="chrome120")
    session.headers.update({
        "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
        "Accept-Language": "en-US,en;q=0.9",
        "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
    })
    return session


def clean_opt_str(val):
    if val is None:
        return None
    val_str = str(val).strip()
    if val_str in ("", "N/A", "None", "null", "nan"):
        return None
    return val_str


def clean_opt_float(val):
    if val is None:
        return None
    try:
        return float(val)
    except (ValueError, TypeError):
        return None


def extract_wkn_from_isin(isin):
    if isin and len(isin) == 12 and isin.startswith("DE"):
        return isin[5:11]
    return None


class MarketDataServiceServicer(market_data_pb2_grpc.MarketDataServiceServicer):

    def GetHistoricalOHLC(self, request, context):
        logging.info(f"GetHistoricalOHLC for symbol='{request.symbol}', start='{request.start_date}'")
        try:
            session = get_authenticated_session()
            ticker = yf.Ticker(request.symbol, session=session)
            df = ticker.history(
                start=request.start_date,
                interval=request.interval if request.interval else "1d",
                auto_adjust=True
            )

            if df.empty:
                return market_data_pb2.HistoryResponse(symbol=request.symbol, candles=[])

            candles = []
            for timestamp, row in df.iterrows():
                candles.append(market_data_pb2.OHLCPoint(
                    timestamp=int(timestamp.timestamp()),
                    open=float(row['Open']),
                    high=float(row['High']),
                    low=float(row['Low']),
                    close=float(row['Close']),
                    volume=int(row['Volume'])
                ))

            currency = ticker.fast_info.currency or "EUR"
            return market_data_pb2.HistoryResponse(
                symbol=request.symbol,
                currency=currency,
                candles=candles
            )

        except Exception as e:
            logging.error(f"Error in GetHistoricalOHLC: {e}")
            context.abort(grpc.StatusCode.INTERNAL, f"Failed to fetch OHLC data: {str(e)}")

    def GetLiveQuote(self, request, context):
        logging.info(f"GetLiveQuote for symbol='{request.symbol}'")
        try:
            session = get_authenticated_session()
            ticker = yf.Ticker(request.symbol, session=session)
            fast = ticker.fast_info

            price = fast.last_price
            if price is None or price == 0:
                context.abort(grpc.StatusCode.NOT_FOUND, f"Symbol {request.symbol} not found")

            currency = fast.currency or "EUR"
            now_unix = int(time.time())

            return market_data_pb2.QuoteResponse(
                symbol=request.symbol,
                price=float(price),
                currency=currency,
                timestamp=now_unix
            )
        except Exception as e:
            logging.error(f"Error in GetLiveQuote: {e}")
            context.abort(grpc.StatusCode.INTERNAL, f"Failed to fetch live quote: {str(e)}")

    def GetAssetMetadata(self, request, context):
        raw_identifier = request.identifier.strip()
        logging.info(f"GetAssetMetadata for identifier='{raw_identifier}'")

        try:
            session = get_authenticated_session()
            ticker_symbol = raw_identifier
            resolved_isin = None
            resolved_wkn = None

            is_isin = bool(re.match(r'^[A-Z]{2}[A-Z0-9]{9}\d$', raw_identifier))
            is_wkn = bool(re.match(r'^[A-Z0-9]{6}$', raw_identifier)) and not is_isin

            if is_isin:
                resolved_isin = raw_identifier
                resolved_wkn = extract_wkn_from_isin(resolved_isin)
                search_res = yf.Search(raw_identifier, max_results=1, session=session)
                if search_res.quotes:
                    ticker_symbol = search_res.quotes[0]['symbol']
            elif is_wkn:
                resolved_wkn = raw_identifier
                search_res = yf.Search(raw_identifier, max_results=1, session=session)
                if search_res.quotes:
                    ticker_symbol = search_res.quotes[0]['symbol']

            ticker = yf.Ticker(ticker_symbol, session=session)
            info = ticker.info or {}

            if not info or info.get("quoteType") is None:
                context.abort(grpc.StatusCode.NOT_FOUND, f"No metadata found for identifier: {raw_identifier}")

            if not resolved_isin:
                resolved_isin = clean_opt_str(info.get("isin"))
                if not resolved_isin:
                    search_res = yf.Search(ticker_symbol, max_results=1, session=session)
                    if search_res.quotes and 'isin' in search_res.quotes[0]:
                        resolved_isin = search_res.quotes[0]['isin']

            if not resolved_wkn and resolved_isin:
                resolved_wkn = extract_wkn_from_isin(resolved_isin)

            return market_data_pb2.AssetMetadataResponse(
                symbol=ticker_symbol,
                isin=clean_opt_str(resolved_isin),
                wkn=clean_opt_str(resolved_wkn),
                name=info.get("longName") or info.get("shortName") or ticker_symbol,
                currency=info.get("currency") or ticker.fast_info.currency or "EUR",
                quote_type=info.get("quoteType", "UNKNOWN"),

                sector=clean_opt_str(info.get("sector")),
                industry=clean_opt_str(info.get("industry")),
                country=clean_opt_str(info.get("country")),
                summary=clean_opt_str(info.get("longBusinessSummary")),

                market_cap=clean_opt_float(info.get("marketCap")),
                dividend_yield=clean_opt_float(info.get("dividendYield")),
                trailing_pe=clean_opt_float(info.get("trailingPE")),
                fifty_two_week_high=clean_opt_float(info.get("fiftyTwoWeekHigh")),
                fifty_two_week_low=clean_opt_float(info.get("fiftyTwoWeekLow"))
            )

        except Exception as e:
            logging.error(f"Error in GetAssetMetadata: {e}")
            context.abort(grpc.StatusCode.INTERNAL, f"Failed to fetch metadata: {str(e)}")

    def GetDividends(self, request, context):
        logging.info(f"GetDividends for symbol='{request.symbol}'")
        try:
            session = get_authenticated_session()
            ticker = yf.Ticker(request.symbol, session=session)
            divs = ticker.dividends

            if divs.empty:
                return market_data_pb2.DividendsResponse(symbol=request.symbol, events=[])

            events = []
            for timestamp, amount in divs.items():
                ts_sec = int(timestamp.timestamp())

                if request.HasField("start_date") and request.start_date:
                    start_ts = int(time.mktime(time.strptime(request.start_date, "%Y-%m-%d")))
                    if ts_sec < start_ts:
                        continue

                events.append(market_data_pb2.DividendEvent(
                    ex_date_timestamp=ts_sec,
                    amount_per_share=float(amount)
                ))

            return market_data_pb2.DividendsResponse(symbol=request.symbol, events=events)

        except Exception as e:
            logging.error(f"Error in GetDividends: {e}")
            context.abort(grpc.StatusCode.INTERNAL, f"Failed to fetch dividends: {str(e)}")


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    market_data_pb2_grpc.add_MarketDataServiceServicer_to_server(
        MarketDataServiceServicer(), server
    )

    SERVICE_NAMES = (
        market_data_pb2.DESCRIPTOR.services_by_name['MarketDataService'].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    server.add_insecure_port('[::]:50051')
    logging.info("Starting Market Data YF Sidecar on port 50051...")
    server.start()

    def handle_sigterm(*_):
        logging.info("Received shutdown signal. Stopping gRPC server gracefully...")
        server.stop(grace=5)
        sys.exit(0)

    signal.signal(signal.SIGINT, handle_sigterm)
    signal.signal(signal.SIGTERM, handle_sigterm)

    server.wait_for_termination()


if __name__ == '__main__':
    serve()
