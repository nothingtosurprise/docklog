<template>
  <div
    class="logs-container-layout"
    :class="{ 'sidebar-collapsed': isSidebarHidden }"
  >
    <!-- RESOURCES SIDEBAR -->
    <aside class="resources-sidebar glass">
      <div class="sidebar-top-nav">
        <!-- Mobile Header Row -->
        <div class="mobile-header-row show-mobile">
          <router-link to="/dashboard" class="back-nav-link-mobile">
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
            >
              <polyline points="15 18 9 12 15 6"></polyline>
            </svg>
            <span>Dashboard</span>
          </router-link>

          <button
            class="minimal-close-btn-glass"
            @click="isSidebarHidden = true"
          >
            <svg
              viewBox="0 0 24 24"
              width="20"
              height="20"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
            >
              <line x1="18" y1="6" x2="6" y2="18"></line>
              <line x1="6" y1="6" x2="18" y2="18"></line>
            </svg>
          </button>
        </div>
      </div>

      <div class="sidebar-header">
        <span class="label-caps">Log Resources</span>
        <div class="sidebar-header-row">
          <div v-if="showRuntimeToggle" class="runtime-toggle">
            <button
              class="runtime-btn"
              :class="{ active: activeRuntime === 'docker' }"
              @click="setRuntime('docker')"
            >
              Docker
            </button>
            <button
              class="runtime-btn"
              :class="{ active: activeRuntime === 'kubernetes' }"
              @click="setRuntime('kubernetes')"
            >
              Kubernetes
            </button>
          </div>
          <div class="sidebar-controls">
            <button
              class="mini-icon-btn"
              @click="isSidebarHidden = true"
              data-tooltip="Collapse Sidebar"
            >
              <svg
                viewBox="0 0 24 24"
                width="12"
                height="12"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <polyline points="15 18 9 12 15 6"></polyline>
              </svg>
            </button>

            <button
              class="mini-icon-btn hide-mobile"
              :class="{ 'active-toggle': splitView }"
              @click="toggleSplitView"
              :data-tooltip="
                splitView ? 'Disable Split View' : 'Enable Split View'
              "
            >
              <svg
                viewBox="0 0 24 24"
                width="12"
                height="12"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
              >
                <rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect>
                <line x1="12" y1="3" x2="12" y2="21"></line>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <div class="resource-list">
        <template v-if="dockerEnabled() && activeRuntime === 'docker'">
          <div v-if="kubernetesEnabled()" class="resource-section-label">Docker</div>
          <div
            v-for="c in filteredContainers"
            :key="c.id"
            class="resource-card group"
            :class="{ active: isContainerVisible(c.id) }"
            @click="toggleContainerStream(c.id)"
            @mouseenter="startLiveStats(c.id)"
            @mouseleave="stopLiveStats"
          >
            <div class="card-status-dot" :class="c.state"></div>
            <div class="card-info">
              <span class="card-name">{{ c.name }}</span>
              <span class="card-image-tag">{{ c.image }}</span>
            </div>

            <div v-if="c.state === 'running'" class="stats-peek-inline">
              <div class="peek-stat">
                <span
                  class="p-value"
                  :class="{ 'text-live': activeLiveId === c.id }"
                >
                  {{
                    (activeLiveId === c.id ? liveStats.cpu : c.cpu)?.toFixed(2) ||
                    "0.00"
                  }}%
                </span>
              </div>
              <div class="peek-stat">
                <span
                  class="p-value"
                  :class="{ 'text-live': activeLiveId === c.id }"
                >
                  {{
                    formatBytes(
                      activeLiveId === c.id ? liveStats.memory : c.memory,
                    )
                  }}
                </span>
              </div>
            </div>
          </div>
          <div v-if="filteredContainers.length === 0" class="empty-search-msg">
            <p class="text-mute">No containers found</p>
          </div>
        </template>

        <template v-if="kubernetesEnabled() && activeRuntime === 'kubernetes'">
          <div v-if="dockerEnabled()" class="resource-section-label">Kubernetes</div>
          <div
            v-for="pod in filteredPods"
            :key="podKey(pod)"
            class="resource-card group"
            :class="{ active: isPodVisible(pod) }"
            @click="togglePodStream(pod)"
          >
            <div class="card-status-dot" :class="(pod.phase || '').toLowerCase()"></div>
            <div class="card-info">
              <span class="card-name">{{ pod.name }}</span>
              <span class="card-image-tag">{{ pod.namespace }}</span>
            </div>
            <div class="stats-peek-inline">
              <span class="p-value">{{ pod.ready }}</span>
            </div>
          </div>
          <div v-if="filteredPods.length === 0" class="empty-search-msg">
            <p class="text-mute">No pods found</p>
          </div>
        </template>
      </div>
    </aside>

    <!-- Mobile Overlay -->
    <div
      v-if="!isSidebarHidden"
      class="mobile-sidebar-overlay show-mobile"
      @click="isSidebarHidden = true"
    ></div>

    <!-- FIXED TRIGGER (Always accessible to reopen sidebar) -->
    <button
      v-if="isSidebarHidden"
      class="fixed-open-trigger glass shadow-xl"
      @click="isSidebarHidden = false"
    >
      <svg
        viewBox="0 0 24 24"
        width="16"
        height="16"
        fill="none"
        stroke="currentColor"
        stroke-width="3"
      >
        <polyline points="9 18 15 12 9 6"></polyline>
      </svg>
    </button>

    <!-- MAIN VIEWPORT -->
    <main class="logs-main-content">
      <div
        v-if="displayStreams.length > 0"
        class="logs-grid"
        :class="gridClass"
      >
        <LogViewer
          v-for="stream in displayStreams"
          :key="stream.key"
          :container="stream.type === 'container' ? stream.data : undefined"
          :pod="stream.type === 'pod' ? stream.data : undefined"
          showClose
          @close="removeStream(stream)"
          @stats="handleViewerStats"
        />
      </div>

      <!-- PREMIUM HERO EMPTY STATE -->
      <div v-else class="empty-state-wrapper animate-fade-in">
        <div class="observability-hero">
          <div class="hero-visual">
            <div class="radar-scan">
              <div class="circle c1"></div>
              <div class="circle c2"></div>
              <div class="circle c3"></div>
              <svg
                viewBox="0 0 24 24"
                class="hero-icon"
                fill="none"
                stroke="currentColor"
                stroke-width="1.5"
              >
                <path
                  d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"
                ></path>
                <path d="M8 9h8M8 13h5" stroke-linecap="round"></path>
              </svg>
            </div>
          </div>
          <div class="hero-text">
            <h2 class="display-title">Ready for Insight?</h2>
            <p class="subtitle">
              Select a resource from the sidebar to launch a real-time log
              stream. Toggle split-view to monitor up to two resources
              simultaneously.
            </p>
            <div class="hero-actions" v-if="isSidebarHidden">
              <button
                @click="isSidebarHidden = false"
                class="premium-btn primary"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="18"
                  height="18"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="3"
                >
                  <path d="M9 18l6-6-6-6"></path>
                </svg>
                Open Resources
              </button>
            </div>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { sharedState, dockerEnabled, kubernetesEnabled } from "../utils/sharedState";
