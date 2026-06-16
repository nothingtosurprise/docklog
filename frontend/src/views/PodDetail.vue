<template>
  <div class="detail-view animate-fade-in">
    <div v-if="loading && !pod" class="detail-loading">
      <div class="shimmer-block"></div>
      <div class="shimmer-block wide"></div>
    </div>

    <template v-else-if="pod">
      <section class="detail-hero">
        <div class="hero-top">
          <router-link :to="backToKubernetes" class="back-link">
            <AppIcon name="chevronLeft" :size="16" :stroke-width="3" />
            <span>Kubernetes</span>
          </router-link>
          <div class="state-pill" :class="phaseClass(pod.phase)">
            <span v-if="isRunning" class="pulse-dot"></span>
            {{ pod.phase || 'Unknown' }}
          </div>
        </div>

        <div class="hero-main">
          <div class="hero-icon" :class="heroIconClass">
            <AppIcon name="containers" :size="28" />
          </div>
          <div class="hero-copy">
            <h1>{{ pod.name }}</h1>
            <div class="hero-meta">
              <button
                v-if="pod.uid"
                type="button"
                class="id-chip"
                @click="copyText(pod.uid, 'Pod UID')"
              >
                <span>{{ shortUid }}</span>
                <AppIcon name="copy" :size="14" />
              </button>
              <span v-if="pod.uid" class="meta-dot">·</span>
              <span class="meta-image">{{ pod.namespace }}</span>
              <span v-if="primaryImage" class="meta-dot">·</span>
              <span v-if="primaryImage" class="meta-image" :title="primaryImage">{{ primaryImage }}</span>
            </div>
            <p class="hero-status">{{ pod.status || 'No status available' }}</p>
          </div>
        </div>

        <div class="action-rail">
          <button type="button" class="action-chip logs" @click="goToLogs">
            <AppIcon name="logsBubble" :size="16" />
            <span>Logs</span>
          </button>
          <button
            v-if="userCanShell(sharedState.currentUser) && isRunning"
            type="button"
            class="action-chip shell"
            @click="goToShell"
          >
            <AppIcon name="terminal" :size="16" />
            <span>Shell</span>
          </button>
          <button
            v-if="userCanStart(sharedState.currentUser) && !isRunning"
            type="button"
            class="action-chip start"
            @click="triggerConfirm('start')"
          >
            <AppIcon name="play" :size="16" :stroke-width="3" />
            <span>Start</span>
          </button>
          <button
            v-if="userCanStop(sharedState.currentUser) && isRunning"
            type="button"
            class="action-chip stop"
            @click="triggerConfirm('stop')"
          >
            <AppIcon name="stopOutline" :size="16" :stroke-width="3" />
            <span>Stop</span>
          </button>
          <button
            v-if="userCanRestart(sharedState.currentUser)"
            type="button"
            class="action-chip restart"
            @click="triggerConfirm('restart')"
          >
            <AppIcon name="refresh" :size="16" :stroke-width="3" />
            <span>Restart</span>
          </button>
          <button
            v-if="userCanDelete(sharedState.currentUser)"
            type="button"
            class="action-chip delete"
            @click="triggerConfirm('remove')"
          >
            <AppIcon name="trash" :size="16" />
            <span>Delete</span>
          </button>
        </div>
      </section>

      <section class="stats-grid">
        <template v-if="isRunning">
          <article class="stat-card">
            <span class="stat-label">CPU</span>
            <span class="stat-value">{{ cpuDisplay }}</span>
            <div v-if="liveStats.metrics_available" class="stat-bar">
              <div
                class="stat-bar-fill accent"
                :style="{ width: `${Math.min(liveStats.cpu, 100)}%` }"
              ></div>
            </div>
            <span v-else class="stat-hint">metrics-server required</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">Memory</span>
            <span class="stat-value">{{ memoryDisplay }}</span>
            <span v-if="memLimit" class="stat-sub">of {{ formatBytes(memLimit) }}</span>
            <span v-else-if="!liveStats.metrics_available" class="stat-hint">metrics-server required</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">CPU limit</span>
            <span class="stat-value sm">{{ cpuLimitLabel }}</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">Created</span>
            <span class="stat-value sm">{{ formatDate(pod.created) }}</span>
          </article>
        </template>
        <template v-else>
          <article class="stat-card">
            <span class="stat-label">Ready</span>
            <span class="stat-value">{{ pod.ready || '—' }}</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">Restarts</span>
            <span class="stat-value">{{ pod.restarts ?? 0 }}</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">Node</span>
            <span class="stat-value sm">{{ pod.node || '—' }}</span>
          </article>
          <article class="stat-card">
            <span class="stat-label">Created</span>
            <span class="stat-value sm">{{ formatDate(pod.created) }}</span>
          </article>
        </template>
      </section>

      <section class="detail-panels">
        <article class="panel">
          <div class="panel-head">
            <h2>Overview</h2>
          </div>
          <div class="kv-grid">
            <div class="kv-item">
              <span class="kv-label">Namespace</span>
              <span class="kv-value">{{ pod.namespace }}</span>
            </div>
            <div class="kv-item">
              <span class="kv-label">Node</span>
              <span class="kv-value">{{ pod.node || '—' }}</span>
            </div>
            <div class="kv-item">
              <span class="kv-label">Pod IP</span>
              <span class="kv-value mono">{{ pod.ip || '—' }}</span>
            </div>
            <div class="kv-item">
              <span class="kv-label">Host IP</span>
              <span class="kv-value mono">{{ pod.host_ip || '—' }}</span>
            </div>
            <div class="kv-item">
              <span class="kv-label">Service account</span>
              <span class="kv-value">{{ pod.service_account || 'default' }}</span>
            </div>
            <div class="kv-item">
              <span class="kv-label">QoS class</span>
              <span class="kv-value">{{ pod.qos_class || '—' }}</span>
            </div>
          </div>
        </article>

        <article class="panel">
          <div class="panel-head">
            <h2>Network ports</h2>
          </div>
          <div v-if="allPorts.length" class="port-list">
            <div v-for="port in allPorts" :key="port" class="port-row mono">{{ port }}</div>
          </div>
          <p v-else class="panel-empty">No container ports exposed.</p>
        </article>

        <article class="panel panel-wide">
          <div class="panel-head">
            <h2>Containers</h2>
            <span v-if="pod.init_containers?.length" class="panel-count">
              {{ pod.containers?.length || 0 }} app · {{ pod.init_containers.length }} init
            </span>
          </div>
          <div v-if="pod.containers?.length || pod.init_containers?.length" class="container-cards">
            <div
              v-for="container in pod.containers"
              :key="container.name"
              class="container-card"
            >
              <div class="container-card-head">
                <div class="container-card-title">
                  <AppIcon name="box" :size="15" />
                  <strong>{{ container.name }}</strong>
                </div>
                <span class="state-pill" :class="containerStateClass(container.state)">
                  {{ container.state }}
                </span>
              </div>
              <p class="container-card-image mono">{{ container.image }}</p>
              <div class="container-card-grid">
                <div>
                  <span class="mini-label">CPU</span>
                  <span class="mini-value mono">{{ container.cpu_request }} / {{ container.cpu_limit }}</span>
                </div>
                <div>
                  <span class="mini-label">Memory</span>
                  <span class="mini-value mono">{{ container.memory_request }} / {{ container.memory_limit }}</span>
                </div>
                <div>
                  <span class="mini-label">Restarts</span>
                  <span class="mini-value">{{ container.restart_count ?? 0 }}</span>
                </div>
                <div v-if="container.ports && container.ports !== '-'">
                  <span class="mini-label">Ports</span>
                  <span class="mini-value mono">{{ container.ports }}</span>
                </div>
              </div>
            </div>
            <div
              v-for="container in pod.init_containers"
              :key="`init-${container.name}`"
              class="container-card init"
            >
              <div class="container-card-head">
                <div class="container-card-title">
                  <AppIcon name="refresh" :size="14" />
                  <strong>{{ container.name }}</strong>
                  <span class="init-badge">Init</span>
                </div>
                <span class="state-pill" :class="containerStateClass(container.state)">
                  {{ container.state }}
                </span>
              </div>
              <p class="container-card-image mono">{{ container.image }}</p>
            </div>
          </div>
          <p v-else class="panel-empty">No containers found.</p>
        </article>

        <article class="panel panel-wide">
          <div class="panel-head">
            <h2>Environment</h2>
            <div class="env-search">
              <AppIcon name="search" :size="14" />
              <input v-model="envQuery" type="text" placeholder="Filter variables..." />
            </div>
          </div>
          <div v-if="filteredEnv.length" class="env-list">
            <div v-for="item in filteredEnv" :key="item.raw" class="env-row">
              <span class="env-key mono">{{ item.container }} · {{ item.key }}</span>
              <span class="env-val mono">{{ item.value }}</span>
            </div>
          </div>
          <p v-else class="panel-empty">No environment variables matched.</p>
        </article>

        <article class="panel panel-wide detail-tabs-panel">
          <div class="detail-tabs">
            <button
              v-for="tab in detailTabs"
              :key="tab.id"
              type="button"
              class="detail-tab"
              :class="{ active: activeDetailTab === tab.id }"
              @click="activeDetailTab = tab.id"
            >
              {{ tab.label }}
              <span v-if="tab.count != null" class="tab-count">{{ tab.count }}</span>
            </button>
          </div>

          <div v-if="activeDetailTab === 'storage'" class="tab-panel">
            <div class="tab-split">
              <div>
                <h3>Volume mounts</h3>
                <div v-if="allMounts.length" class="key-list flush">
                  <div v-for="line in allMounts" :key="line" class="key-row mono">{{ line }}</div>
                </div>
                <p v-else class="panel-empty">No volume mounts.</p>
              </div>
              <div>
                <h3>Volumes</h3>
                <div v-if="pod.volumes?.length" class="key-list flush">
                  <div v-for="line in pod.volumes" :key="line" class="key-row mono">{{ line }}</div>
                </div>
                <p v-else class="panel-empty">No volumes.</p>
              </div>
            </div>
          </div>

          <div v-else-if="activeDetailTab === 'metadata'" class="tab-panel">
            <div class="tab-split">
              <div>
                <h3>Labels</h3>
                <div v-if="labelEntries.length" class="tag-list">
                  <span v-for="[k, v] in labelEntries" :key="k" class="tag-chip mono">{{ k }}={{ v }}</span>
                </div>
                <p v-else class="panel-empty">No labels.</p>
              </div>
              <div>
                <h3>Annotations</h3>
                <div v-if="annotationEntries.length" class="key-list flush">
                  <div v-for="[k, v] in annotationEntries" :key="k" class="key-row">
                    <span class="mono tag-key">{{ k }}</span>
                    <span class="mono tag-val">{{ v }}</span>
                  </div>
                </div>
                <p v-else class="panel-empty">No annotations.</p>
              </div>
            </div>
            <div class="tab-block">
              <h3>Conditions</h3>
              <div v-if="pod.conditions?.length" class="key-list flush">
                <div v-for="line in pod.conditions" :key="line" class="key-row mono">{{ line }}</div>
              </div>
              <p v-else class="panel-empty">No conditions available.</p>
            </div>
            <div class="tab-block">
              <h3>Owner references</h3>
              <div v-if="pod.owner_references?.length" class="tag-list">
                <span v-for="line in pod.owner_references" :key="line" class="tag-chip mono">{{ line }}</span>
              </div>
              <p v-else class="panel-empty">No owner references.</p>
            </div>
          </div>

          <div v-else class="tab-panel">
            <div class="tab-block">
              <h3>Linked workloads & services</h3>
              <div v-if="hasLinkedResources" class="linked-grid">
                <div v-if="pod.linked_workloads?.length" class="linked-card">
                  <span class="linked-label">Workloads</span>
                  <span class="linked-value mono">{{ pod.linked_workloads.join(', ') }}</span>
                </div>
                <div v-if="pod.linked_hpas?.length" class="linked-card">
                  <span class="linked-label">HPAs</span>
                  <span class="linked-value mono">{{ pod.linked_hpas.join(', ') }}</span>
                </div>
                <div v-if="pod.linked_services?.length" class="linked-card">
                  <span class="linked-label">Services</span>
                  <span class="linked-value mono">{{ pod.linked_services.join(', ') }}</span>
                </div>
              </div>
              <p v-else class="panel-empty">No linked workloads or services.</p>
            </div>

            <div class="tab-split">
              <div>
                <h3>ConfigMaps</h3>
                <div v-if="pod.configmaps?.length" class="resource-list">
                  <div v-for="cm in pod.configmaps" :key="cm.name" class="resource-card">
                    <div class="resource-card-head">
                      <strong class="mono">{{ cm.name }}</strong>
                      <span class="sub-text">{{ Object.keys(cm.data || {}).length }} keys</span>
                    </div>
                    <div v-if="Object.keys(cm.data || {}).length" class="kv-inline">
                      <div v-for="(v, k) in cm.data" :key="`${cm.name}-${k}`" class="kv-inline-row">
                        <span class="mono kv-key">{{ k }}</span>
                        <span class="mono kv-val">{{ v }}</span>
                      </div>
                    </div>
                  </div>
                </div>
                <p v-else class="panel-empty">No linked ConfigMaps.</p>
              </div>
              <div>
                <h3>Secrets</h3>
                <div v-if="pod.secrets?.length" class="resource-list">
                  <div v-for="sec in pod.secrets" :key="sec.name" class="resource-card">
                    <strong class="mono">{{ sec.name }}</strong>
                    <span class="sub-text">{{ sec.type || 'Opaque' }}</span>
                    <span class="sub-text">Keys: {{ (sec.keys || []).join(', ') || '—' }}</span>
                  </div>
                </div>
                <p v-else class="panel-empty">No linked Secrets.</p>
              </div>
            </div>
          </div>
        </article>
      </section>
    </template>

    <div v-else class="detail-empty">
      <h2>Pod not found</h2>
      <p>This pod may have been deleted or you may not have access.</p>
      <router-link :to="backToKubernetes" class="back-btn">Back to Kubernetes</router-link>
    </div>

    <Teleport to="body">
      <Transition name="fade">
        <div
          v-if="showConfirm"
          class="modal-overlay"
          @click.self="closeConfirm"
        >
          <div
            class="modal-content shadow-2xl"
            :class="{ 'modal-content-working': actionPending }"
            :aria-busy="actionPending"
          >
            <div :class="['modal-icon', actionClass]">
              <AppIcon
                v-if="pendingAction === 'start'"
                name="play"
                :size="28"
                :stroke-width="2.5"
              />
              <AppIcon
                v-else-if="pendingAction === 'restart'"
                name="refresh"
                :size="28"
                :stroke-width="2.5"
              />
              <AppIcon v-else name="alert" :size="28" />
            </div>
            <div class="modal-text-center">
              <h3>Confirm operation</h3>
              <p>
                Are you sure you want to <strong>{{ pendingAction }}</strong>
                <strong>{{ podName }}</strong>?
              </p>
            </div>
            <div class="modal-actions">
              <button
                type="button"
                class="modal-btn cancel"
                :disabled="actionPending"
                @click="closeConfirm"
              >
                Cancel
              </button>
              <button
                type="button"
                :class="['modal-btn confirm', actionClass, { 'is-working': actionPending }]"
                :disabled="actionPending"
                @click="confirmAction"
              >
                <span v-if="actionPending" class="btn-spinner" aria-hidden="true"></span>
                {{ actionPending ? `Working on ${pendingAction}...` : `Confirm ${pendingAction}` }}
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { apiFetch } from '../utils/apiFetch';
import { secureStorage } from '../utils/storage';
import { logsRouteForPod } from '../utils/logRoutes';
import AppIcon from '../components/AppIcon.vue';
import {
  sharedState,
  showToast,
  formatBytes,
  userCanStart,
  userCanStop,
  userCanRestart,
  userCanDelete,
  userCanShell,
} from '../utils/sharedState';

