package server

import (
	"fmt"
	"log"
	"net/http"

	"docklog/db"
	"docklog/middleware"
	"docklog/models"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (s *Server) registerUserRoutes(r *echo.Group) {
	r.POST("/user/change-password", func(c echo.Context) error {
		token := c.Get("user").(*jwt.Token)
		user := token.Claims.(*models.UserClaims)
		newPassword := c.FormValue("password")

		if len(newPassword) < middleware.MinPasswordLength {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Password must be at least %d characters", middleware.MinPasswordLength)})
		}

		var hashedPassword string
		var passwordChanged bool
		err := db.DB.QueryRow("SELECT password, password_changed FROM users WHERE id = ?", user.ID).Scan(&hashedPassword, &passwordChanged)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
		}

		if passwordChanged {
			currentPassword := c.FormValue("current_password")
			if currentPassword == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Current password is required"})
			}
			if bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(currentPassword)) != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Current password is incorrect"})
			}
		}

		h, _ := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		_, err = db.DB.Exec("UPDATE users SET password = ?, password_changed = 1, password_version = COALESCE(password_version, 1) + 1 WHERE id = ?", string(h), user.ID)
		if err != nil {
			log.Printf("Error updating password for user %d: %v", user.ID, err)
			return err
		}
		log.Printf("Password successfully updated for user %d", user.ID)
		return c.NoContent(http.StatusOK)
	})

	r.GET("/user/me", func(c echo.Context) error {
		token := c.Get("user").(*jwt.Token)
		claims := token.Claims.(*models.UserClaims)
		var dbUser models.User
		err := db.DB.QueryRow("SELECT id, username, is_admin, password_changed, can_start, can_stop, can_restart, can_delete, can_shell, is_restricted_access, allowed_containers, is_active FROM users WHERE id = ?", claims.ID).
			Scan(&dbUser.ID, &dbUser.Username, &dbUser.IsAdmin, &dbUser.PasswordChanged, &dbUser.CanStart, &dbUser.CanStop, &dbUser.CanRestart, &dbUser.CanDelete, &dbUser.CanShell, &dbUser.IsRestrictedAccess, &dbUser.AllowedContainers, &dbUser.IsActive)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
		}
		if !dbUser.IsActive {
			return c.JSON(http.StatusForbidden, map[string]string{
				"error": "Account deactivated",
				"code":  "ACCOUNT_DEACTIVATED",
			})
		}
		return c.JSON(http.StatusOK, dbUser)
	})
}