import { parsePodKey } from "../utils/logRoutes";
import { usePods } from "../composables/usePods";
import { secureStorage } from "../utils/storage";
import { apiFetch } from "../utils/apiFetch";
import LogViewer from "../components/LogViewer.vue";

const route = useRoute();
const router = useRouter();

const formatBytes = (bytes) => {
  if (!bytes || bytes === 0) return "0B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + sizes[i];
};

const containers = ref([]);
const { pods } = usePods({ autoPoll: false });

const podKey = (pod) => `${pod.namespace}/${pod.name}`;

// Live Stats on Hover Logic
const activeLiveId = ref(null);
const liveStats = ref({ cpu: 0, memory: 0 });
let liveInterval = null;

const handleViewerStats = (data) => {
  if (data.id === activeLiveId.value) {
    liveStats.value = { cpu: data.cpu, memory: data.memory };
  }
};

const startLiveStats = (id) => {
  activeLiveId.value = id;
  fetchStatsNow(id);
  if (liveInterval) clearInterval(liveInterval);
  liveInterval = setInterval(() => fetchStatsNow(id), 1000);
};

const stopLiveStats = () => {
  activeLiveId.value = null;
  if (liveInterval) clearInterval(liveInterval);
  liveInterval = null;
};

const fetchStatsNow = async (id) => {
  try {
    const token = secureStorage.getItem("token");
    const res = await apiFetch(`/api/containers/${id}/stats-now`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      const data = await res.json();
      liveStats.value = { cpu: data.cpu, memory: data.memory };
    }
  } catch (err) {
    console.error("Live stats fetch failed", err);
  }
};