const route = useRoute();
const router = useRouter();

const loading = ref(true);
const pod = ref(null);
const envQuery = ref('');
const actionPending = ref(false);
const showConfirm = ref(false);
const pendingAction = ref('');
const activeDetailTab = ref('metadata');
const liveStats = ref({
  cpu: 0,
  memory: 0,
  memory_limit: 0,
  cpu_limit_millicores: 0,
  metrics_available: false,
});
let statsTimer = null;

const namespace = computed(() => String(route.params.namespace || ''));
const podName = computed(() => String(route.params.pod || ''));

const backToKubernetes = computed(() => ({
  path: '/kubernetes',
  query: { tab: 'pods', namespace: namespace.value },
}));

const labelEntries = computed(() => Object.entries(pod.value?.labels || {}));
const annotationEntries = computed(() => Object.entries(pod.value?.annotations || {}));
const allEnv = computed(() => {
  const lines = [];
  const containers = pod.value?.containers || [];
  const initContainers = pod.value?.init_containers || [];
  for (const c of [...containers, ...initContainers]) {
    for (const env of c.env || []) {
      const idx = env.indexOf('=');
      const key = idx === -1 ? env : env.slice(0, idx);
      const value = idx === -1 ? '' : env.slice(idx + 1);
      lines.push({
        container: c.name,
        key,
        value,
        raw: `${c.name}:${env}`,
      });
    }
  }
  return lines;
});
const filteredEnv = computed(() => {
  const q = envQuery.value.trim().toLowerCase();
  if (!q) return allEnv.value;
  return allEnv.value.filter((line) =>
    `${line.container} ${line.key} ${line.value}`.toLowerCase().includes(q),
  );
});
const allMounts = computed(() => {
  const lines = [];
  const containers = pod.value?.containers || [];
  const initContainers = pod.value?.init_containers || [];
  for (const c of [...containers, ...initContainers]) {
    for (const mount of c.volume_mounts || []) {
      lines.push(`${c.name}: ${mount}`);
    }
  }
  return lines;
});

