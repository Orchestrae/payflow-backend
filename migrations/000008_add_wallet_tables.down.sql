-- Migration: Rollback business_wallets and wallet_transactions tables
-- Created: 2026-01-14

DROP TABLE IF EXISTS wallet_transactions;
DROP TABLE IF EXISTS business_wallets;
