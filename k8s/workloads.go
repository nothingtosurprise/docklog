package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"docklog/config"
	"docklog/models"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func requireNamespace(namespace string) error {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if !config.K8sNamespaceAllowed(namespace) {
		return fmt.Errorf("namespace %q is not allowed", namespace)
	}
	return nil
}

func ListDeployments(ctx context.Context, client kubernetes.Interface, namespace string) ([]models.K8sDeployment, error) {
	if err := requireNamespace(namespace); err != nil {
		return nil, err
	}

	res, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments in %s: %w", namespace, err)
	}

	out := make([]models.K8sDeployment, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, summarizeDeployment(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ListHPAs(ctx context.Context, client kubernetes.Interface, namespace string) ([]models.K8sHPA, error) {
	if err := requireNamespace(namespace); err != nil {
		return nil, err
	}

	res, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list hpas in %s: %w", namespace, err)
	}

	out := make([]models.K8sHPA, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, summarizeHPA(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ListServices(ctx context.Context, client kubernetes.Interface, namespace string) ([]models.K8sService, error) {
	if err := requireNamespace(namespace); err != nil {
		return nil, err
	}

	res, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services in %s: %w", namespace, err)
	}

	out := make([]models.K8sService, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, summarizeService(item))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ListEvents(ctx context.Context, client kubernetes.Interface, namespace string) ([]models.K8sEvent, error) {
	if err := requireNamespace(namespace); err != nil {
		return nil, err
	}

	res, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events in %s: %w", namespace, err)
	}

	out := make([]models.K8sEvent, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, summarizeEvent(item))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastTimestamp == out[j].LastTimestamp {
			return out[i].Name < out[j].Name
		}
		return out[i].LastTimestamp > out[j].LastTimestamp
	})
	if len(out) > 200 {
		out = out[:200]
	}
	return out, nil
}

func GetOverview(ctx context.Context, client kubernetes.Interface, namespace string) (models.K8sOverview, error) {
	if err := requireNamespace(namespace); err != nil {
		return models.K8sOverview{}, err
	}

	overview := models.K8sOverview{Namespace: namespace}

	pods, err := ListPods(ctx, client, namespace)
	if err != nil {
		return overview, err
	}
	overview.Pods = len(pods)
	for _, pod := range pods {
		if pod.Phase == "Running" {
			overview.RunningPods++
		}
	}

	deployments, err := ListDeployments(ctx, client, namespace)
	if err != nil {
		return overview, err
	}
	overview.Deployments = len(deployments)

	hpas, err := ListHPAs(ctx, client, namespace)
	if err != nil {
		return overview, err
	}
	overview.HPAs = len(hpas)

	services, err := ListServices(ctx, client, namespace)
	if err != nil {
		return overview, err
	}
	overview.Services = len(services)

	events, err := ListEvents(ctx, client, namespace)
	if err != nil {
		return overview, err
	}
	for _, event := range events {
		if event.Type == "Warning" {
			overview.WarningEvents++
		}
	}

	return overview, nil
}

func summarizeDeployment(dep appsv1.Deployment) models.K8sDeployment {
	images := make([]string, 0, len(dep.Spec.Template.Spec.Containers))
	for _, c := range dep.Spec.Template.Spec.Containers {
		if c.Image != "" {
			images = append(images, c.Image)
		}
	}

	status := "Progressing"
	for _, cond := range dep.Status.Conditions {
		if cond.Type == appsv1.DeploymentAvailable && cond.Status == corev1.ConditionTrue {
			status = "Available"
			break
		}
	}
	if dep.Status.UnavailableReplicas > 0 {
		status = "Degraded"
	}

	created := dep.CreationTimestamp.Time
	if created.IsZero() {
		created = time.Now()
	}

	return models.K8sDeployment{
		Namespace: dep.Namespace,
		Name:      dep.Name,
		UID:       string(dep.UID),
		Replicas:  derefInt32(dep.Spec.Replicas, 1),
		Ready:     dep.Status.ReadyReplicas,
		Available: dep.Status.AvailableReplicas,
		Updated:   dep.Status.UpdatedReplicas,
		Strategy:  string(dep.Spec.Strategy.Type),
		Images:    images,
		Status:    status,
		Created:   created.Unix(),
	}
}

func summarizeHPA(hpa autoscalingv2.HorizontalPodAutoscaler) models.K8sHPA {
	targetKind := ""
	targetName := ""
	if ref := hpa.Spec.ScaleTargetRef; ref.Name != "" {
		targetKind = ref.Kind
		targetName = ref.Name
	}

	status := "Active"
	for _, cond := range hpa.Status.Conditions {
		if cond.Type == autoscalingv2.AbleToScale && cond.Status == corev1.ConditionFalse {
			status = cond.Reason
			if status == "" {
				status = "UnableToScale"
			}
			break
		}
		if cond.Type == autoscalingv2.ScalingLimited && cond.Status == corev1.ConditionTrue {
			status = "ScalingLimited"
		}
	}

	created := hpa.CreationTimestamp.Time
	if created.IsZero() {
		created = time.Now()
	}

	return models.K8sHPA{
		Namespace:       hpa.Namespace,
		Name:            hpa.Name,
		UID:             string(hpa.UID),
		TargetKind:      targetKind,
		TargetName:      targetName,
		MinReplicas:     derefInt32(hpa.Spec.MinReplicas, 1),
		MaxReplicas:     hpa.Spec.MaxReplicas,
		CurrentReplicas: hpa.Status.CurrentReplicas,
		DesiredReplicas: hpa.Status.DesiredReplicas,
		Metrics:         formatHPAMetrics(hpa.Spec.Metrics),
		Status:          status,
		Created:         created.Unix(),
	}
}

func summarizeService(svc corev1.Service) models.K8sService {
	ports := make([]string, 0, len(svc.Spec.Ports))
	for _, p := range svc.Spec.Ports {
		port := fmt.Sprintf("%d/%s", p.Port, p.Protocol)
		if p.NodePort > 0 {
			port += fmt.Sprintf(":%d", p.NodePort)
		}
		ports = append(ports, port)
	}

	selectorParts := make([]string, 0, len(svc.Spec.Selector))
	for k, v := range svc.Spec.Selector {
		selectorParts = append(selectorParts, fmt.Sprintf("%s=%s", k, v))
	}
	sort.Strings(selectorParts)

	created := svc.CreationTimestamp.Time
	if created.IsZero() {
		created = time.Now()
	}

	return models.K8sService{
		Namespace: svc.Namespace,
		Name:      svc.Name,
		UID:       string(svc.UID),
		Type:      string(svc.Spec.Type),
		ClusterIP: svc.Spec.ClusterIP,
		Ports:     strings.Join(ports, ", "),
		Selector:  strings.Join(selectorParts, ", "),
		Created:   created.Unix(),
	}
}

func summarizeEvent(event corev1.Event) models.K8sEvent {
	last := event.LastTimestamp.Time
	if last.IsZero() {
		last = event.EventTime.Time
	}
	if last.IsZero() {
		last = event.CreationTimestamp.Time
	}

	return models.K8sEvent{
		Namespace:       event.Namespace,
		Name:            event.Name,
		UID:             string(event.UID),
		ResourceVersion: event.ResourceVersion,
		Type:            event.Type,
		Reason:          event.Reason,
		Message:         event.Message,
		InvolvedKind:    event.InvolvedObject.Kind,
		InvolvedName:    event.InvolvedObject.Name,
		Count:           event.Count,
		LastTimestamp:   last.Unix(),
	}
}

func formatHPAMetrics(metrics []autoscalingv2.MetricSpec) string {
	if len(metrics) == 0 {
		return "—"
	}
	parts := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		switch metric.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if metric.Resource == nil {
				continue
			}
			if metric.Resource.Target.AverageUtilization != nil {
				parts = append(parts, fmt.Sprintf("%s %d%%", metric.Resource.Name, *metric.Resource.Target.AverageUtilization))
				continue
			}
			if metric.Resource.Target.AverageValue != nil {
				parts = append(parts, fmt.Sprintf("%s %s", metric.Resource.Name, metric.Resource.Target.AverageValue.String()))
			}
		case autoscalingv2.PodsMetricSourceType:
			if metric.Pods != nil && metric.Pods.Metric.Name != "" {
				parts = append(parts, "pods:"+metric.Pods.Metric.Name)
			}
		case autoscalingv2.ObjectMetricSourceType:
			if metric.Object != nil {
				parts = append(parts, fmt.Sprintf("object:%s/%s", metric.Object.DescribedObject.Kind, metric.Object.DescribedObject.Name))
			}
		default:
			parts = append(parts, string(metric.Type))
		}
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, ", ")
}

func derefInt32(v *int32, fallback int32) int32 {
	if v == nil {
		return fallback
	}
	return *v
}
