CREATE TABLE IF NOT EXISTS price_history (
    id INTEGER PRIMARY KEY,
    origin TEXT NOT NULL,
    destination TEXT NOT NULL,
    departure_date TEXT NOT NULL,
    return_date TEXT NOT NULL,
    price REAL NOT NULL,
    currency TEXT NOT NULL,
    api_source TEXT NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sent_deals (
    id INTEGER PRIMARY KEY,
    flight_hash TEXT UNIQUE NOT NULL,
    sent_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_price_history_route
ON price_history(origin, destination, departure_date, return_date);

CREATE INDEX IF NOT EXISTS idx_price_history_checked_at ON price_history(checked_at);
CREATE INDEX IF NOT EXISTS idx_sent_deals_sent_at ON sent_deals(sent_at);
