package config

import (
	"log"
	"os"
)

const defaultSecretKey = "secret-key-change-this"

var (
	AuthDisabled bool
	CanStart     bool
	CanStop      bool
	CanRestart   bool
	CanDelete    bool
	AllowShell   bool
	SecretKey    []byte
	TrustProxy   bool
)

func InitSecretKey() {
	key := os.Getenv("SECRET_KEY")
	if key == "" {
		key = defaultSecretKey
	}
	SecretKey = []byte(key)

	if AuthDisabled {
		return
	}
	if key == defaultSecretKey {
		env := os.Getenv("ENV")
		if env == "production" || os.Getenv("GO_ENV") == "production" {
			log.Fatalf("SECRET_KEY must be set in production")
		}
		log.Println("WARNING: Using default SECRET_KEY. Set the SECRET_KEY environment variable before deploying.")
	}
}

func LoadActionFlags() {
	getEnvBool := func(key string, defaultVal bool) bool {
		val := os.Getenv(key)
		if val == "" {
			return defaultVal
		}
		return val == "true"
	}

	CanStart = getEnvBool("ALLOW_START", false)
	CanStop = getEnvBool("ALLOW_STOP", false)
	CanRestart = getEnvBool("ALLOW_RESTART", false)
	CanDelete = getEnvBool("ALLOW_DELETE", false)
	AllowShell = getEnvBool("ALLOW_SHELL", false) || getEnvBool("ALLOW_BASH", false)
}

func LoadAuthDisabled() {
	AuthDisabled = os.Getenv("DISABLE_AUTH") == "true" || os.Getenv("NO_AUTH") == "true"
}
