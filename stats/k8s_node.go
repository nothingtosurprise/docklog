package stats

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	appk8s "docklog/k8s"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
)

var (
	k8sNodeMu     sync.RWMutex
	k8sNodeClient kubernetes.Interface
	k8sNodeName   string
	k8sNodeOnce   sync.Once
)

func SetKubernetesClient(client kubernetes.Interface) {
	k8sNodeMu.Lock()
	defer k8sNodeMu.Unlock()
	k8sNodeClient = client
	k8sNodeName = ""
	k8sNodeOnce = sync.Once{}
}

func k8sHostMemory() (total, used uint64, ok bool) {
	snapshot, ok := k8sNodeSnapshot()
	if !ok {
		return 0, 0, false
	}
	return snapshot.totalMemory, snapshot.usedMemory, snapshot.totalMemory > 0
}

func k8sHostLogicalCPUs() (int, bool) {
	snapshot, ok := k8sNodeSnapshot()
	if !ok || snapshot.logicalCPUs <= 0 {
		return 0, false
	}
	return snapshot.logicalCPUs, true
}

func k8sHostCPUPercent() (float64, bool) {
	snapshot, ok := k8sNodeSnapshot()
	if !ok {
		return 0, false
	}
	return snapshot.cpuPercent, true
}

type k8sNodeSnapshotData struct {
	totalMemory  uint64
	usedMemory   uint64
	logicalCPUs  int
	cpuPercent   float64
}

func k8sNodeSnapshot() (k8sNodeSnapshotData, bool) {
	k8sNodeMu.RLock()
	client := k8sNodeClient
	k8sNodeMu.RUnlock()
	if client == nil {
		return k8sNodeSnapshotData{}, false
	}

	nodeName, err := resolveKubernetesNodeName(client)
	if err != nil || nodeName == "" {
		return k8sNodeSnapshotData{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return k8sNodeSnapshotData{}, false
	}

	snapshot := k8sNodeSnapshotData{
		totalMemory: uint64(node.Status.Capacity.Memory().Value()),
		logicalCPUs: int(node.Status.Capacity.Cpu().Value()),
	}

	if metricsClient, err := newMetricsClient(); err == nil {
		if metrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{}); err == nil {
			snapshot.usedMemory = uint64(metrics.Usage.Memory().Value())
			capacityMilli := node.Status.Capacity.Cpu().MilliValue()
			if capacityMilli > 0 {
				snapshot.cpuPercent = float64(metrics.Usage.Cpu().MilliValue()) / float64(capacityMilli) * 100
			}
		}
	}

	if snapshot.usedMemory == 0 && snapshot.totalMemory > 0 {
		if alloc := node.Status.Allocatable.Memory().Value(); alloc > 0 && alloc < int64(snapshot.totalMemory) {
			snapshot.usedMemory = snapshot.totalMemory - uint64(alloc)
		}
	}

	if snapshot.totalMemory == 0 && snapshot.logicalCPUs == 0 {
		return k8sNodeSnapshotData{}, false
	}
	return snapshot, true
}

func newMetricsClient() (metricsclient.Interface, error) {
	cfg, err := appk8s.NewRESTConfig()
	if err != nil {
		return nil, err
	}
	return metricsclient.NewForConfig(cfg)
}

func resolveKubernetesNodeName(client kubernetes.Interface) (string, error) {
	k8sNodeMu.RLock()
	cached := k8sNodeName
	k8sNodeMu.RUnlock()
	if cached != "" {
		return cached, nil
	}

	var resolved string
	var resolveErr error
	k8sNodeOnce.Do(func() {
		if name := strings.TrimSpace(os.Getenv("NODE_NAME")); name != "" {
			resolved = name
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		podName, _ := os.Hostname()
		namespace := strings.TrimSpace(os.Getenv("POD_NAMESPACE"))
		if namespace == "" {
			if raw, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
				namespace = strings.TrimSpace(string(raw))
			}
		}
		if podName == "" || namespace == "" {
			resolveErr = os.ErrInvalid
			return
		}

		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			resolveErr = err
			return
		}
		if pod.Spec.NodeName == "" {
			resolveErr = os.ErrInvalid
			return
		}
		resolved = pod.Spec.NodeName
	})

	if resolved == "" {
		return "", resolveErr
	}

	k8sNodeMu.Lock()
	k8sNodeName = resolved
	k8sNodeMu.Unlock()
	return resolved, nil
}

func resetKubernetesNodeCacheForTest() {
	k8sNodeMu.Lock()
	defer k8sNodeMu.Unlock()
	k8sNodeClient = nil
	k8sNodeName = ""
	k8sNodeOnce = sync.Once{}
}
