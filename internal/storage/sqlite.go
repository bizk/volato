package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const migrationSQL = `
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
`

// PriceEntry represents a flight price record
type PriceEntry struct {
	ID            int64
	Origin        string
	Destination   string
	DepartureDate string
	ReturnDate    string
	Price         float64
	Currency      string
	APISource     string
	CheckedAt     time.Time
}

// Store handles all database operations
type Store struct {
	db *sql.DB
}

// New opens a database connection and runs migrations
func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &Store{db: db}

	// Run migrations
	if err := store.runMigrations(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

// runMigrations executes the migration SQL
func (s *Store) runMigrations() error {
	_, err := s.db.Exec(migrationSQL)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SavePrice saves a price entry to the database
func (s *Store) SavePrice(e PriceEntry) error {
	query := `
		INSERT INTO price_history (origin, destination, departure_date, return_date, price, currency, api_source)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.Exec(
		query,
		e.Origin,
		e.Destination,
		e.DepartureDate,
		e.ReturnDate,
		e.Price,
		e.Currency,
		e.APISource,
	)
	if err != nil {
		return fmt.Errorf("failed to insert price: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	e.ID = id
	return nil
}

// GetAveragePrice returns the average price for a route within the specified days
func (s *Store) GetAveragePrice(origin, destination, departureDate, returnDate string, days int) (float64, error) {
	query := `
		SELECT AVG(price) FROM price_history
		WHERE origin = ? AND destination = ? AND departure_date = ? AND return_date = ?
		AND checked_at >= datetime('now', ? || ' days')
	`

	var avg sql.NullFloat64
	err := s.db.QueryRow(query, origin, destination, departureDate, returnDate, -days).Scan(&avg)
	if err != nil {
		return 0, fmt.Errorf("failed to get average price: %w", err)
	}

	if !avg.Valid {
		return 0, nil
	}

	return avg.Float64, nil
}

// IsDealSent checks if a deal has already been sent
func (s *Store) IsDealSent(hash string) (bool, error) {
	query := `SELECT COUNT(*) FROM sent_deals WHERE flight_hash = ?`

	var count int
	err := s.db.QueryRow(query, hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if deal sent: %w", err)
	}

	return count > 0, nil
}

// MarkDealSent marks a deal as sent
func (s *Store) MarkDealSent(hash string) error {
	query := `INSERT INTO sent_deals (flight_hash) VALUES (?)`

	_, err := s.db.Exec(query, hash)
	if err != nil {
		return fmt.Errorf("failed to mark deal as sent: %w", err)
	}

	return nil
}

// GetRecentPrices returns the most recent price entries
func (s *Store) GetRecentPrices(limit int) ([]PriceEntry, error) {
	query := `
		SELECT id, origin, destination, departure_date, return_date, price, currency, api_source, checked_at
		FROM price_history
		ORDER BY checked_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent prices: %w", err)
	}
	defer rows.Close()

	var entries []PriceEntry
	for rows.Next() {
		var e PriceEntry
		err := rows.Scan(
			&e.ID,
			&e.Origin,
			&e.Destination,
			&e.DepartureDate,
			&e.ReturnDate,
			&e.Price,
			&e.Currency,
			&e.APISource,
			&e.CheckedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan price entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating price entries: %w", err)
	}

	return entries, nil
}

// Cleanup removes old records based on age
func (s *Store) Cleanup(priceHistoryDays, sentDealsDays int) error {
	// Delete old price history
	priceQuery := `
		DELETE FROM price_history
		WHERE checked_at < datetime('now', ? || ' days')
	`
	_, err := s.db.Exec(priceQuery, -priceHistoryDays)
	if err != nil {
		return fmt.Errorf("failed to clean price history: %w", err)
	}

	// Delete old sent deals
	dealsQuery := `
		DELETE FROM sent_deals
		WHERE sent_at < datetime('now', ? || ' days')
	`
	_, err = s.db.Exec(dealsQuery, -sentDealsDays)
	if err != nil {
		return fmt.Errorf("failed to clean sent deals: %w", err)
	}

	return nil
}

// SetMetadata sets a metadata key-value pair
func (s *Store) SetMetadata(key, value string) error {
	query := `INSERT OR REPLACE INTO metadata (key, value) VALUES (?, ?)`

	_, err := s.db.Exec(query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return nil
}

// GetMetadata retrieves a metadata value by key
func (s *Store) GetMetadata(key string) (string, error) {
	query := `SELECT value FROM metadata WHERE key = ?`

	var value string
	err := s.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("metadata key not found: %s", key)
		}
		return "", fmt.Errorf("failed to get metadata: %w", err)
	}

	return value, nil
}