const allPorts = computed(() => {
  const ports = new Set();
  for (const c of pod.value?.containers || []) {
    if (c.ports && c.ports !== '-') {
      c.ports.split(',').forEach((p) => {
        const trimmed = p.trim();
        if (trimmed) ports.add(trimmed);
      });
    }
  }
  return [...ports].sort();
});

const hasLinkedResources = computed(
  () =>
    (pod.value?.linked_workloads?.length || 0) > 0
    || (pod.value?.linked_hpas?.length || 0) > 0
    || (pod.value?.linked_services?.length || 0) > 0,
);

const detailTabs = computed(() => [
  {
    id: 'metadata',
    label: 'Metadata',
    count: labelEntries.value.length + annotationEntries.value.length,
  },
  {
    id: 'storage',
    label: 'Storage',
    count: allMounts.value.length + (pod.value?.volumes?.length || 0),
  },
  {
    id: 'linked',
    label: 'Linked resources',
    count:
      (pod.value?.configmaps?.length || 0)
      + (pod.value?.secrets?.length || 0)
      + (hasLinkedResources.value ? 1 : 0),
  },
]);
const isRunning = computed(() => String(pod.value?.phase || '').toLowerCase() === 'running');

const shortUid = computed(() => {
  const uid = String(pod.value?.uid || '');
  if (!uid) return '';
  return uid.length > 12 ? uid.slice(0, 12) : uid;
});

