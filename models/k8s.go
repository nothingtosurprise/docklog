package models

type K8sNamespace struct {
	Name string `json:"name"`
}

type K8sPod struct {
	Namespace string   `json:"namespace"`
	Name      string   `json:"name"`
	UID       string   `json:"uid"`
	Phase     string   `json:"phase"`
	Node      string   `json:"node"`
	IP        string   `json:"ip"`
	Ready     string   `json:"ready"`
	Restarts  int32    `json:"restarts"`
	Created   int64    `json:"created"`
	Images    []string `json:"images"`
	Status    string   `json:"status"`
}

type K8sPodContainer struct {
	Name          string   `json:"name"`
	Image         string   `json:"image"`
	Ready         bool     `json:"ready"`
	RestartCount  int32    `json:"restart_count"`
	State         string   `json:"state"`
	CPURequest    string   `json:"cpu_request"`
	CPULimit      string   `json:"cpu_limit"`
	MemoryRequest string   `json:"memory_request"`
	MemoryLimit   string   `json:"memory_limit"`
	Ports         string   `json:"ports"`
	Env           []string `json:"env"`
	VolumeMounts  []string `json:"volume_mounts"`
}

type K8sPodLiveStats struct {
	CPU              float64 `json:"cpu"`
	Memory           int64   `json:"memory"`
	MemoryLimit      int64   `json:"memory_limit"`
	CPULimitMilli    int64   `json:"cpu_limit_millicores"`
	MetricsAvailable bool    `json:"metrics_available"`
}

type K8sPodDetail struct {
	Namespace         string            `json:"namespace"`
	Name              string            `json:"name"`
	UID               string            `json:"uid"`
	Phase             string            `json:"phase"`
	Node              string            `json:"node"`
	IP                string            `json:"ip"`
	ServiceAccount    string            `json:"service_account"`
	HostIP            string            `json:"host_ip"`
	QoSClass          string            `json:"qos_class"`
	Ready             string            `json:"ready"`
	Restarts          int32             `json:"restarts"`
	Created           int64             `json:"created"`
	Labels            map[string]string `json:"labels"`
	Annotations       map[string]string `json:"annotations"`
	Status            string            `json:"status"`
	Conditions        []string          `json:"conditions"`
	Volumes           []string          `json:"volumes"`
	OwnerReferences   []string          `json:"owner_references"`
	LinkedServices    []string          `json:"linked_services"`
	LinkedHPAs        []string          `json:"linked_hpas"`
	LinkedWorkloads   []string          `json:"linked_workloads"`
	ConfigMaps        []K8sConfigMapRef `json:"configmaps"`
	Secrets           []K8sSecretRef    `json:"secrets"`
	Containers        []K8sPodContainer `json:"containers"`
	InitContainers    []K8sPodContainer `json:"init_containers"`
	ContainerStatuses []string          `json:"container_statuses"`
}

type K8sConfigMapRef struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

type K8sSecretRef struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Keys []string `json:"keys"`
}

type K8sDeployment struct {
	Namespace string   `json:"namespace"`
	Name      string   `json:"name"`
	UID       string   `json:"uid"`
	Replicas  int32    `json:"replicas"`
	Ready     int32    `json:"ready"`
	Available int32    `json:"available"`
	Updated   int32    `json:"updated"`
	Strategy  string   `json:"strategy"`
	Images    []string `json:"images"`
	Status    string   `json:"status"`
	Created   int64    `json:"created"`
}

type K8sHPA struct {
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	UID             string `json:"uid"`
	TargetKind      string `json:"target_kind"`
	TargetName      string `json:"target_name"`
	MinReplicas     int32  `json:"min_replicas"`
	MaxReplicas     int32  `json:"max_replicas"`
	CurrentReplicas int32  `json:"current_replicas"`
	DesiredReplicas int32  `json:"desired_replicas"`
	Metrics         string `json:"metrics"`
	Status          string `json:"status"`
	Created         int64  `json:"created"`
}

type K8sService struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	UID       string `json:"uid"`
	Type      string `json:"type"`
	ClusterIP string `json:"cluster_ip"`
	Ports     string `json:"ports"`
	Selector  string `json:"selector"`
	Created   int64  `json:"created"`
}

type K8sEvent struct {
	Namespace        string `json:"namespace"`
	Name             string `json:"name"`
	UID              string `json:"uid"`
	ResourceVersion  string `json:"resource_version"`
	Type             string `json:"type"`
	Reason           string `json:"reason"`
	Message          string `json:"message"`
	InvolvedKind     string `json:"involved_kind"`
	InvolvedName     string `json:"involved_name"`
	Count            int32  `json:"count"`
	LastTimestamp    int64  `json:"last_timestamp"`
}

type K8sOverview struct {
	Namespace    string `json:"namespace"`
	Pods         int    `json:"pods"`
	RunningPods  int    `json:"running_pods"`
	Deployments  int    `json:"deployments"`
	HPAs         int    `json:"hpas"`
	Services     int    `json:"services"`
	WarningEvents int   `json:"warning_events"`
}
