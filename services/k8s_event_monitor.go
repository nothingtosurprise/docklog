package services

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"docklog/config"
	appk8s "docklog/k8s"
	"docklog/models"

	"k8s.io/client-go/kubernetes"
)

const k8sEventPollInterval = 45 * time.Second

// StartK8sEventMonitor polls Kubernetes warning events for enabled k8s alert rules.
func StartK8sEventMonitor(client kubernetes.Interface, engine *AlertEngine) {
	if client == nil || engine == nil {
		return
	}
	go runK8sEventMonitor(client, engine)
}

func runK8sEventMonitor(client kubernetes.Interface, engine *AlertEngine) {
	seen := make(map[string]struct{})
	var seenMu sync.Mutex
	bootstrapped := false

	ticker := time.NewTicker(k8sEventPollInterval)
	defer ticker.Stop()

	poll := func() {
		if len(engine.rulesBySource(models.AlertSourceK8sEvents)) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		namespaces, err := appk8s.ListNamespaces(ctx, client)
		if err != nil {
			config.Debugf("K8s event monitor: list namespaces: %v", err)
			return
		}

		for _, ns := range namespaces {
			events, err := appk8s.ListEvents(ctx, client, ns.Name)
			if err != nil {
				config.Debugf("K8s event monitor: list events in %s: %v", ns.Name, err)
				continue
			}
			for _, event := range events {
				if !strings.EqualFold(strings.TrimSpace(event.Type), "Warning") {
					continue
				}
				key := k8sEventDedupKey(event)
				seenMu.Lock()
				if _, ok := seen[key]; ok {
					seenMu.Unlock()
					continue
				}
				seen[key] = struct{}{}
				shouldProcess := bootstrapped
				seenMu.Unlock()
				if shouldProcess {
					engine.ProcessK8sEvent(event)
				}
			}
		}

		if !bootstrapped {
			bootstrapped = true
			log.Println("K8s event monitor: baseline captured, watching for new warning events")
		}

		seenMu.Lock()
		if len(seen) > 5000 {
			seen = make(map[string]struct{})
		}
		seenMu.Unlock()
	}

	poll()
	for range ticker.C {
		poll()
	}
}