const filteredContainers = computed(() => {
  if (!sharedState.searchQuery) return containers.value;
  const q = sharedState.searchQuery.toLowerCase();
  return containers.value.filter(
    (c) => c.name.toLowerCase().includes(q) || c.id.toLowerCase().includes(q),
  );
});
const filteredPods = computed(() => {
  if (!sharedState.searchQuery) return pods.value;
  const q = sharedState.searchQuery.toLowerCase();
  return pods.value.filter(
    (pod) =>
      pod.name.toLowerCase().includes(q) ||
      pod.namespace.toLowerCase().includes(q) ||
      (pod.images || []).some((image) => image.toLowerCase().includes(q)),
  );
});

const selectedContainerIds = ref([]);
const selectedPodKeys = ref([]);
const isSidebarHidden = ref(window.innerWidth < 1024);
const splitView = ref(route.query.split === "true");
const activeRuntime = ref(dockerEnabled() ? "docker" : "kubernetes");

const showRuntimeToggle = computed(
  () => dockerEnabled() && kubernetesEnabled(),
);

const syncStateFromUrl = () => {
  const containerParam = route.query.c;
  const podParam = route.query.p;

  selectedContainerIds.value = containerParam
    ? String(containerParam).split(",").filter(Boolean)
    : [];
  selectedPodKeys.value = podParam
    ? String(podParam).split(",").filter(Boolean)
    : [];

  splitView.value = route.query.split === "true";
  if (showRuntimeToggle.value) {
    const rt = String(route.query.rt || "").toLowerCase();
    if (rt === "kubernetes" || rt === "docker") {
      activeRuntime.value = rt;
    } else if (selectedPodKeys.value.length > 0) {
      activeRuntime.value = "kubernetes";
    } else if (selectedContainerIds.value.length > 0) {
      activeRuntime.value = "docker";
    } else {
      activeRuntime.value = dockerEnabled() ? "docker" : "kubernetes";
    }
  } else {
    activeRuntime.value = dockerEnabled() ? "docker" : "kubernetes";
  }
};

const displayContainers = computed(() => {
  if (containers.value.length === 0 || selectedContainerIds.value.length === 0) {
    return [];
  }

  const ordered = selectedContainerIds.value
    .map((id) => {
      let match = containers.value.find((c) => c.id === id);
      if (!match) {
        match = containers.value.find(
          (c) => c.id.startsWith(id) || id.startsWith(c.id),
        );
      }
      return match;
    })
    .filter(Boolean);

  return splitView.value ? ordered.slice(-2) : [ordered[ordered.length - 1]].filter(Boolean);
});

const displayPods = computed(() => {
  if (selectedPodKeys.value.length === 0) {
    return [];
  }

  const ordered = selectedPodKeys.value
    .map((key) => {
      const match = pods.value.find((pod) => podKey(pod) === key);
      if (match) return match;
      const parsed = parsePodKey(key);
      if (!parsed) return null;
      return { ...parsed, phase: "", ready: "", images: [] };
    })
    .filter(Boolean);

  return splitView.value ? ordered.slice(-2) : [ordered[ordered.length - 1]].filter(Boolean);
});

const displayStreams = computed(() => {
  const streams = [
    ...displayContainers.value.map((container) => ({
      type: "container",
      key: `container:${container.id}`,
      data: container,
    })),
    ...displayPods.value.map((pod) => ({
      type: "pod",
      key: `pod:${podKey(pod)}`,
      data: pod,
    })),
  ];
  return splitView.value ? streams.slice(-2) : streams.slice(-1);
});

