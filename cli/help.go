package cli

import "fmt"

// PrintHelp prints CLI usage. Optional topic shows help for a single command.
func PrintHelp(topic []string) {
	if len(topic) > 0 {
		switch topic[0] {
		case "server":
			fmt.Println(`docklog server: run the full DockLog dashboard (API, WebSockets, embedded Vue UI).

Environment variables configure auth, permissions, and ports. See README.md.`)
			return
		case "agent":
			fmt.Println(`docklog agent: run DockLog as a fleet agent with the local UI enabled.

Sets DOCKLOG_MODE=agent. Optional CONTROL_PLANE_URL registers this host with a remote DockLog control plane (future).`)
			return
		case "agent-only":
			fmt.Println(`docklog agent-only: headless agent (API + WebSockets only, no bundled web UI).

Sets DOCKLOG_MODE=agent and DOCKLOG_AGENT_ONLY=true. Use when a remote UI or mobile app connects to this host.`)
			return
		case "reset-password":
			fmt.Println(`docklog reset-password <username> <new-password>

Reset a user password in the SQLite database. Invalidates existing sessions.
Requires DB_PATH to point at the on-disk database (not :memory:).`)
			return
		}
	}

	fmt.Print(`DockLog: self-hosted Docker logs, RBAC, and monitoring.

Usage:
  docklog [command]

Commands:
  server          Run full dashboard with embedded web UI (default)
  agent           Run as fleet agent (local UI + API)
  agent-only      Run headless agent (API/WebSockets only)
  reset-password  Reset a user password in SQLite
  config          Print effective non-secret configuration
  version         Print version
  help            Show this help

Examples:
  docklog
  docklog server
  docklog agent-only
  docklog reset-password admin 'NewSecurePass1'
  docklog config

Install globally:
  make install
  # or: go install .

Docker:
  docker exec docklog docklog reset-password admin 'NewSecurePass1'
`)
}
