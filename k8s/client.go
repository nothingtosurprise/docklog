package k8s

import (
	"fmt"
	"log"
	"os"
	"strings"

	"docklog/config"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const kubeConfigHelp = "mount a kubeconfig (e.g. -v ~/.kube:/app/kube:ro and KUBECONFIG=/app/kube/config) or deploy DockLog in-cluster with a ServiceAccount"

func NewClient() (kubernetes.Interface, error) {
	cfg, err := loadRESTConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func NewRESTConfig() (*rest.Config, error) {
	return loadRESTConfig()
}

func loadRESTConfig() (*rest.Config, error) {
	if cfg, err := rest.InClusterConfig(); err == nil {
		return cfg, nil
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	explicitPath := config.KubeConfig
	if explicitPath == "" {
		explicitPath = os.Getenv("KUBECONFIG")
	}
	if explicitPath != "" {
		loadingRules.ExplicitPath = explicitPath
		if _, err := os.Stat(explicitPath); err != nil {
			return nil, fmt.Errorf("kubernetes client config: KUBECONFIG %q is not accessible (%v); %s", explicitPath, err, kubeConfigHelp)
		}
	} else if _, err := os.Stat(loadingRules.GetDefaultFilename()); os.IsNotExist(err) {
		return nil, fmt.Errorf("kubernetes client config: no kubeconfig found at %q; set KUBECONFIG or %s", loadingRules.GetDefaultFilename(), kubeConfigHelp)
	}

	overrides := &clientcmd.ConfigOverrides{}
	if config.K8sContext != "" {
		overrides.CurrentContext = config.K8sContext
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
	cfg, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kubernetes client config: %w; %s", err, kubeConfigHelp)
	}

	applyAPIServerOverrides(cfg)
	return cfg, nil
}

func applyAPIServerOverrides(cfg *rest.Config) {
	original := cfg.Host

	if override := strings.TrimSpace(config.K8sAPIServer); override != "" {
		log.Printf("Kubernetes API server override: %s", override)
		cfg.Host = override
	}

	if config.K8sRewriteLocalhost {
		rewriteLocalhostAPIHost(cfg, original)
	}

	if shouldUseDockerDesktopTLS(cfg.Host, original) {
		applyDockerDesktopTLS(cfg, original)
	}
}

func rewriteLocalhostAPIHost(cfg *rest.Config, original string) {
	if strings.Contains(cfg.Host, "127.0.0.1") {
		cfg.Host = strings.Replace(cfg.Host, "127.0.0.1", "host.docker.internal", 1)
		log.Printf("Kubernetes API server rewritten for Docker: %s -> %s", original, cfg.Host)
		return
	}
	if strings.Contains(cfg.Host, "localhost") {
		cfg.Host = strings.Replace(cfg.Host, "localhost", "host.docker.internal", 1)
		log.Printf("Kubernetes API server rewritten for Docker: %s -> %s", original, cfg.Host)
	}
}

func shouldUseDockerDesktopTLS(host, original string) bool {
	if config.K8sInsecureSkipTLS {
		return true
	}
	if strings.Contains(host, "host.docker.internal") {
		return true
	}
	if config.K8sRewriteLocalhost && (strings.Contains(original, "127.0.0.1") || strings.Contains(original, "localhost")) {
		return true
	}
	return false
}

func applyDockerDesktopTLS(cfg *rest.Config, original string) {
	if config.K8sInsecureSkipTLS {
		disableTLSVerify(cfg)
		log.Printf("WARNING: Kubernetes TLS verification disabled (K8S_INSECURE_SKIP_TLS_VERIFY=true)")
		return
	}

	// Docker Desktop serves a cert for localhost while the container connects via host.docker.internal.
	cfg.TLSClientConfig.ServerName = dockerDesktopServerName(original)
	log.Printf("Kubernetes TLS server name set for Docker Desktop: %s", cfg.TLSClientConfig.ServerName)
}

func dockerDesktopServerName(original string) string {
	if strings.Contains(original, "127.0.0.1") || strings.Contains(original, "localhost") {
		return "localhost"
	}
	return "kubernetes"
}

func disableTLSVerify(cfg *rest.Config) {
	cfg.TLSClientConfig.Insecure = true
	cfg.TLSClientConfig.CAFile = ""
	cfg.TLSClientConfig.CAData = nil
	cfg.CAFile = ""
	cfg.CAData = nil
}