watch(
  () => [containers.value, pods.value],
  () => {
    if (selectedContainerIds.value.length > 0 || selectedPodKeys.value.length > 0) {
      syncStateFromUrl();
    }
  },
  { immediate: true },
);

const isContainerVisible = (id) =>
  displayContainers.value.some((c) => c.id === id);
const isPodVisible = (pod) =>
  displayPods.value.some((item) => podKey(item) === podKey(pod));
const gridClass = computed(() =>
  displayStreams.value.length > 1 ? "grid-dual" : "grid-single",
);

const fetchContainers = async () => {
  try {
    const token = secureStorage.getItem("token");
    const res = await apiFetch("/api/containers", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      containers.value = await res.json();
    }
  } catch (err) {
    console.error(err);
  }
};

const fetchPodsForLogs = async () => {
  if (!kubernetesEnabled()) return;
  try {
    const token = secureStorage.getItem("token");
    const nsRes = await apiFetch("/api/namespaces", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (!nsRes.ok) return;
    const namespaces = await nsRes.json();
    const results = await Promise.all(
      namespaces.map(async ({ name }) => {
        const res = await apiFetch(
          `/api/pods?namespace=${encodeURIComponent(name)}`,
          { headers: { Authorization: `Bearer ${token}` } },
        );
        if (!res.ok) return [];
        return res.json();
      }),
    );
    pods.value = results.flat();
  } catch (err) {
    console.error(err);
  }
};

const fetchResources = async () => {
  const tasks = [];
  if (dockerEnabled()) tasks.push(fetchContainers());
  if (kubernetesEnabled()) tasks.push(fetchPodsForLogs());
  await Promise.all(tasks);
  syncStateFromUrl();
  ensureUrlReflectsSelection();
};

const ensureUrlReflectsSelection = () => {
  if (!showRuntimeToggle.value) return;

  const expectedRt =
    selectedPodKeys.value.length > 0
      ? "kubernetes"
      : selectedContainerIds.value.length > 0
        ? "docker"
        : activeRuntime.value;

  const expectedP =
    selectedPodKeys.value.length > 0
      ? selectedPodKeys.value.join(",")
      : "";
  const expectedC =
    selectedContainerIds.value.length > 0
      ? selectedContainerIds.value.join(",")
      : "";

  const currentRt = String(route.query.rt || "");
  const currentP = String(route.query.p || "");
  const currentC = String(route.query.c || "");

  if (
    currentRt !== expectedRt ||
    (expectedP && currentP !== expectedP) ||
    (expectedC && currentC !== expectedC) ||
    (!expectedP && currentP) ||
    (!expectedC && currentC)
  ) {
    updateUrl();
  }
};

const updateUrl = () => {
  const query = { ...route.query };

  if (selectedContainerIds.value.length > 0) {
    query.c = selectedContainerIds.value.join(",");
  } else {
    delete query.c;
  }

  if (selectedPodKeys.value.length > 0) {
    query.p = selectedPodKeys.value.join(",");
  } else {
    delete query.p;
  }

  if (splitView.value) {
    query.split = "true";
  } else {
    delete query.split;
  }

  if (showRuntimeToggle.value) {
    query.rt = activeRuntime.value;
  } else {
    delete query.rt;
  }

  router.replace({ query });
};

const setRuntime = (runtime) => {
  if (!showRuntimeToggle.value) return;
  if (runtime !== "docker" && runtime !== "kubernetes") return;
  if (activeRuntime.value === runtime) return;
  activeRuntime.value = runtime;
  selectedContainerIds.value = [];
  selectedPodKeys.value = [];
  updateUrl();
};

const toggleSplitView = () => {
  splitView.value = !splitView.value;
  updateUrl();
};

const toggleContainerStream = (id) => {
  if (splitView.value) {
    if (selectedContainerIds.value.includes(id)) {
      selectedContainerIds.value = selectedContainerIds.value.filter((sid) => sid !== id);
    } else {
      if (selectedContainerIds.value.length >= 2) {
        selectedContainerIds.value.shift();
      }
      selectedContainerIds.value.push(id);
    }
  } else {
    selectedPodKeys.value = [];
    selectedContainerIds.value = selectedContainerIds.value.includes(id) ? [] : [id];
  }
  if (window.innerWidth < 900) {
    isSidebarHidden.value = true;
  }
  updateUrl();
};

const togglePodStream = (pod) => {
  const key = podKey(pod);
  if (splitView.value) {
    if (selectedPodKeys.value.includes(key)) {
      selectedPodKeys.value = selectedPodKeys.value.filter((sid) => sid !== key);
    } else {
      if (selectedPodKeys.value.length >= 2) {
        selectedPodKeys.value.shift();
      }
      selectedPodKeys.value.push(key);
    }
  } else {
    selectedContainerIds.value = [];
    selectedPodKeys.value = selectedPodKeys.value.includes(key) ? [] : [key];
  }
  if (window.innerWidth < 900) {
    isSidebarHidden.value = true;
  }
  updateUrl();
};

const removeStream = (stream) => {
  if (stream.type === "container") {
    selectedContainerIds.value = selectedContainerIds.value.filter(
      (sid) => sid !== stream.data.id,
    );
  } else {
    selectedPodKeys.value = selectedPodKeys.value.filter(
      (sid) => sid !== podKey(stream.data),
    );
  }
  updateUrl();
};

let statusInterval = null;

onMounted(() => {
  fetchResources();
  statusInterval = setInterval(fetchResources, 5000);
});

onUnmounted(() => {
  if (statusInterval) clearInterval(statusInterval);
});

watch(() => route.query, syncStateFromUrl, { immediate: true });
</script>

<style scoped>
.logs-container-layout {
  display: flex;
  height: calc(100vh - 80px);
  position: relative;
  overflow: hidden;
  background: var(--bg-main);
}

.resource-section-label {
  margin: 0.75rem 0 0.35rem;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-mute);
}

