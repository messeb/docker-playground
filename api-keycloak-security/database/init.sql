-- ── Schema ────────────────────────────────────────────────────────────────────
-- The bank account number is set as a custom attribute in Keycloak and
-- included in every JWT as the 'bank_account_number' claim. No UUID matching
-- between Keycloak and this database is required.

CREATE TABLE IF NOT EXISTS bank_accounts (
    id             SERIAL PRIMARY KEY,
    account_number VARCHAR(50)    NOT NULL UNIQUE,
    owner_name     VARCHAR(100)   NOT NULL,
    balance        NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transactions (
    id          SERIAL PRIMARY KEY,
    account_id  INTEGER        NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
    type        VARCHAR(10)    NOT NULL CHECK (type IN ('deposit', 'withdrawal')),
    amount      NUMERIC(15, 2) NOT NULL CHECK (amount > 0),
    description TEXT,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);

-- ── Seed Data ─────────────────────────────────────────────────────────────────
-- account_number values must match the 'bank_account_number' user attribute
-- set for each user in keycloak/realm-export.json.

INSERT INTO bank_accounts (account_number, owner_name, balance)
VALUES
    ('BANK-0001-2024', 'Alice Demo', 2500.00),
    ('BANK-0002-2024', 'Bob Admin',  5000.00)
ON CONFLICT DO NOTHING;

INSERT INTO transactions (account_id, type, amount, description)
SELECT id, 'deposit', 2500.00, 'Initial deposit'
FROM bank_accounts WHERE account_number = 'BANK-0001-2024'
ON CONFLICT DO NOTHING;

INSERT INTO transactions (account_id, type, amount, description)
SELECT id, 'deposit', 5000.00, 'Initial deposit'
FROM bank_accounts WHERE account_number = 'BANK-0002-2024'
ON CONFLICT DO NOTHING;
