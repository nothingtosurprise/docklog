package db

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrDBNotFound = errors.New("database file not found")

func DefaultPath() string {
	if path := strings.TrimSpace(os.Getenv("DB_PATH")); path != "" {
		return path
	}
	return "docklog.db"
}

func ServerPath(authDisabled bool) string {
	if path := strings.TrimSpace(os.Getenv("DB_PATH")); path != "" {
		return path
	}
	if authDisabled {
		return ":memory:"
	}
	return "docklog.db"
}

// OpenExisting opens the database without creating a new file on disk.
func OpenExisting(path string) error {
	if path == ":memory:" {
		return InitDB(path)
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrDBNotFound, path)
		}
		return err
	}
	return InitDB(path)
}