.card-status-dot.pending {
  background: var(--warning);
}

.card-status-dot.failed,
.card-status-dot.unknown {
  background: var(--error);
}

/* SIDEBAR UI FIXES */
.resources-sidebar {
  width: 300px;
  height: 100%;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1.25rem;
  transition: all 0.4s cubic-bezier(0.4, 0, 0.2, 1);
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
}

.sidebar-top-nav {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-decoration: none;
  width: 280px;
}

.logo-img-sidebar {
  width: 36px;
  height: 36px;
  object-fit: contain;
  border-radius: var(--radius-sm);
  background: transparent;
  border: none;
}

.logo-text {
  font-size: 1.2rem;
  font-weight: 950;
  color: var(--text-main);
  letter-spacing: -0.03em;
}

.back-nav-link,
.back-nav-link-mobile {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-dim);
  font-size: 0.75rem;
  font-weight: 700;
  text-decoration: none;
  transition: all 0.2s;
}

.back-nav-link:hover,
.back-nav-link-mobile:hover {
  color: var(--accent);
  border-color: rgba(var(--accent-rgb), 0.35);
  background: var(--accent-soft);
  transform: none;
}

.sidebar-header {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 0.75rem;
  margin-bottom: 1.25rem;
}

.label-caps {
  text-transform: uppercase;
  font-size: 0.75rem;
  font-weight: 900;
  color: var(--text-mute);
  letter-spacing: 0.05em;
}

.resource-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  overflow-y: auto;
  padding: 0 0.5rem;
}

.resource-card {
  padding: 0.85rem 1rem;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  gap: 0.85rem;
  border: 1px solid var(--border);
  background: var(--bg-card);
  cursor: pointer;
  transition: border-color 0.2s, background 0.2s, transform 0.2s;
  position: relative;
  overflow: hidden;
}

.resource-card:hover {
  border-color: var(--accent);
  transform: translateX(4px);
  background: var(--card-hover);
}

/* Stats Peek Styling */
.stats-peek-inline {
  position: absolute;
  right: 0.75rem;
  top: 50%;
  transform: translateY(-50%) translateX(10px);
  display: flex;
  gap: 0.75rem;
  background: var(--bg-glass);
  backdrop-filter: blur(8px);
  padding: 0.4rem 0.6rem;
  border-radius: 8px;
  border: 1px solid var(--border-light);
  opacity: 0;
  pointer-events: none;
  transition: all 0.2s ease;
  z-index: 10;
}

