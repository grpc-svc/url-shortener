package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"url-shortener/internal/storage"

	"github.com/mattn/go-sqlite3"
)

type Storage struct {
	db *sql.DB
}

// New initializes a new SQLite storage with the given file path.
func New(storagePath string) (*Storage, error) {
	const op = "storage.sqlite.New"

	db, err := sql.Open("sqlite3", storagePath)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// SaveURL saves the original URL with the given alias.
func (s *Storage) SaveURL(ctx context.Context, alias, originalURL, ownerEmail string) error {
	const op = "storage.sqlite.SaveURL"

	stmt, err := s.db.PrepareContext(ctx, "INSERT INTO urls(alias, url, owner_email) VALUES(?, ?, ?)")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = stmt.Close() }()

	_, err = stmt.ExecContext(ctx, alias, originalURL, ownerEmail)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return fmt.Errorf("%s: %w", op, storage.ErrURLExists)
		}
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

// GetURL retrieves the original URL for the given alias.
func (s *Storage) GetURL(ctx context.Context, alias string) (string, error) {
	const op = "storage.sqlite.GetURL"

	stmt, err := s.db.PrepareContext(ctx, "SELECT url FROM urls WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = stmt.Close() }()

	row := stmt.QueryRowContext(ctx, alias)

	var originalURL string
	err = row.Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return originalURL, nil
}

// GetURLOwner retrieves the owner email for the given alias.
func (s *Storage) GetURLOwner(ctx context.Context, alias string) (string, error) {
	const op = "storage.sqlite.GetURLOwner"

	stmt, err := s.db.PrepareContext(ctx, "SELECT owner_email FROM urls WHERE alias = ?")
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = stmt.Close() }()

	row := stmt.QueryRowContext(ctx, alias)

	var ownerEmail string
	err = row.Scan(&ownerEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
		}
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return ownerEmail, nil
}

// DeleteURL removes the URL with the given alias from storage.
func (s *Storage) DeleteURL(ctx context.Context, alias string) error {
	const op = "storage.sqlite.DeleteURL"

	stmt, err := s.db.PrepareContext(ctx, "DELETE FROM urls WHERE alias = ?")
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}
	defer func() { _ = stmt.Close() }()

	result, err := stmt.ExecContext(ctx, alias)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: %w", op, storage.ErrURLNotFound)
	}

	return nil
}
