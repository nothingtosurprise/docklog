package server

import (
	"net/http"

	"docklog/config"
	"docklog/middleware"

	"github.com/labstack/echo/v4"
)

func (s *Server) registerPublicRoutes() {
	s.echo.GET("/api/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"auth_disabled": config.AuthDisabled,
			"allow_start":   config.CanStart,
			"allow_stop":    config.CanStop,
			"allow_restart": config.CanRestart,
			"allow_delete":  config.CanDelete,
			"allow_shell":   config.AllowShell,
			"runtime_mode":    config.RuntimeMode,
			"k8s_namespaces":  config.K8sNamespaces,
			"k8s_default_ns":  config.DefaultK8sNamespace(),
			"k8s_context":     config.K8sContext,
			"k8s_available":   config.K8sAvailable,
			"k8s_error":       config.K8sConfigError,
			"client_access": middleware.ClientAccessConfig(),
		})
	})
}