.resource-card:hover .stats-peek-inline {
  opacity: 1;
  transform: translateY(-50%) translateX(0);
}

.peek-stat {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.p-label {
  font-size: 0.55rem;
  font-weight: 900;
  color: var(--text-mute);
  text-transform: uppercase;
}

.p-value {
  font-size: 0.7rem;
  font-weight: 800;
  color: var(--accent);
  font-family: var(--font-mono);
  transition: color 0.3s;
}

.text-live {
  color: var(--success) !important;
  text-shadow: 0 0 8px rgba(var(--success-rgb), 0.4);
}

.resource-card.active {
  border-color: var(--accent);
  background: rgba(var(--accent-rgb), 0.05);
}

.card-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #4b5563;
  flex-shrink: 0;
}

.card-status-dot.running {
  background: var(--success);
  box-shadow: 0 0 10px var(--success);
}

.card-info {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
  flex: 1;
}

.card-name {
  font-weight: 800;
  font-size: 0.9rem;
  color: var(--text-main);
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  display: block;
  max-width: 100%;
}

.card-image-tag {
  font-size: 0.7rem;
  color: var(--text-mute);
  font-family: monospace;
  white-space: nowrap !important;
  overflow: hidden !important;
  text-overflow: ellipsis !important;
  max-width: 100%;
  display: block;
}

.sidebar-header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.6rem;
  margin-top: 0.5rem;
}

.mini-icon-btn {
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-mute);
  width: 28px;
  height: 28px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}

.active-toggle {
  background: var(--accent) !important;
  color: white !important;
  border-color: var(--accent) !important;
  box-shadow: 0 4px 14px rgba(var(--accent-rgb), 0.35);
}

.sidebar-controls {
  display: flex;
  gap: 0.5rem;
}

.sidebar-collapsed .resources-sidebar {
  width: 0;
  margin-right: -1.5rem;
  opacity: 0;
  pointer-events: none;
}

