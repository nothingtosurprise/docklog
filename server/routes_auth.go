package server

import (
	"net/http"
	"strings"
	"time"

	"docklog/db"
	"docklog/middleware"
	"docklog/models"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerAuthRoutes() {
	s.echo.POST("/api/token", func(c echo.Context) error {
		ip := c.RealIP()
		if middleware.LoginRateLimit.IsLimited(ip, 10, 15*time.Minute) {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"error": "Too many login attempts. Try again later."})
		}

		username := strings.TrimSpace(c.FormValue("username"))
		password := c.FormValue("password")

		var id int
		var storedUsername string
		var hashedPassword string
		var isAdmin, passwordChanged, canStart, canStop, canRestart, canDelete, canShell, isRestricted, isActive bool
		var allowedContainers string
		var passwordVersion int
		err := db.DB.QueryRow(`SELECT id, username, password, is_admin, password_changed, can_start, can_stop, can_restart, can_delete, can_shell, is_restricted_access, allowed_containers, is_active, COALESCE(password_version, 1) FROM users WHERE lower(username) = lower(?) LIMIT 1`, username).Scan(
			&id, &storedUsername, &hashedPassword, &isAdmin, &passwordChanged, &canStart, &canStop, &canRestart, &canDelete, &canShell, &isRestricted, &allowedContainers, &isActive, &passwordVersion,
		)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) != nil {
			middleware.LoginRateLimit.RecordFailure(ip)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		}

		middleware.LoginRateLimit.Clear(ip)

		if !isActive {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Account deactivated. Please contact administrator."})
		}

		claims := &models.UserClaims{
			ID:                 id,
			Username:           storedUsername,
			IsAdmin:            isAdmin,
			IsRestrictedAccess: isRestricted,
			CanStart:           canStart,
			CanStop:            canStop,
			CanRestart:         canRestart,
			CanDelete:          canDelete,
			CanShell:           canShell,
			AllowedContainers:  allowedContainers,
			IsActive:           isActive,
			PasswordVersion:    passwordVersion,
		}

		accessToken, refreshToken, err := middleware.IssueTokenPair(claims)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"access_token":     accessToken,
			"refresh_token":    refreshToken,
			"is_admin":         isAdmin,
			"password_changed": passwordChanged,
		})
	})

	s.echo.POST("/api/token/refresh", func(c echo.Context) error {
		refreshToken := strings.TrimSpace(c.FormValue("refresh_token"))
		if refreshToken == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing refresh token"})
		}

		claims, err := middleware.ValidateRefreshToken(refreshToken)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid refresh token"})
		}

		var passwordChanged bool
		if err := db.DB.QueryRow(
			"SELECT password_changed FROM users WHERE id = ?",
			claims.ID,
		).Scan(&passwordChanged); err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User verification failed"})
		}

		accessToken, newRefreshToken, err := middleware.IssueTokenPair(claims)
		if err != nil {
			return err
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"access_token":     accessToken,
			"refresh_token":    newRefreshToken,
			"is_admin":         claims.IsAdmin,
			"password_changed": passwordChanged,
		})
	})
}