const primaryImage = computed(() => pod.value?.containers?.[0]?.image || '');

const heroIconClass = computed(() => {
  const phase = String(pod.value?.phase || '').toLowerCase();
  if (phase === 'running') return 'running';
  if (phase === 'pending') return 'pending';
  return 'stopped';
});

const actionClass = computed(() => {
  if (pendingAction.value === 'start') return 'success';
  if (pendingAction.value === 'restart') return 'warning';
  return 'error';
});

const memLimit = computed(() => liveStats.value.memory_limit || 0);

const cpuLimitLabel = computed(() => {
  const milli = liveStats.value.cpu_limit_millicores;
  if (!milli) return 'No Limit';
  if (milli % 1000 === 0) {
    const cpus = milli / 1000;
    return `${cpus} CPU${cpus === 1 ? '' : 's'}`;
  }
  return `${milli}m`;
});

const cpuDisplay = computed(() => {
  if (!liveStats.value.metrics_available) return '—';
  return `${liveStats.value.cpu.toFixed(1)}%`;
});

const memoryDisplay = computed(() => {
  if (!liveStats.value.metrics_available) return '—';
  return formatBytes(liveStats.value.memory);
});

const formatDate = (unix) => {
  if (!unix) return '—';
  return new Date(unix * 1000).toLocaleString();
};

