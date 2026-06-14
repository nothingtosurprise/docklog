package server

import (
	"log"
	"net/http"
	"os"
	"strings"

	"docklog/audit"
	"docklog/cli"
	"docklog/config"
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

	s.echo.Use(echomiddleware.Logger())
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

	if s.deps.Docker == nil {
		cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
		if err != nil {
			return err
		}
		s.deps.Docker = cli
	}

	stats.StartCollector(s.deps.Docker)
	services.StartHealthMonitor(s.deps.Docker, audit.Log)
	services.StartContainerEventMonitor(s.deps.Docker, audit.Log)
	seed.Admin()

	s.registerAuthRoutes()
	s.registerPublicRoutes()

	api := s.echo.Group("/api")
	s.setupAPIMiddleware(api)
	s.registerContainerRoutes(api)
	s.registerUserRoutes(api)
	s.registerAdminRoutes(api)
	s.registerWebSocketRoutes()

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
