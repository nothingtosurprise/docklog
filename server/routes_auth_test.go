package server

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"docklog/db"

	"golang.org/x/crypto/bcrypt"
)

func TestLoginUsernameCaseInsensitive(t *testing.T) {
	if err := db.InitDB(":memory:"); err != nil {
		t.Fatalf("init db: %v", err)
	}

	h, err := bcrypt.GenerateFromPassword([]byte("user1234"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if _, err := db.DB.Exec(
		`INSERT INTO users (username, password, is_admin, password_changed, is_active) VALUES (?, ?, 0, 1, 1)`,
		"User", string(h),
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	srv := New(Deps{})
	srv.registerAuthRoutes()

	form := url.Values{}
	form.Set("username", "user")
	form.Set("password", "user1234")

	req := httptest.NewRequest(http.MethodPost, "/api/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	srv.echo.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
}