const phaseClass = (phase) => {
  const normalized = String(phase || '').toLowerCase();
  if (normalized === 'running') return 'is-running';
  if (normalized === 'pending') return 'is-pending';
  if (normalized === 'failed' || normalized === 'unknown') return 'is-failed';
  return 'is-neutral';
};

const containerStateClass = (state) => {
  const normalized = String(state || '').toLowerCase();
  if (normalized === 'running') return 'is-running';
  if (normalized === 'waiting') return 'is-pending';
  if (normalized === 'terminated') return 'is-failed';
  return 'is-neutral';
};

const goToLogs = () => {
  router.push(logsRouteForPod(`${namespace.value}/${podName.value}`));
};

const goToShell = () => {
  router.push({ path: '/shell', query: { p: `${namespace.value}/${podName.value}` } });
};

function copyText(text, label) {
  navigator.clipboard.writeText(text).then(() => {
    showToast('Copied', `${label} copied to clipboard.`, 'success');
  });
}

function triggerConfirm(action) {
  pendingAction.value = action;
  showConfirm.value = true;
}

function closeConfirm() {
  if (actionPending.value) return;
  showConfirm.value = false;
  pendingAction.value = '';
}

async function confirmAction() {
  await runPodAction(pendingAction.value);
}

const runPodAction = async (action) => {
  if (actionPending.value) return;

  actionPending.value = true;
  try {
    const token = secureStorage.getItem('token');
    const formData = new URLSearchParams();
    formData.set('action', action);
    const res = await apiFetch(
      `/api/namespaces/${encodeURIComponent(namespace.value)}/pods/${encodeURIComponent(podName.value)}/action`,
      {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/x-www-form-urlencoded' },
        body: formData.toString(),
      },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error || `Failed to ${action} pod`);
    }
    showToast('Success', `Pod ${action} requested.`, 'success');
    showConfirm.value = false;
    pendingAction.value = '';
    if (action === 'remove' || action === 'stop' || action === 'restart' || action === 'start') {
      router.push(backToKubernetes.value);
      return;
    }
    await fetchPod();
  } catch (err) {
    showToast('Action failed', err.message || `Failed to ${action} pod`, 'error');
  } finally {
    actionPending.value = false;
  }
};

