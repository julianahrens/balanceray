# BalanceRay – Modern Portfolio Tracker & Performance Engine

BalanceRay is a privacy-focused, self-hosted portfolio tracking application and financial performance engine. It enables comprehensive tracking and analysis of classic securities, physical precious metals, and cryptocurrencies within a single, highly performant ecosystem.

The core philosophy of BalanceRay revolves around **true physical and geographical transparency (look-through)** and the mathematically precise mapping of diverse asset classes.

---

## Core Financial Features

### 1. Holistic Asset Class Architecture
To accurately reflect their real-world characteristics, BalanceRay strictly differentiates between specialized asset types rather than treating them as generic entities:
* **Securities (Stocks & ETFs):** Tracking via ISIN/WKN including automatic issuer mapping.
* **Physical Precious Metals:** Fine-grained support for form factors (bars, coins, coin-bars, granules). Includes manufacturer tracking (LBMA status for resale premium verification) and face value protection (guaranteed legal tender minimums for bullion coins like Maple Leafs or Kangaroos).
* **Cryptocurrencies:** Native Layer-1 coins (BTC, ETH, SOL) vs. smart-contract tokens (ERC-20, SPL). Tied directly to the respective blockchain network with full support for on-chain metadata (contract addresses).

### 2. ETF Look-Through & Concentration Risk Analytics (Deep Analytics)
Instead of treating ETFs as a "black box", BalanceRay disassembles pooled products into their underlying holdings:
* **Equity Look-Through:** Automatically calculates aggregate company exposure across all your ETFs. You see your *real* total weight in individual stocks (e.g., your effective Apple exposure across a combined MSCI World and S&P 500 portfolio).
* **Geographical Look-Through:** Aggregates underlying country allocations (ISO-code based) for precise macroeconomic portfolio exposure mapping.

### 3. Advanced Crypto Fiscal Logic
Handling web3-specific transactions requires specialized accounting parameters. BalanceRay integrates native booking systems for:
* **Staking Rewards & Airdrops:** Tax-compliant and mathematically sound handling of inbound assets with zero acquisition cost (cost basis = 0).
* **Multi-Asset Gas Fees:** Tracking of network fees paid in a different currency than the transaction asset itself (e.g., buying USDC but paying the gas fee in ETH) to keep wallet balances perfectly accurate.

### 4. Double-Entry Cashflow Tracking
Every brokerage account and crypto wallet is tied to a clearing account. Internal transfers, cash positions, high-yield savings accounts, as well as dividend and interest flows are recorded as a closed, double-entry system.

---

## Technical Architecture & Domain Model

The codebase follows a strict domain separation pattern (**Class-Table-Inheritance**):
1. **Core Domain (Base Asset):** Manages global financial metadata such as ticker symbols, base currencies, and live price feeds.
2. **Specialized Sub-Domains:** Extend the base asset with the highly specific attributes required by each asset class (Physical Metal parameters, Crypto-Chain definitions, Security-Scraper properties).

---

## Repository Structure

This project is organized as a monorepo:
* `/backend`: The core performance engine and GraphQL API written in Go (highly optimized for batch processing via PostgreSQL).
* `/frontend`: The interactive dashboard UI built with SvelteKit and Houdini GraphQL.

*Technical documentation regarding local setup, database migrations (`sqlc`), and API generation (`gqlgen`/`Houdini`) can be found in the respective README files inside the subdirectories.*
