package db

import (
	"testing"
)

func TestUsernameTakenCaseInsensitive(t *testing.T) {
	if err := InitDB(":memory:"); err != nil {
		t.Fatalf("init db: %v", err)
	}

	_, err := DB.Exec(
		`INSERT INTO users (username, password, is_admin, password_changed, is_active) VALUES (?, ?, 0, 1, 1)`,
		"User", "hash",
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	taken, err := UsernameTaken("user")
	if err != nil {
		t.Fatalf("UsernameTaken: %v", err)
	}
	if !taken {
		t.Fatal("expected username user to match stored User")
	}

	taken, err = UsernameTaken("USER")
	if err != nil {
		t.Fatalf("UsernameTaken: %v", err)
	}
	if !taken {
		t.Fatal("expected username USER to match stored User")
	}

	taken, err = UsernameTaken("other")
	if err != nil {
		t.Fatalf("UsernameTaken: %v", err)
	}
	if taken {
		t.Fatal("expected other to be available")
	}
}

func TestTrimUsername(t *testing.T) {
	if got := TrimUsername("  admin  "); got != "admin" {
		t.Fatalf("got %q", got)
	}
}