async function fetchPod() {
  loading.value = true;
  try {
    const token = secureStorage.getItem('token');
    const res = await apiFetch(
      `/api/namespaces/${encodeURIComponent(namespace.value)}/pods/${encodeURIComponent(podName.value)}`,
      { headers: { Authorization: `Bearer ${token}` } },
    );
    if (!res.ok) {
      pod.value = null;
      return;
    }
    pod.value = await res.json();
  } catch (err) {
    console.error(err);
    pod.value = null;
  } finally {
    loading.value = false;
  }
}

async function fetchLiveStats() {
  if (!isRunning.value) return;
  try {
    const token = secureStorage.getItem('token');
    const res = await apiFetch(
      `/api/namespaces/${encodeURIComponent(namespace.value)}/pods/${encodeURIComponent(podName.value)}/stats-now`,
      { headers: { Authorization: `Bearer ${token}` } },
    );
    if (!res.ok) return;
    const data = await res.json();
    liveStats.value = {
      cpu: Number(data.cpu) || 0,
      memory: Number(data.memory) || 0,
      memory_limit: Number(data.memory_limit) || 0,
      cpu_limit_millicores: Number(data.cpu_limit_millicores) || 0,
      metrics_available: data.metrics_available === true,
    };
  } catch (err) {
    console.error(err);
  }
}

function startStatsPolling() {
  stopStatsPolling();
  fetchLiveStats();
  statsTimer = setInterval(fetchLiveStats, 3000);
}

function stopStatsPolling() {
  if (statsTimer) {
    clearInterval(statsTimer);
    statsTimer = null;
  }
}

watch(isRunning, (running) => {
  if (running) startStatsPolling();
  else stopStatsPolling();
});

onMounted(async () => {
  await fetchPod();
  if (isRunning.value) startStatsPolling();
});

onUnmounted(() => {
  stopStatsPolling();
});
</script>

<style scoped>
.detail-view {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding-bottom: 2rem;
}

.detail-loading {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.shimmer-block {
  height: 120px;
  border-radius: var(--radius-2xl);
  background: linear-gradient(90deg, var(--bg-input) 25%, var(--bg-subtle) 50%, var(--bg-input) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.2s infinite;
}

.shimmer-block.wide {
  height: 280px;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

.detail-hero {
  padding: 1.35rem 1.5rem;
  border-radius: var(--radius-2xl);
  border: 1px solid var(--border);
  background: linear-gradient(135deg, var(--bg-card) 0%, rgba(var(--accent-rgb), 0.04) 100%);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.hero-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
}

.back-link {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  color: var(--text-dim);
  text-decoration: none;
  font-size: 0.85rem;
  font-weight: 700;
}

.back-link:hover {
  color: var(--accent);
}

.state-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.35rem 0.7rem;
  border-radius: 999px;
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border: 1px solid var(--border);
}

.state-pill.is-running {
  color: var(--success);
  border-color: rgba(var(--success-rgb), 0.35);
  background: rgba(var(--success-rgb), 0.08);
}

.state-pill.is-pending {
  color: var(--warning);
  border-color: rgba(var(--warning-rgb), 0.35);
  background: rgba(var(--warning-rgb), 0.08);
}

.state-pill.is-failed {
  color: var(--error);
  border-color: rgba(var(--error-rgb), 0.35);
  background: rgba(var(--error-rgb), 0.08);
}

.state-pill.is-neutral {
  color: var(--text-mute);
}

.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}

.state-pill.is-running .pulse-dot {
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.35; }
}

.hero-main {
  display: flex;
  gap: 1rem;
  align-items: flex-start;
}

.hero-icon {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.hero-icon.running {
  color: var(--success);
  background: rgba(var(--success-rgb), 0.08);
}

.hero-icon.pending {
  color: var(--warning);
  background: rgba(var(--warning-rgb), 0.08);
}

.hero-icon.stopped {
  color: var(--text-mute);
  background: var(--bg-input);
}

.hero-copy h1 {
  margin: 0 0 0.35rem;
  font-size: clamp(1.35rem, 2.5vw, 1.85rem);
  font-weight: 800;
  letter-spacing: -0.03em;
  color: var(--text-main);
}

.hero-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
  margin-bottom: 0.35rem;
}

.id-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-dim);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  cursor: pointer;
}

.id-chip:hover {
  border-color: var(--border-active);
  color: var(--accent);
}

.meta-dot,
.meta-image {
  color: var(--text-mute);
  font-size: 0.85rem;
}

