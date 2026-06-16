package k8s

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type PodLogRequest struct {
	Container  string
	Tail       int
	Since      string
	Until      string
	Timestamps bool
}

func GetPod(ctx context.Context, client kubernetes.Interface, namespace, podName string) (*corev1.Pod, error) {
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s/%s: %w", namespace, podName, err)
	}
	return pod, nil
}

func DefaultPodContainer(pod *corev1.Pod) string {
	if pod == nil {
		return ""
	}
	if len(pod.Spec.Containers) == 1 {
		return pod.Spec.Containers[0].Name
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Running != nil {
			return status.Name
		}
	}
	if len(pod.Spec.Containers) > 0 {
		return pod.Spec.Containers[0].Name
	}
	return ""
}

func ReadPodLogs(ctx context.Context, client kubernetes.Interface, namespace, podName string, req PodLogRequest) ([]string, error) {
	pod, err := GetPod(ctx, client, namespace, podName)
	if err != nil {
		return nil, err
	}

	container := strings.TrimSpace(req.Container)
	if container == "" {
		container = DefaultPodContainer(pod)
	}

	tail := int64(req.Tail)
	if tail <= 0 {
		tail = 100
	}

	opts := &corev1.PodLogOptions{
		Container:  container,
		Timestamps: req.Timestamps,
		TailLines:  &tail,
	}
	if since := strings.TrimSpace(req.Since); since != "" {
		if ts, err := parseLogTimestamp(since); err == nil && !ts.IsZero() {
			meta := metav1.NewTime(ts)
			opts.SinceTime = &meta
		}
	}

	stream, err := client.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream pod logs %s/%s: %w", namespace, podName, err)
	}
	defer stream.Close()

	lines, err := readLogLines(stream)
	if err != nil {
		return nil, err
	}

	if until := strings.TrimSpace(req.Until); until != "" {
		untilTime, err := parseLogTimestamp(until)
		if err != nil {
			return nil, fmt.Errorf("invalid until timestamp: %w", err)
		}
		lines = filterLinesUntil(lines, untilTime)
	}

	if len(lines) > int(tail) {
		lines = lines[len(lines)-int(tail):]
	}

	return lines, nil
}

func StreamPodLogs(ctx context.Context, client kubernetes.Interface, namespace, podName, container string, tail int) (io.ReadCloser, error) {
	pod, err := GetPod(ctx, client, namespace, podName)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(container) == "" {
		container = DefaultPodContainer(pod)
	}
	if tail <= 0 {
		tail = 100
	}
	tailLines := int64(tail)

	opts := &corev1.PodLogOptions{
		Container:  container,
		Follow:       true,
		Timestamps:   true,
		TailLines:    &tailLines,
	}
	return client.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
}

func CountPodLogs(ctx context.Context, client kubernetes.Interface, namespace, podName, container string) (int, error) {
	lines, err := ReadPodLogs(ctx, client, namespace, podName, PodLogRequest{
		Container:  container,
		Tail:       100000,
		Timestamps: true,
	})
	if err != nil {
		return 0, err
	}
	return len(lines), nil
}

func readLogLines(stream io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(stream)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

func filterLinesUntil(lines []string, until time.Time) []string {
	if until.IsZero() {
		return lines
	}
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 0 {
			continue
		}
		ts, err := parseLogTimestamp(parts[0])
		if err != nil {
			continue
		}
		if !ts.After(until) {
			filtered = append(filtered, line)
		}
	}
	return filtered
}

func parseLogTimestamp(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, nil
	}
	if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return ts, nil
	}
	if ts, err := time.Parse(time.RFC3339, raw); err == nil {
		return ts, nil
	}
	if unix, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(unix, 0), nil
	}
	return time.Time{}, fmt.Errorf("invalid timestamp %q", raw)
}