.fixed-open-trigger {
  position: fixed;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 60px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-left: none;
  border-radius: 0 12px 12px 0;
  z-index: 500;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* GRID */
.logs-main-content {
  flex: 1;
  min-width: 0;
  min-height: 0;
  padding: 1.5rem;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.logs-grid {
  display: grid;
  gap: 1.5rem;
  flex: 1;
  min-height: 0;
}
.logs-grid > :deep(.log-viewer) {
  min-height: 0;
  height: 100%;
}
.grid-single {
  grid-template-columns: 1fr;
}
.grid-dual {
  grid-template-columns: 1fr 1fr;
}

/* HERO EMPTY STATE */
.empty-state-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
}

.observability-hero {
  text-align: center;
  max-width: 460px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2.5rem;
}

.hero-visual {
  position: relative;
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.radar-scan {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.hero-icon {
  width: 56px;
  height: 56px;
  color: var(--accent);
  z-index: 2;
  filter: drop-shadow(0 0 15px rgba(var(--accent-rgb), 0.4));
}

.circle {
  position: absolute;
  border: 1px solid var(--accent);
  border-radius: 50%;
  opacity: 0;
  animation: radar 4s infinite linear;
}

.c1 {
  width: 60px;
  height: 60px;
  animation-delay: 0s;
}
.c2 {
  width: 60px;
  height: 60px;
  animation-delay: 1.3s;
}
.c3 {
  width: 60px;
  height: 60px;
  animation-delay: 2.6s;
}

@keyframes radar {
  0% {
    transform: scale(1);
    opacity: 0.6;
  }
  100% {
    transform: scale(3);
    opacity: 0;
  }
}

.display-title {
  font-size: 2.2rem;
  font-weight: 950;
  color: var(--text-main);
  letter-spacing: -0.04em;
  margin-bottom: 1rem;
}

.subtitle {
  font-size: 1rem;
  line-height: 1.6;
  color: var(--text-mute);
  font-weight: 500;
  margin-bottom: 2rem;
}

.runtime-toggle {
  display: inline-flex;
  gap: 0.25rem;
  padding: 0.2rem;
  border-radius: 8px;
  border: 1px solid var(--border);
  background: var(--bg-input);
}

.runtime-btn {
  border: none;
  background: transparent;
  color: var(--text-mute);
  font-size: 0.68rem;
  font-weight: 800;
  padding: 0.3rem 0.5rem;
  border-radius: 6px;
  cursor: pointer;
}

.runtime-btn.active {
  background: var(--accent);
  color: #fff;
}

@media (max-width: 1024px) {
  .logs-container-layout {
    flex-direction: column;
  }

  .resources-sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: 280px;
    z-index: 10000 !important;
    box-shadow: 20px 0 60px rgba(0, 0, 0, 0.8);
    transform: translateX(0);
    opacity: 1;
    padding: 1.25rem 1rem;
    backdrop-filter: none !important;
    transition: all 0.4s cubic-bezier(0.23, 1, 0.32, 1);
    visibility: visible;
  }

  .sidebar-top-nav {
    margin-bottom: 1.5rem !important;
    display: flex !important;
    flex-direction: column !important;
    gap: 0.75rem !important;
  }

  .mobile-header-row {
    display: flex !important;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--border);
  }

  .sidebar-logo-mobile {
    display: flex;
    align-items: center;
    gap: 0.85rem;
    text-decoration: none;
  }

  .logo-icon-premium {
    width: 36px;
    height: 36px;
    background: var(--accent);
    color: #fff;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: 0 0 20px rgba(var(--accent-rgb), 0.35);
  }

  .logo-text-premium {
    font-size: 1.25rem;
    font-weight: 900;
    letter-spacing: -0.03em;
    color: #fff;
  }

  .minimal-close-btn-glass {
    width: 40px;
    height: 40px;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 10px;
    color: var(--text-mute);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
  }

  .minimal-close-btn-glass:hover {
    background: rgba(239, 68, 68, 0.15);
    color: var(--error);
    border-color: rgba(239, 68, 68, 0.3);
    transform: rotate(90deg);
  }

  .minimal-close-btn-glass:active {
    transform: scale(0.9);
  }

  .back-nav-link-mobile {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--text-main);
    text-decoration: none;
    font-weight: 850;
    font-size: 0.9rem;
    padding: 0.5rem 0.75rem;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 10px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    transition: all 0.2s;
  }

  .back-nav-link-mobile:hover {
    background: var(--accent-soft);
    color: var(--accent);
    border-color: rgba(var(--accent-rgb), 0.3);
  }

  .sidebar-header {
    margin-bottom: 1rem !important;
  }

  .sidebar-collapsed .resources-sidebar {
    transform: translateX(-100%);
    opacity: 0;
    visibility: hidden;
    pointer-events: none;
  }

  .logs-main-content {
    padding: 1rem;
    height: 100%;
    min-height: 0;
    z-index: 1;
    position: relative;
  }

  .grid-dual {
    grid-template-columns: 1fr;
    grid-template-rows: 1fr 1fr;
  }

  .display-title {
    font-size: 1.75rem;
  }

  .hide-mobile {
    display: none !important;
  }

  .show-mobile {
    display: flex !important;
  }

  .mobile-close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    padding: 1rem;
    background: rgba(239, 68, 68, 0.08);
    color: #ef4444;
    border: 1px solid rgba(239, 68, 68, 0.15);
    border-radius: 18px;
    font-size: 0.9rem;
    font-weight: 900;
    cursor: pointer;
    margin-top: 0.5rem;
    width: 100%;
    transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
  }

  .mobile-close-btn:hover {
    background: #ef4444;
    color: #fff;
    box-shadow: 0 10px 25px rgba(239, 68, 68, 0.3);
    transform: translateY(-2px);
  }

  .mobile-sidebar-overlay {
    position: fixed;
    inset: 0;
    background: rgba(2, 6, 23, 0.8);
    z-index: 4000;
    animation: fade-in 0.4s ease;
  }
}

.hero-actions {
  justify-items: center;
}

@keyframes fade-in {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

.show-mobile {
  display: none;
}

@media (max-width: 600px) {
  .resources-sidebar {
    width: 100%;
    max-width: none;
  }
}
</style>
