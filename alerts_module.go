package main

import (
	"docklog/repositories"
	"docklog/services"
)

var alertEngine *services.AlertEngine

func initAlertModule() {
	alertEngine = services.NewAlertEngine(repositories.NewAlertRepository(), notificationService)
	if err := alertEngine.Initialize(); err != nil {
		panic(err)
	}
}
