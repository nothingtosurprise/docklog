package server

import (
	"log"
	"net/http"
	"os"
	"strings"

	"docklog/audit"
	"docklog/cli"
	"docklog/config"
	appk8s "docklog/k8s"
	appmiddleware "docklog/middleware"
	"docklog/seed"
	"docklog/services"
	"docklog/stats"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/moby/moby/client"
)

type Server struct {
	echo *echo.Echo
	deps Deps
}

func New(deps Deps) *Server {
	return &Server{
		echo: echo.New(),
		deps: deps,
	}
}

func (s *Server) Run(rt cli.Runtime) error {
	if config.TrustProxy {
		s.echo.IPExtractor = echo.ExtractIPFromXFFHeader()
	}

	if config.DebugMode {
		s.echo.Use(echomiddleware.Logger())
		log.Println("Debug mode enabled: verbose debug and HTTP access logs are ON")
	} else {
		log.Println("Debug mode disabled: verbose debug and HTTP access logs are OFF")
	}
	s.echo.Use(echomiddleware.Recover())
	s.echo.Use(appmiddleware.SecurityHeadersMiddleware())
	s.echo.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOriginFunc: func(origin string) (bool, error) {
			return appmiddleware.CorsOriginAllowed(origin), nil
		},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderAuthorization,
			echo.HeaderContentType,
			appmiddleware.HeaderDockLogClient,
		},
	}))
	s.echo.Use(appmiddleware.ClientAccessMiddleware())

	if config.DockerEnabled() {
		if s.deps.Docker == nil {
			cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
			if err != nil {
				return err
			}
			s.deps.Docker = cli
		}

		services.StartHealthMonitor(s.deps.Docker, audit.Log, s.deps.Alerts)
		services.StartContainerEventMonitor(s.deps.Docker, audit.Log, s.deps.Alerts)
		services.StartLogAlertMonitor(s.deps.Docker, s.deps.Alerts)
		services.StartMetricAlertEvaluator(s.deps.Docker, s.deps.Alerts)
	}

	if config.KubernetesEnabled() {
		if s.deps.K8s == nil {
			k8sClient, err := appk8s.NewClient()
			if err != nil {
				config.K8sAvailable = false
				config.K8sConfigError = err.Error()
				log.Printf("WARNING: Kubernetes is enabled but unavailable: %v", err)
			} else {
				s.deps.K8s = k8sClient
				config.K8sAvailable = true
				log.Println("Kubernetes runtime enabled")
			}
		}
		if s.deps.K8s != nil && s.deps.Alerts != nil {
			config.K8sAvailable = true
			services.StartK8sEventMonitor(s.deps.K8s, s.deps.Alerts)
		}
	}

	stats.SetKubernetesClient(s.deps.K8s)
	stats.StartCollector(s.deps.Docker)

	seed.Admin()

	s.registerAuthRoutes()
	s.registerPublicRoutes()

	api := s.echo.Group("/api")
	s.setupAPIMiddleware(api)
	s.registerSystemRoutes(api)
	if config.DockerEnabled() {
		s.registerContainerRoutes(api)
	}
	if config.KubernetesEnabled() {
		s.registerK8sRoutes(api)
		s.registerK8sLogRoutes(api)
	}
	s.registerUserRoutes(api)
	s.registerAdminRoutes(api)
	s.registerWebSocketRoutes()
	if config.KubernetesEnabled() {
		s.registerK8sWebSocketRoutes()
	}

	if rt.ServeFrontend {
		s.echo.Use(echomiddleware.StaticWithConfig(echomiddleware.StaticConfig{
			Root:   "frontend/dist",
			Browse: false,
			HTML5:  true,
			Skipper: func(c echo.Context) bool {
				return strings.HasPrefix(c.Path(), "/api") || strings.HasPrefix(c.Path(), "/ws")
			},
		}))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("DockLog %s listening on %s\n", cli.Version, port)
	return s.echo.Start(port)
}
