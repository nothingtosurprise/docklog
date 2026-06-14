package seed

import (
	"log"

	"docklog/db"

	"golang.org/x/crypto/bcrypt"
)

func Admin() {
	var count int
	db.DB.QueryRow("SELECT COUNT(*) FROM users WHERE username = 'admin'").Scan(&count)
	if count == 0 {
		const plain = "admin123"
		log.Println("Default admin account created (username: admin, password: admin123). Change the password on first login.")

		h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Failed to hash default admin password: %v", err)
		}
		_, err = db.DB.Exec(
			`INSERT INTO users (username, password, is_admin, can_start, can_stop, can_restart, can_delete, can_shell, password_changed)
			 VALUES (?, ?, 1, 1, 1, 1, 1, 1, 0)`,
			"admin", string(h),
		)
		if err != nil {
			log.Fatalf("Failed to create default admin: %v", err)
		}
	}

	_, err := db.DB.Exec(
		`UPDATE users SET can_start = 1, can_stop = 1, can_restart = 1, can_delete = 1, can_shell = 1
		 WHERE is_admin = 1 AND (can_start = 0 OR can_stop = 0 OR can_restart = 0 OR can_delete = 0 OR can_shell = 0)`,
	)
	if err != nil {
		log.Printf("Warning: failed to sync admin action permissions: %v", err)
	}
}
