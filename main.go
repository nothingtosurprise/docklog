package main

import (
	"log"
	"os"

	"docklog/audit"
	appcli "docklog/cli"
	"docklog/config"
	"docklog/containers"
	"docklog/db"
	"docklog/middleware"
	"docklog/server"
)

func main() {
	exit, code, appRuntime := appcli.Dispatch(os.Args)
	if exit {
		os.Exit(code)
	}

	appcli.LogRunMode(appRuntime)

	config.LoadAuthDisabled()
	config.InitSecretKey()
	config.LoadActionFlags()
	middleware.InitClientAccess()
	middleware.InitWSUpgrader()
	containers.Init()

	if err := db.InitDB(db.ServerPath(config.AuthDisabled)); err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}

	initNotificationModule()
	audit.OnLogged = dispatchAuditNotification

	srv := server.New(server.Deps{
		Notifications: notificationService,
	})
	if err := srv.Run(appRuntime); err != nil {
		log.Fatal(err)
	}
}
