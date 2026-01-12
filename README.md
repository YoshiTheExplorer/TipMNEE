# TipMNEE Backend (Frontend Dev Setup)

This backend powers the TipMNEE frontend and browser extension.
It handles wallet login, YouTube channel verification, payout resolution, and claim signing.

Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Ethereum mainnet RPC (Infura / Alchemy / QuickNode)
- Deployed TipEscrow contract on mainnet

1. Database
Create a local database:
make postgres
make createdb

Run migrations:
make migrateup

2. Environment Variables
Create a .env file in the repo root:
DB_SOURCE=postgresql://root:secret@localhost:5434/tipmnee?sslmode=disable
POSTGRES_USER=root
POSTGRES_PASSWORD=secret
POSTGRES_DB=tipmnee
POSTGRES_PORT=5434

JWT_SECRET=LongString

PORT=8080

CHAIN_ID=1

ESCROW_CONTRACT=0x677F8622BCE181Ea7c85aF75742DF592192b4500

TOKEN_CONTRACT=0x8ccedbAe4916b79da7F3F612EfB2EB93A2bFD6cF

ETH_RPC_URL=https://eth-mainnet.g.alchemy.com/v2/YOUR_KEY

SEPOLIA_RPC_URL=https://eth-sepolia.g.alchemy.com/v2/YOUR_KEY

VERIFIER_PRIVATE_KEY=YOUR_PRIVATE_KEY

Replace all instance of SEPOLIA_RPC_URL with ETH_RPC_URL

3. Run the Server
go mod download
make server

Verify:
curl http://localhost:8080/health
Expected:
{ "ok": true }

4. Frontend Integration
Point the frontend or extension API base URL to:
http://localhost:8080
