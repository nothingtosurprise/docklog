package audit

import (
	"log"

	"docklog/db"
)

type NotificationCallback func(userID int, username, action, resource, status, message string)

var OnLogged NotificationCallback

func Log(userID int, username, action, resource, status, message string) {
	_, err := db.DB.Exec(
		"INSERT INTO audit_logs (user_id, username, action, resource, status, message) VALUES (?, ?, ?, ?, ?, ?)",
		userID, username, action, resource, status, message,
	)
	if err != nil {
		log.Printf("Failed to write audit log: %v", err)
		return
	}
	if OnLogged != nil {
		OnLogged(userID, username, action, resource, status, message)
	}
}
