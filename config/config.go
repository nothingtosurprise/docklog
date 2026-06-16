package config

import (
	"log"
	"os"
	"strings"
)

const defaultSecretKey = "secret-key-change-this"

var (
	AuthDisabled bool
	CanStart     bool
	CanStop      bool
	CanRestart   bool
	CanDelete    bool
	AllowShell   bool
	DebugMode    bool
	RuntimeMode   string
	K8sNamespaces []string
	K8sContext    string
	KubeConfig         string
	K8sAPIServer         string
	K8sRewriteLocalhost  bool
	K8sInsecureSkipTLS   bool
	K8sAvailable         bool
	K8sConfigError string
	SecretKey     []byte
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

func LoadDebugMode() {
	DebugMode = os.Getenv("DEBUG_MODE") == "true"
}

func LoadRuntimeConfig() {
	RuntimeMode = os.Getenv("RUNTIME_MODE")
	if RuntimeMode == "" {
		RuntimeMode = "docker"
	}
	switch RuntimeMode {
	case "docker", "kubernetes", "both":
	default:
		log.Printf("Invalid RUNTIME_MODE=%q. Falling back to docker.", RuntimeMode)
		RuntimeMode = "docker"
	}

	K8sNamespaces = parseK8sNamespaces(os.Getenv("K8S_NAMESPACES"))
	K8sContext = os.Getenv("K8S_CONTEXT")
	KubeConfig = os.Getenv("KUBECONFIG")
	K8sAPIServer = strings.TrimSpace(os.Getenv("K8S_API_SERVER"))
	switch strings.ToLower(strings.TrimSpace(os.Getenv("K8S_REWRITE_LOCALHOST"))) {
	case "true":
		K8sRewriteLocalhost = true
	case "false":
		K8sRewriteLocalhost = false
	default:
		_, err := os.Stat("/.dockerenv")
		K8sRewriteLocalhost = err == nil
	}
	switch strings.ToLower(strings.TrimSpace(os.Getenv("K8S_INSECURE_SKIP_TLS_VERIFY"))) {
	case "true":
		K8sInsecureSkipTLS = true
	case "false":
		K8sInsecureSkipTLS = false
	default:
		K8sInsecureSkipTLS = false
	}
}

func parseK8sNamespaces(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// DefaultK8sNamespace returns the first configured namespace for UI defaults.
func DefaultK8sNamespace() string {
	if len(K8sNamespaces) > 0 {
		return K8sNamespaces[0]
	}
	return "default"
}

// K8sNamespaceAllowed reports whether a namespace is in scope for DockLog.
// An empty K8S_NAMESPACES list means all namespaces allowed by cluster RBAC.
func K8sNamespaceAllowed(namespace string) bool {
	if len(K8sNamespaces) == 0 {
		return true
	}
	namespace = strings.TrimSpace(namespace)
	for _, allowed := range K8sNamespaces {
		if allowed == namespace {
			return true
		}
	}
	return false
}

func DockerEnabled() bool {
	return RuntimeMode == "docker" || RuntimeMode == "both"
}

func KubernetesEnabled() bool {
	return RuntimeMode == "kubernetes" || RuntimeMode == "both"
}

func Debugf(format string, args ...interface{}) {
	if !DebugMode {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}
