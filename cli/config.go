package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"docklog/db"
	"docklog/repositories"
	"docklog/services"
)

// PrintConfig prints non-secret effective configuration to stdout.
func PrintConfig(rt Runtime) {
	fmt.Println("DockLog configuration (non-secret):")
	fmt.Printf("  version              %s\n", Version)
	fmt.Printf("  mode                 %s\n", runModeLabel(rt))
	fmt.Printf("  port                 %s\n", envOrDefault("PORT", "8000"))
	fmt.Printf("  db_path              %s\n", db.DefaultPath())
	fmt.Printf("  docker_host          %s\n", envOrDefault("DOCKER_HOST", "unix:///var/run/docker.sock"))
	fmt.Printf("  disable_auth         %s\n", boolEnv("DISABLE_AUTH", false))
	fmt.Printf("  debug_mode           %s\n", boolEnv("DEBUG_MODE", false))
	fmt.Printf("  client_access        %s\n", envOrDefault("CLIENT_ACCESS", "strict"))
	if excluded := strings.TrimSpace(os.Getenv("EXCLUDE_CONTAINERS")); excluded != "" {
		fmt.Printf("  exclude_containers     %s\n", excluded)
	} else {
		fmt.Println("  exclude_containers     (empty; docklog self still hidden)")
	}
	fmt.Printf("  allow_start          %s\n", boolEnv("ALLOW_START", false))
	fmt.Printf("  allow_stop           %s\n", boolEnv("ALLOW_STOP", false))
	fmt.Printf("  allow_restart        %s\n", boolEnv("ALLOW_RESTART", false))
	fmt.Printf("  allow_delete         %s\n", boolEnv("ALLOW_DELETE", false))
	allowShell := boolEnv("ALLOW_SHELL", false) == "true" || boolEnv("ALLOW_BASH", false) == "true"
	fmt.Printf("  allow_shell          %t\n", allowShell)
	printNotificationConfigLines()
	if url := strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")); url != "" {
		fmt.Printf("  control_plane_url    %s\n", url)
	}
	if secret := strings.TrimSpace(os.Getenv("SECRET_KEY")); secret != "" {
		fmt.Println("  secret_key           (set)")
	} else {
		fmt.Println("  secret_key           (default; change in production)")
	}
}

func printNotificationConfigLines() {
	dbPath := db.DefaultPath()
	if dbPath == ":memory:" {
		fmt.Println("  notifications          configure in admin UI")
		return
	}
	if err := db.OpenExisting(dbPath); err != nil {
		if errors.Is(err, db.ErrDBNotFound) {
			fmt.Printf("  notifications          no database at %s\n", dbPath)
			return
		}
		fmt.Println("  notifications          unavailable (database not readable)")
		return
	}
	svc := services.NewNotificationService(repositories.NewNotificationRepository())
	for _, line := range svc.CLIConfigLines() {
		fmt.Println(line)
	}
}
