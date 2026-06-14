package db

import (
	"database/sql"
	"errors"
	"strings"
)

// TrimUsername normalizes user input before storage or lookup.
func TrimUsername(username string) string {
	return strings.TrimSpace(username)
}

// UsernameTaken reports whether another account already uses this username (case-insensitive).
func UsernameTaken(username string) (bool, error) {
	username = TrimUsername(username)
	if username == "" {
		return false, nil
	}

	var id int
	err := DB.QueryRow(
		`SELECT id FROM users WHERE lower(username) = lower(?) LIMIT 1`,
		username,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
