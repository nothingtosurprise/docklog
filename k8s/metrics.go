package k8s

import (
	"context"
	"fmt"
	"strings"

	"docklog/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

// GetPodStatsNow returns live pod usage when metrics-server is available, plus spec limits.
func GetPodStatsNow(ctx context.Context, client kubernetes.Interface, namespace, name string) (models.K8sPodLiveStats, error) {
	if err := requireNamespace(namespace); err != nil {
		return models.K8sPodLiveStats{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return models.K8sPodLiveStats{}, fmt.Errorf("pod name is required")
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return models.K8sPodLiveStats{}, fmt.Errorf("get pod %s/%s: %w", namespace, name, err)
	}

	cpuLimitMilli, memLimitBytes := podResourceTotals(pod)
	stats := models.K8sPodLiveStats{
		MemoryLimit:      memLimitBytes,
		CPULimitMilli:    cpuLimitMilli,
		MetricsAvailable: false,
	}

	cfg, err := loadRESTConfig()
	if err != nil {
		return stats, nil
	}
	metricsClient, err := metricsclient.NewForConfig(cfg)
	if err != nil {
		return stats, nil
	}

	pm, err := metricsClient.MetricsV1beta1().PodMetricses(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return stats, nil
	}

	var cpuMilli int64
	var memBytes int64
	for _, container := range pm.Containers {
		if q := container.Usage.Cpu(); q != nil {
			cpuMilli += q.MilliValue()
		}
		if q := container.Usage.Memory(); q != nil {
			memBytes += q.Value()
		}
	}

	stats.Memory = memBytes
	stats.MetricsAvailable = true
	if cpuLimitMilli > 0 {
		stats.CPU = float64(cpuMilli) / float64(cpuLimitMilli) * 100
	}
	return stats, nil
}

func podResourceTotals(pod *corev1.Pod) (cpuLimitMilli, memLimitBytes int64) {
	for _, container := range pod.Spec.Containers {
		if q := container.Resources.Limits.Cpu(); q != nil {
			cpuLimitMilli += q.MilliValue()
		}
		if q := container.Resources.Limits.Memory(); q != nil {
			memLimitBytes += q.Value()
		}
	}
	return cpuLimitMilli, memLimitBytes
}
