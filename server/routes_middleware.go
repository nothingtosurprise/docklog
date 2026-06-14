package server

import (
	"net/http"

	"docklog/config"
	"docklog/db"
	"docklog/middleware"
	"docklog/models"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

func (s *Server) setupAPIMiddleware(r *echo.Group) {
	if config.AuthDisabled {
		r.Use(authDisabledMiddleware())
	} else {
		r.Use(echojwt.WithConfig(echojwt.Config{
			NewClaimsFunc: func(c echo.Context) jwt.Claims {
				return new(models.UserClaims)
			},
			SigningKey: config.SecretKey,
		}))
	}
	r.Use(s.passwordChangeMiddleware())
}

func authDisabledMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := &models.UserClaims{
				ID:                 1,
				Username:           "admin",
				IsAdmin:            true,
				IsRestrictedAccess: false,
				CanStart:           true,
				CanStop:            true,
				CanRestart:         true,
				CanDelete:          true,
				IsActive:           true,
			}
			token := &jwt.Token{
				Claims: claims,
				Valid:  true,
			}
			c.Set("user", token)
			return next(c)
		}
	}
}

func (s *Server) passwordChangeMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.AuthDisabled {
				return next(c)
			}
			token := c.Get("user").(*jwt.Token)
			claims := token.Claims.(*models.UserClaims)

			if claims.TokenType == middleware.TokenTypeRefresh {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
			}

			if err := middleware.RefreshClaimsFromDB(claims); err != nil {
				switch err.Error() {
				case "account deactivated":
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "Account deactivated. Please contact administrator.",
						"code":  "ACCOUNT_DEACTIVATED",
					})
				case "session invalidated":
					return c.JSON(http.StatusUnauthorized, map[string]string{
						"error": "Session invalidated. Password was changed. Please re-login.",
						"code":  "SESSION_INVALIDATED",
					})
				default:
					return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User verification failed"})
				}
			}

			var changed bool
			err := db.DB.QueryRow("SELECT password_changed FROM users WHERE id = ?", claims.ID).Scan(&changed)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User verification failed"})
			}

			if c.Path() == "/api/user/change-password" || c.Path() == "/api/user/me" {
				return next(c)
			}

			if !changed {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Password change required", "code": "FORCE_PASSWORD_CHANGE"})
			}

			return next(c)
		}
	}
}

func (s *Server) adminMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := c.Get("user").(*jwt.Token)
			user := token.Claims.(*models.UserClaims)
			var isAdmin bool
			err := db.DB.QueryRow("SELECT is_admin FROM users WHERE id = ? AND is_active = 1", user.ID).Scan(&isAdmin)
			if err != nil || !isAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Admin access required"})
			}
			user.IsAdmin = isAdmin
			return next(c)
		}
	}
}
