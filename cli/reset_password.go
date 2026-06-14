package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"docklog/db"

	"golang.org/x/crypto/bcrypt"
)

const minPasswordLength = 8

// RunResetPassword resets a user password in the SQLite database.
func RunResetPassword(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: docklog reset-password <username> <new-password>")
	}

	username := strings.TrimSpace(args[0])
	password := args[1]
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}

	dbPath := db.DefaultPath()
	if dbPath == ":memory:" {
		return fmt.Errorf("cannot reset password on an in-memory database")
	}

	if err := db.OpenExisting(dbPath); err != nil {
		if errors.Is(err, db.ErrDBNotFound) {
			return fmt.Errorf("database not found at %s (set DB_PATH or run from the project directory)", dbPath)
		}
		return fmt.Errorf("open database: %w", err)
	}

	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	var id int
	var storedUsername string
	err = db.DB.QueryRow(
		`SELECT id, username FROM users WHERE lower(username) = lower(?) LIMIT 1`,
		username,
	).Scan(&id, &storedUsername)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("user %q not found", username)
		}
		return fmt.Errorf("lookup user: %w", err)
	}

	res, err := db.DB.Exec(
		`UPDATE users SET password = ?, password_changed = 0, password_version = COALESCE(password_version, 1) + 1 WHERE id = ?`,
		string(h), id,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user %q not found", username)
	}

	fmt.Printf("Password reset for %q. Existing sessions are invalidated.\n", storedUsername)
	return nil
}