.meta-image {
  max-width: min(420px, 55vw);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hero-status {
  margin: 0;
  color: var(--text-dim);
  font-size: 0.9rem;
}

.action-rail {
  display: flex;
  flex-wrap: wrap;
  gap: 0.55rem;
  margin-top: 1.25rem;
  padding-top: 1.15rem;
  border-top: 1px solid var(--border-subtle);
}

.action-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.55rem 0.9rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-main);
  font-size: 0.8rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-chip:hover {
  transform: translateY(-1px);
}

.action-chip.logs:hover {
  color: var(--accent);
  border-color: rgba(var(--accent-rgb), 0.35);
  background: var(--accent-soft);
}

.action-chip.shell:hover {
  color: #8b5cf6;
  border-color: rgba(139, 92, 246, 0.35);
  background: rgba(139, 92, 246, 0.1);
}

.action-chip.start:hover {
  color: var(--success);
  border-color: rgba(var(--success-rgb), 0.35);
  background: rgba(var(--success-rgb), 0.08);
}

.action-chip.stop:hover,
.action-chip.delete:hover {
  color: var(--error);
  border-color: rgba(var(--error-rgb), 0.35);
  background: rgba(var(--error-rgb), 0.08);
}

.action-chip.restart:hover {
  color: var(--accent);
  border-color: rgba(var(--accent-rgb), 0.35);
  background: var(--accent-soft);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.85rem;
}

.stat-card {
  padding: 1rem 1.1rem;
  border-radius: var(--radius-xl);
  border: 1px solid var(--border);
  background: var(--bg-card);
}

.stat-label {
  display: block;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: 0.35rem;
}

.stat-value {
  display: block;
  font-size: 1.35rem;
  font-weight: 800;
  color: var(--text-main);
  font-variant-numeric: tabular-nums;
}

.stat-value.sm {
  font-size: 0.95rem;
  font-weight: 700;
}

.stat-sub {
  display: block;
  margin-top: 0.2rem;
  font-size: 0.78rem;
  color: var(--text-dim);
}

.stat-hint {
  display: block;
  margin-top: 0.35rem;
  font-size: 0.68rem;
  color: var(--text-mute);
}

.stat-bar {
  margin-top: 0.65rem;
  height: 5px;
  border-radius: 999px;
  background: var(--bg-input);
  overflow: hidden;
}

.stat-bar-fill {
  height: 100%;
  border-radius: inherit;
}

.stat-bar-fill.accent {
  background: linear-gradient(90deg, var(--accent), #22d3ee);
}

.detail-panels {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.85rem;
}

.panel {
  border-radius: var(--radius-xl);
  border: 1px solid var(--border);
  background: var(--bg-card);
  padding: 1rem 1.1rem;
}

.panel-wide {
  grid-column: 1 / -1;
}

.panel-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 0.85rem;
}

.panel-head h2 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 800;
  color: var(--text-main);
}

.panel-count {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--text-mute);
}

.env-search {
  display: flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.45rem 0.7rem;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border);
  background: var(--bg-input);
  min-width: 220px;
}

.env-search input {
  border: none;
  outline: none;
  background: transparent;
  color: var(--text-main);
  font-size: 0.82rem;
  width: 100%;
}

.kv-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.75rem 1rem;
}

.kv-label {
  display: block;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: 0.2rem;
}

.kv-value {
  color: var(--text-main);
  font-size: 0.88rem;
  word-break: break-word;
}

.port-list,
.env-list,
.key-list {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
  max-height: 280px;
  overflow: auto;
}

.key-list.flush {
  max-height: 220px;
}

.port-row,
.env-row,
.key-row {
  padding: 0.55rem 0.7rem;
  border-radius: var(--radius-sm);
  background: var(--bg-input);
  border: 1px solid var(--border-subtle);
  font-size: 0.82rem;
}

.env-row {
  display: grid;
  grid-template-columns: minmax(120px, 34%) 1fr;
  gap: 0.75rem;
}

.container-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 0.75rem;
}

.container-card {
  padding: 0.85rem 0.95rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: linear-gradient(180deg, var(--bg-input) 0%, var(--bg-card) 100%);
}

.container-card.init {
  border-style: dashed;
  opacity: 0.92;
}

.container-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.65rem;
  margin-bottom: 0.55rem;
}

.container-card-title {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  min-width: 0;
}

