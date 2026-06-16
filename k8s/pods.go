package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"docklog/config"
	"docklog/models"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/kubernetes"
)

func ListNamespaces(ctx context.Context, client kubernetes.Interface) ([]models.K8sNamespace, error) {
	if len(config.K8sNamespaces) > 0 {
		out := make([]models.K8sNamespace, 0, len(config.K8sNamespaces))
		for _, ns := range config.K8sNamespaces {
			out = append(out, models.K8sNamespace{Name: ns})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil
	}

	res, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	out := make([]models.K8sNamespace, 0, len(res.Items))
	for _, item := range res.Items {
		out = append(out, models.K8sNamespace{Name: item.Name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func ListPods(ctx context.Context, client kubernetes.Interface, namespace string) ([]models.K8sPod, error) {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if !config.K8sNamespaceAllowed(namespace) {
		return nil, fmt.Errorf("namespace %q is not allowed", namespace)
	}

	res, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods in %s: %w", namespace, err)
	}

	out := make([]models.K8sPod, 0, len(res.Items))
	for _, pod := range res.Items {
		out = append(out, summarizePod(pod))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func GetPodDetail(ctx context.Context, client kubernetes.Interface, namespace, podName string) (models.K8sPodDetail, error) {
	namespace = strings.TrimSpace(namespace)
	podName = strings.TrimSpace(podName)
	if namespace == "" || podName == "" {
		return models.K8sPodDetail{}, fmt.Errorf("namespace and pod are required")
	}
	if !config.K8sNamespaceAllowed(namespace) {
		return models.K8sPodDetail{}, fmt.Errorf("namespace %q is not allowed", namespace)
	}

	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return models.K8sPodDetail{}, fmt.Errorf("get pod %s/%s: %w", namespace, podName, err)
	}

	ready := 0
	total := len(pod.Spec.Containers)
	var restarts int32
	containers := make([]models.K8sPodContainer, 0, len(pod.Spec.Containers))
	initContainers := make([]models.K8sPodContainer, 0, len(pod.Spec.InitContainers))
	stateSummaries := make([]string, 0, len(pod.Status.ContainerStatuses))
	conditions := make([]string, 0, len(pod.Status.Conditions))
	volumes := make([]string, 0, len(pod.Spec.Volumes))
	owners := make([]string, 0, len(pod.OwnerReferences))
	linkedConfigMaps := map[string]struct{}{}
	linkedSecrets := map[string]struct{}{}

	statusByName := make(map[string]corev1.ContainerStatus, len(pod.Status.ContainerStatuses))
	for _, status := range pod.Status.ContainerStatuses {
		statusByName[status.Name] = status
		if status.Ready {
			ready++
		}
		restarts += status.RestartCount
		state := containerStateLabel(status.State)
		stateSummaries = append(stateSummaries, fmt.Sprintf("%s: %s", status.Name, state))
	}

	initStatusByName := make(map[string]corev1.ContainerStatus, len(pod.Status.InitContainerStatuses))
	for _, status := range pod.Status.InitContainerStatuses {
		initStatusByName[status.Name] = status
	}

	for _, c := range pod.Spec.Containers {
		status := statusByName[c.Name]
		collectContainerRefs(c, linkedConfigMaps, linkedSecrets)
		containers = append(containers, models.K8sPodContainer{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        status.Ready,
			RestartCount: status.RestartCount,
			State:        containerStateLabel(status.State),
			CPURequest:   resourceValue(c.Resources.Requests.Cpu()),
			CPULimit:     resourceValue(c.Resources.Limits.Cpu()),
			MemoryRequest: resourceValue(c.Resources.Requests.Memory()),
			MemoryLimit:  resourceValue(c.Resources.Limits.Memory()),
			Ports:        containerPorts(c.Ports),
			Env:          envVars(c.Env),
			VolumeMounts: volumeMounts(c.VolumeMounts),
		})
	}
	for _, c := range pod.Spec.InitContainers {
		status := initStatusByName[c.Name]
		collectContainerRefs(c, linkedConfigMaps, linkedSecrets)
		initContainers = append(initContainers, models.K8sPodContainer{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        status.Ready,
			RestartCount: status.RestartCount,
			State:        containerStateLabel(status.State),
			CPURequest:   resourceValue(c.Resources.Requests.Cpu()),
			CPULimit:     resourceValue(c.Resources.Limits.Cpu()),
			MemoryRequest: resourceValue(c.Resources.Requests.Memory()),
			MemoryLimit:  resourceValue(c.Resources.Limits.Memory()),
			Ports:        containerPorts(c.Ports),
			Env:          envVars(c.Env),
			VolumeMounts: volumeMounts(c.VolumeMounts),
		})
	}

	for _, cond := range pod.Status.Conditions {
		conditions = append(conditions, fmt.Sprintf("%s=%s (%s)", cond.Type, cond.Status, strings.TrimSpace(cond.Reason)))
	}
	for _, vol := range pod.Spec.Volumes {
		collectVolumeRefs(vol, linkedConfigMaps, linkedSecrets)
		volumes = append(volumes, volumeSummary(vol))
	}
	for _, owner := range pod.OwnerReferences {
		owners = append(owners, fmt.Sprintf("%s/%s", owner.Kind, owner.Name))
	}
	workloads := resolveLinkedWorkloads(ctx, client, pod.Namespace, pod.OwnerReferences)
	linkedServices := resolveLinkedServices(ctx, client, pod.Namespace, pod.Labels)
	linkedHPAs := resolveLinkedHPAs(ctx, client, pod.Namespace, workloads)
	configMaps := resolveConfigMaps(ctx, client, pod.Namespace, linkedConfigMaps)
	secrets := resolveSecrets(ctx, client, pod.Namespace, linkedSecrets)

	created := pod.CreationTimestamp.Time
	if created.IsZero() {
		created = time.Now()
	}

	return models.K8sPodDetail{
		Namespace:         pod.Namespace,
		Name:              pod.Name,
		UID:               string(pod.UID),
		Phase:             string(pod.Status.Phase),
		Node:              pod.Spec.NodeName,
		IP:                pod.Status.PodIP,
		ServiceAccount:    pod.Spec.ServiceAccountName,
		HostIP:            pod.Status.HostIP,
		QoSClass:          string(pod.Status.QOSClass),
		Ready:             fmt.Sprintf("%d/%d", ready, total),
		Restarts:          restarts,
		Created:           created.Unix(),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		Status:            podStatusMessage(*pod),
		Conditions:        conditions,
		Volumes:           volumes,
		OwnerReferences:   owners,
		LinkedServices:    linkedServices,
		LinkedHPAs:        linkedHPAs,
		LinkedWorkloads:   workloads,
		ConfigMaps:        configMaps,
		Secrets:           secrets,
		Containers:        containers,
		InitContainers:    initContainers,
		ContainerStatuses: stateSummaries,
	}, nil
}

func ExecutePodAction(ctx context.Context, client kubernetes.Interface, namespace, podName, action string) error {
	namespace = strings.TrimSpace(namespace)
	podName = strings.TrimSpace(podName)
	action = strings.TrimSpace(strings.ToLower(action))
	if namespace == "" || podName == "" {
		return fmt.Errorf("namespace and pod are required")
	}
	if !config.K8sNamespaceAllowed(namespace) {
		return fmt.Errorf("namespace %q is not allowed", namespace)
	}
	switch action {
	case "start", "stop", "restart", "remove":
	default:
		return fmt.Errorf("invalid action")
	}

	// Kubernetes pods are typically controlled by a workload. Deleting a pod is
	// the closest equivalent for stop/restart/remove, and for start we rely on
	// controller reconciliation to recreate if needed.
	propagation := metav1.DeletePropagationForeground
	grace := int64(0)
	return client.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{
		GracePeriodSeconds: &grace,
		PropagationPolicy:  &propagation,
	})
}

func summarizePod(pod corev1.Pod) models.K8sPod {
	ready := 0
	total := len(pod.Spec.Containers)
	var restarts int32
	images := make([]string, 0, total)
	seenImages := map[string]struct{}{}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Ready {
			ready++
		}
		restarts += cs.RestartCount
		if cs.Image != "" {
			if _, ok := seenImages[cs.Image]; !ok {
				seenImages[cs.Image] = struct{}{}
				images = append(images, cs.Image)
			}
		}
	}
	for _, c := range pod.Spec.Containers {
		if c.Image == "" {
			continue
		}
		if _, ok := seenImages[c.Image]; !ok {
			seenImages[c.Image] = struct{}{}
			images = append(images, c.Image)
		}
	}

	created := pod.CreationTimestamp.Time
	if created.IsZero() {
		created = time.Now()
	}

	return models.K8sPod{
		Namespace: pod.Namespace,
		Name:      pod.Name,
		UID:       string(pod.UID),
		Phase:     string(pod.Status.Phase),
		Node:      pod.Spec.NodeName,
		IP:        pod.Status.PodIP,
		Ready:     fmt.Sprintf("%d/%d", ready, total),
		Restarts:  restarts,
		Created:   created.Unix(),
		Images:    images,
		Status:    podStatusMessage(pod),
	}
}

func podStatusMessage(pod corev1.Pod) string {
	if pod.DeletionTimestamp != nil {
		return "Terminating"
	}
	if pod.Status.Message != "" {
		return pod.Status.Message
	}
	if pod.Status.Reason != "" {
		return pod.Status.Reason
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			return cs.State.Waiting.Reason
		}
	}
	return string(pod.Status.Phase)
}

func containerStateLabel(state corev1.ContainerState) string {
	switch {
	case state.Running != nil:
		return "Running"
	case state.Waiting != nil:
		if state.Waiting.Reason != "" {
			return state.Waiting.Reason
		}
		return "Waiting"
	case state.Terminated != nil:
		if state.Terminated.Reason != "" {
			return state.Terminated.Reason
		}
		return "Terminated"
	default:
		return "Unknown"
	}
}

func resourceValue(q *resource.Quantity) string {
	if q == nil {
		return "-"
	}
	v := q.String()
	if strings.TrimSpace(v) == "" {
		return "-"
	}
	return v
}

func containerPorts(ports []corev1.ContainerPort) string {
	if len(ports) == 0 {
		return "-"
	}
	out := make([]string, 0, len(ports))
	for _, p := range ports {
		proto := string(p.Protocol)
		if proto == "" {
			proto = "TCP"
		}
		out = append(out, fmt.Sprintf("%d/%s", p.ContainerPort, proto))
	}
	return strings.Join(out, ", ")
}

func envVars(vars []corev1.EnvVar) []string {
	if len(vars) == 0 {
		return nil
	}
	out := make([]string, 0, len(vars))
	for _, v := range vars {
		if strings.TrimSpace(v.Value) != "" {
			out = append(out, fmt.Sprintf("%s=%s", v.Name, v.Value))
			continue
		}
		if v.ValueFrom != nil {
			switch {
			case v.ValueFrom.FieldRef != nil:
				out = append(out, fmt.Sprintf("%s=<field:%s>", v.Name, v.ValueFrom.FieldRef.FieldPath))
			case v.ValueFrom.ConfigMapKeyRef != nil:
				out = append(out, fmt.Sprintf("%s=<configmap:%s/%s>", v.Name, v.ValueFrom.ConfigMapKeyRef.Name, v.ValueFrom.ConfigMapKeyRef.Key))
			case v.ValueFrom.SecretKeyRef != nil:
				out = append(out, fmt.Sprintf("%s=<secret:%s/%s>", v.Name, v.ValueFrom.SecretKeyRef.Name, v.ValueFrom.SecretKeyRef.Key))
			case v.ValueFrom.ResourceFieldRef != nil:
				out = append(out, fmt.Sprintf("%s=<resource:%s>", v.Name, v.ValueFrom.ResourceFieldRef.Resource))
			default:
				out = append(out, fmt.Sprintf("%s=", v.Name))
			}
			continue
		}
		out = append(out, fmt.Sprintf("%s=", v.Name))
	}
	return out
}

func volumeMounts(mounts []corev1.VolumeMount) []string {
	if len(mounts) == 0 {
		return nil
	}
	out := make([]string, 0, len(mounts))
	for _, m := range mounts {
		if m.SubPath != "" {
			out = append(out, fmt.Sprintf("%s:%s (subPath=%s)", m.Name, m.MountPath, m.SubPath))
			continue
		}
		out = append(out, fmt.Sprintf("%s:%s", m.Name, m.MountPath))
	}
	return out
}

func volumeSummary(v corev1.Volume) string {
	switch {
	case v.ConfigMap != nil:
		return fmt.Sprintf("%s: ConfigMap(%s)", v.Name, v.ConfigMap.Name)
	case v.Secret != nil:
		return fmt.Sprintf("%s: Secret(%s)", v.Name, v.Secret.SecretName)
	case v.PersistentVolumeClaim != nil:
		return fmt.Sprintf("%s: PVC(%s)", v.Name, v.PersistentVolumeClaim.ClaimName)
	case v.EmptyDir != nil:
		return fmt.Sprintf("%s: EmptyDir", v.Name)
	case v.HostPath != nil:
		return fmt.Sprintf("%s: HostPath(%s)", v.Name, v.HostPath.Path)
	case v.Projected != nil:
		return fmt.Sprintf("%s: Projected", v.Name)
	default:
		return fmt.Sprintf("%s: %s", v.Name, "Other")
	}
}

func collectContainerRefs(c corev1.Container, cms, secrets map[string]struct{}) {
	for _, e := range c.Env {
		if e.ValueFrom == nil {
			continue
		}
		if e.ValueFrom.ConfigMapKeyRef != nil && e.ValueFrom.ConfigMapKeyRef.Name != "" {
			cms[e.ValueFrom.ConfigMapKeyRef.Name] = struct{}{}
		}
		if e.ValueFrom.SecretKeyRef != nil && e.ValueFrom.SecretKeyRef.Name != "" {
			secrets[e.ValueFrom.SecretKeyRef.Name] = struct{}{}
		}
	}
	for _, from := range c.EnvFrom {
		if from.ConfigMapRef != nil && from.ConfigMapRef.Name != "" {
			cms[from.ConfigMapRef.Name] = struct{}{}
		}
		if from.SecretRef != nil && from.SecretRef.Name != "" {
			secrets[from.SecretRef.Name] = struct{}{}
		}
	}
}

func collectVolumeRefs(v corev1.Volume, cms, secrets map[string]struct{}) {
	if v.ConfigMap != nil && v.ConfigMap.Name != "" {
		cms[v.ConfigMap.Name] = struct{}{}
	}
	if v.Secret != nil && v.Secret.SecretName != "" {
		secrets[v.Secret.SecretName] = struct{}{}
	}
	if v.Projected != nil {
		for _, src := range v.Projected.Sources {
			if src.ConfigMap != nil && src.ConfigMap.Name != "" {
				cms[src.ConfigMap.Name] = struct{}{}
			}
			if src.Secret != nil && src.Secret.Name != "" {
				secrets[src.Secret.Name] = struct{}{}
			}
		}
	}
}

func resolveLinkedServices(ctx context.Context, client kubernetes.Interface, namespace string, podLabels map[string]string) []string {
	res, err := client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	var out []string
	for _, svc := range res.Items {
		if len(svc.Spec.Selector) == 0 {
			continue
		}
		matched := true
		for k, v := range svc.Spec.Selector {
			if podLabels[k] != v {
				matched = false
				break
			}
		}
		if matched {
			out = append(out, svc.Name)
		}
	}
	sort.Strings(out)
	return out
}

func resolveLinkedWorkloads(ctx context.Context, client kubernetes.Interface, namespace string, owners []metav1.OwnerReference) []string {
	out := make([]string, 0, len(owners))
	for _, owner := range owners {
		name := owner.Name
		kind := owner.Kind
		if kind == "ReplicaSet" && name != "" {
			rs, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
			if err == nil {
				for _, parent := range rs.OwnerReferences {
					if parent.Kind == "Deployment" && parent.Name != "" {
						out = append(out, "Deployment/"+parent.Name)
						name = ""
						break
					}
				}
			}
		}
		if name != "" && kind != "" {
			out = append(out, kind+"/"+name)
		}
	}
	sort.Strings(out)
	return uniqStrings(out)
}

func resolveLinkedHPAs(ctx context.Context, client kubernetes.Interface, namespace string, workloads []string) []string {
	res, err := client.AutoscalingV2().HorizontalPodAutoscalers(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil
	}
	workloadSet := map[string]struct{}{}
	for _, w := range workloads {
		workloadSet[strings.ToLower(w)] = struct{}{}
	}
	var out []string
	for _, hpa := range res.Items {
		target := strings.ToLower(strings.TrimSpace(hpa.Spec.ScaleTargetRef.Kind + "/" + hpa.Spec.ScaleTargetRef.Name))
		if _, ok := workloadSet[target]; ok {
			out = append(out, hpa.Name)
		}
	}
	sort.Strings(out)
	return out
}

func resolveConfigMaps(ctx context.Context, client kubernetes.Interface, namespace string, names map[string]struct{}) []models.K8sConfigMapRef {
	keys := setKeys(names)
	out := make([]models.K8sConfigMapRef, 0, len(keys))
	for _, name := range keys {
		cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		out = append(out, models.K8sConfigMapRef{
			Name: cm.Name,
			Data: cm.Data,
		})
	}
	return out
}

func resolveSecrets(ctx context.Context, client kubernetes.Interface, namespace string, names map[string]struct{}) []models.K8sSecretRef {
	keys := setKeys(names)
	out := make([]models.K8sSecretRef, 0, len(keys))
	for _, name := range keys {
		sec, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			continue
		}
		sKeys := make([]string, 0, len(sec.Data))
		for k := range sec.Data {
			sKeys = append(sKeys, k)
		}
		sort.Strings(sKeys)
		out = append(out, models.K8sSecretRef{
			Name: sec.Name,
			Type: string(sec.Type),
			Keys: sKeys,
		})
	}
	return out
}

func setKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func uniqStrings(input []string) []string {
	if len(input) == 0 {
		return input
	}
	out := make([]string, 0, len(input))
	prev := ""
	for i, s := range input {
		if i == 0 || s != prev {
			out = append(out, s)
		}
		prev = s
	}
	return out
}