.container-card-title strong {
  font-size: 0.9rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.init-badge {
  padding: 0.1rem 0.4rem;
  border-radius: 999px;
  background: var(--bg-subtle);
  border: 1px solid var(--border-subtle);
  font-size: 0.62rem;
  font-weight: 800;
  text-transform: uppercase;
  color: var(--text-mute);
}

.container-card-image {
  margin: 0 0 0.65rem;
  font-size: 0.76rem;
  color: var(--text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.container-card-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.55rem 0.75rem;
}

.mini-label {
  display: block;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: 0.15rem;
}

.mini-value {
  font-size: 0.78rem;
  color: var(--text-main);
}

.detail-tabs-panel {
  padding-top: 0.85rem;
}

.detail-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-bottom: 1rem;
  padding-bottom: 0.85rem;
  border-bottom: 1px solid var(--border-subtle);
}

.detail-tab {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.8rem;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-dim);
  font-size: 0.78rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s ease;
}

.detail-tab.active {
  color: var(--accent);
  border-color: rgba(var(--accent-rgb), 0.35);
  background: var(--accent-soft);
}

.tab-count {
  padding: 0.05rem 0.35rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  font-size: 0.65rem;
}

.tab-panel h3 {
  margin: 0 0 0.55rem;
  font-size: 0.78rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-mute);
}

.tab-split {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 1rem;
  margin-bottom: 1rem;
}

.tab-block + .tab-block,
.tab-split + .tab-block {
  margin-top: 1rem;
}

.tag-list {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.tag-chip {
  padding: 0.28rem 0.5rem;
  border-radius: 999px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-input);
  font-size: 0.72rem;
  color: var(--text-dim);
}

.tag-key {
  display: block;
  color: var(--accent);
  font-weight: 600;
  margin-bottom: 0.2rem;
}

.tag-val {
  display: block;
  color: var(--text-dim);
  word-break: break-word;
}

.linked-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.55rem;
  margin-bottom: 1rem;
}

.linked-card {
  padding: 0.65rem 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--bg-input);
}

.linked-label {
  display: block;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--text-mute);
  margin-bottom: 0.2rem;
}

.linked-value {
  font-size: 0.8rem;
  color: var(--text-main);
  word-break: break-word;
}

.resource-list {
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  max-height: 280px;
  overflow: auto;
}

.resource-card {
  padding: 0.65rem 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--bg-input);
}

.resource-card-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.35rem;
}

.env-key {
  color: var(--accent);
  font-weight: 600;
}

.env-val {
  color: var(--text-dim);
  word-break: break-all;
}

.kv-inline {
  margin-top: 0.45rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.kv-inline-row {
  display: grid;
  grid-template-columns: minmax(160px, 34%) 1fr;
  gap: 0.6rem;
  background: var(--bg-subtle);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  padding: 0.4rem 0.5rem;
}

.kv-key { color: var(--accent); }
.kv-val { color: var(--text-dim); word-break: break-word; }
.sub-text { color: var(--text-mute); font-size: 0.78rem; }
.mono { font-family: var(--font-mono); }

.panel-empty {
  margin: 0;
  color: var(--text-mute);
  font-size: 0.88rem;
}

.detail-empty {
  padding: 3rem 1rem;
  text-align: center;
}

.detail-empty h2 { margin: 0 0 0.5rem; }
.detail-empty p { color: var(--text-dim); margin: 0 0 1rem; }

.back-btn {
  display: inline-flex;
  padding: 0.65rem 1rem;
  border-radius: var(--radius-md);
  background: var(--accent);
  color: #fff;
  text-decoration: none;
  font-weight: 700;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(2, 6, 23, 0.55);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  width: min(420px, 100%);
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: 1.5rem;
}

.modal-icon {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
  background: var(--bg-input);
}

.modal-icon.success { color: var(--success); }
.modal-icon.warning { color: var(--warning); }
.modal-icon.error { color: var(--error); }

.modal-text-center { text-align: center; margin-bottom: 1rem; }
.modal-text-center h3 { margin: 0 0 0.5rem; }
.modal-text-center p { margin: 0; color: var(--text-dim); }

.modal-actions { display: flex; gap: 0.65rem; }

.modal-btn {
  flex: 1;
  padding: 0.7rem 1rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  font-weight: 700;
  cursor: pointer;
}

.modal-btn.cancel {
  background: var(--bg-input);
  color: var(--text-main);
}

.modal-btn.confirm {
  color: #fff;
  border: none;
}

.modal-btn.confirm.success { background: var(--success); }
.modal-btn.confirm.warning { background: var(--warning); }
.modal-btn.confirm.error { background: var(--error); }

@media (max-width: 960px) {
  .stats-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .detail-panels,
  .kv-grid,
  .env-row,
  .kv-inline-row,
  .tab-split,
  .container-card-grid { grid-template-columns: 1fr; }
}

@media (max-width: 640px) {
  .hero-main { flex-direction: column; }
  .stats-grid { grid-template-columns: 1fr; }
  .panel-head { flex-direction: column; align-items: stretch; }
  .env-search { min-width: 0; width: 100%; }
}
</style>
