<template>
  <div class="pods-view animate-fade-in">
    <div v-if="!sharedState.k8sAvailable" class="k8s-unavailable-banner">
      <strong>Kubernetes is not connected.</strong>
      <p>{{ sharedState.k8sError || 'Mount a kubeconfig or deploy DockLog in-cluster with a ServiceAccount.' }}</p>
    </div>
    <div v-else-if="fetchError" class="k8s-unavailable-banner">
      <strong>Could not load Kubernetes data.</strong>
      <p>{{ fetchError }}</p>
    </div>

    <section class="pods-hero">
      <div class="hero-copy">
        <span class="hero-eyebrow">Cluster workloads</span>
        <h1>Kubernetes pods</h1>
        <p class="hero-sub">
          Browse pods across namespaces configured for this DockLog instance.
        </p>
      </div>
      <div class="hero-stats">
        <div class="hero-stat">
          <span class="hero-stat-val">{{ pods.length }}</span>
          <span class="hero-stat-lbl">Total</span>
        </div>
        <div class="hero-stat success">
          <span class="hero-stat-val">{{ runningCount }}</span>
          <span class="hero-stat-lbl">Running</span>
        </div>
        <div class="hero-stat warning">
          <span class="hero-stat-val">{{ pendingCount }}</span>
          <span class="hero-stat-lbl">Pending</span>
        </div>
        <div class="hero-stat muted">
          <span class="hero-stat-val">{{ failedCount }}</span>
          <span class="hero-stat-lbl">Failed</span>
        </div>
      </div>
      <div class="hero-mesh" aria-hidden="true"></div>
    </section>

    <section class="pods-panel">
      <div class="panel-toolbar">
        <div class="toolbar-left">
          <div class="namespace-select">
            <label for="namespace">Namespace</label>
            <select
              id="namespace"
              v-model="selectedNamespace"
              :disabled="namespacesLoading || namespaces.length === 0"
            >
              <option v-if="namespacesLoading" value="">Loading...</option>
              <option v-else-if="namespaces.length === 0" value="">No namespaces</option>
              <option v-for="ns in namespaces" :key="ns" :value="ns">
                {{ ns }}
              </option>
            </select>
          </div>

          <div class="filter-pills">
            <button
              v-for="f in filters"
              :key="f.value"
              class="filter-pill"
              :class="{ active: phaseFilter === f.value }"
              @click="phaseFilter = f.value"
            >
              {{ f.label }}
              <span class="pill-count">{{ f.count }}</span>
            </button>
          </div>
        </div>

        <div class="toolbar-right">
          <div class="search-box">
            <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.5">
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <input
              type="text"
              v-model="sharedState.searchQuery"
              placeholder="Filter by pod or image..."
            />
          </div>
          <button class="refresh-btn" @click="refresh" :disabled="loading">
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
              :class="{ spinning: loading }"
            >
              <polyline points="23 4 23 10 17 10"></polyline>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
            </svg>
            Refresh
          </button>
        </div>
      </div>

      <div class="premium-table-container embedded">
        <table class="premium-table pods-table">
          <thead>
            <tr>
              <th>Pod</th>
              <th>Namespace</th>
              <th>Images</th>
              <th>Ready</th>
              <th>Restarts</th>
              <th>Node</th>
              <th>Created</th>
              <th>Phase</th>
              <th class="text-right">Actions</th>
            </tr>
          </thead>
          <tbody v-if="loading">
            <tr>
              <td colspan="9">
                <div class="table-loading">
                  <div class="shimmer"></div>
                </div>
              </td>
            </tr>
          </tbody>
          <tbody v-else-if="displayPods.length > 0">
            <tr
              v-for="pod in displayPods"
              :key="pod.uid || `${pod.namespace}/${pod.name}`"
              class="pod-row"
              :class="{ 'is-running': pod.phase === 'Running' }"
            >
              <td data-label="Pod">
                <div class="name-cell">
                  <div class="pod-avatar" :class="phaseClass(pod.phase)">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <rect x="3" y="3" width="18" height="18" rx="3"></rect>
                      <path d="M9 9h6v6H9z"></path>
                    </svg>
                  </div>
                  <div class="name-main">
                    <span class="pod-title">{{ pod.name }}</span>
                    <span class="pod-status">{{ pod.status }}</span>
                  </div>
                </div>
              </td>
              <td data-label="Namespace">
                <span class="ns-pill">{{ pod.namespace }}</span>
              </td>
              <td data-label="Images">
                <div class="images-cell">
                  <span
                    v-for="image in pod.images || []"
                    :key="image"
                    class="image-chip"
                  >
                    {{ image }}
                  </span>
                  <span v-if="!pod.images?.length" class="text-mute">—</span>
                </div>
              </td>
              <td data-label="Ready">
                <span class="ready-label">{{ pod.ready }}</span>
              </td>
              <td data-label="Restarts">
                <span class="restart-label">{{ pod.restarts }}</span>
              </td>
              <td data-label="Node">
                <span class="node-label">{{ pod.node || '—' }}</span>
              </td>
              <td data-label="Created">
                <span class="date-label">{{ formatDate(pod.created) }}</span>
              </td>
              <td data-label="Phase">
                <div :class="['status-pill', phaseClass(pod.phase)]">
                  <span class="pulse-dot"></span>
                  {{ (pod.phase || 'Unknown').toUpperCase() }}
                </div>
              </td>
              <td class="text-right" data-label="Actions">
                <button class="logs-btn" @click="goToLogs(pod)">
                  Logs
                </button>
              </td>
            </tr>
          </tbody>
          <tbody v-else>
            <tr>
              <td colspan="9" class="empty-state">
                <p>No pods found in this namespace.</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { usePods } from '../composables/usePods';
import { sharedState } from '../utils/sharedState';
import { logsRouteForPod } from '../utils/logRoutes';

const router = useRouter();

const {
  pods,
  namespaces,
  loading,
  namespacesLoading,
  selectedNamespace,
  phaseFilter,
  filteredPods,
  runningCount,
  pendingCount,
  failedCount,
  refresh,
  formatDate,
  fetchError,
} = usePods();

const displayPods = computed(() => filteredPods.value);

const filters = computed(() => [
  { label: 'All', value: 'all', count: pods.value.length },
  { label: 'Running', value: 'running', count: runningCount.value },
  { label: 'Pending', value: 'pending', count: pendingCount.value },
  { label: 'Failed', value: 'failed', count: failedCount.value },
]);

const phaseClass = (phase) => {
  const normalized = (phase || '').toLowerCase();
  if (normalized === 'running') return 'is-running';
  if (normalized === 'pending') return 'is-pending';
  if (normalized === 'failed' || normalized === 'unknown') return 'is-failed';
  return 'is-stopped';
};

const goToLogs = (pod) => {
  router.push(logsRouteForPod(pod));
};
</script>

<style scoped>
.k8s-unavailable-banner {
  padding: 1rem 1.25rem;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--warning-rgb), 0.35);
  background: rgba(var(--warning-rgb), 0.08);
  color: var(--text-main);
}

.k8s-unavailable-banner strong {
  display: block;
  margin-bottom: 0.35rem;
  color: var(--warning);
}

.k8s-unavailable-banner p {
  margin: 0;
  font-size: 0.85rem;
  color: var(--text-dim);
  line-height: 1.5;
}

.pods-view {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding-bottom: 2rem;
}

.pods-hero,
.pods-panel {
  border-radius: var(--radius-2xl);
  border: 1px solid var(--border);
  background: var(--bg-card);
}

.pods-hero {
  position: relative;
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 1.5rem;
  flex-wrap: wrap;
  padding: 1.5rem 1.75rem;
  background: linear-gradient(135deg, var(--bg-card) 0%, var(--bg-card) 55%, rgba(var(--accent-rgb), 0.04) 100%);
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.hero-mesh {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(ellipse 70% 80% at 100% 0%, rgba(var(--accent-rgb), 0.14), transparent 55%),
    radial-gradient(ellipse 40% 50% at 0% 100%, rgba(var(--success-rgb), 0.06), transparent 50%);
  pointer-events: none;
}

.hero-copy {
  position: relative;
  z-index: 1;
}

.hero-eyebrow {
  display: block;
  font-size: 0.72rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--accent);
  margin-bottom: 0.4rem;
}

.pods-hero h1 {
  font-size: clamp(1.35rem, 2.5vw, 1.75rem);
  font-weight: 800;
  letter-spacing: -0.03em;
  margin: 0 0 0.35rem;
  color: var(--text-main);
}

.hero-sub {
  margin: 0;
  font-size: 0.9rem;
  color: var(--text-dim);
  max-width: 520px;
}

.hero-stats {
  position: relative;
  z-index: 1;
  display: flex;
  gap: 0.65rem;
  flex-wrap: wrap;
}

.hero-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 72px;
  padding: 0.65rem 1rem;
  border-radius: var(--radius-md);
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.hero-stat-val {
  font-size: 1.35rem;
  font-weight: 800;
  letter-spacing: -0.03em;
  color: var(--text-main);
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.hero-stat-lbl {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-mute);
}

.hero-stat.success .hero-stat-val { color: var(--success); }
.hero-stat.warning .hero-stat-val { color: var(--warning); }
.hero-stat.muted .hero-stat-val { color: var(--text-dim); }

.pods-panel {
  padding: 1.15rem 1.15rem 0.35rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.panel-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1rem;
  flex-wrap: wrap;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.namespace-select {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.namespace-select label {
  font-size: 0.65rem;
  font-weight: 800;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--text-mute);
}

.namespace-select select {
  min-width: 180px;
  padding: 0.55rem 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 600;
}

.filter-pills {
  display: flex;
  gap: 0.35rem;
  padding: 0.25rem;
  border-radius: var(--radius-md);
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.filter-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.45rem 0.8rem;
  border-radius: calc(var(--radius-md) - 2px);
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--text-dim);
  transition: all 0.2s ease;
}

.filter-pill:hover {
  color: var(--text-main);
  background: var(--bg-subtle);
}

.filter-pill.active {
  background: var(--accent);
  color: #fff;
  box-shadow: 0 4px 12px rgba(var(--accent-rgb), 0.3);
}

.pill-count {
  font-size: 0.65rem;
  padding: 0.1rem 0.35rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.15);
  font-variant-numeric: tabular-nums;
}

.filter-pill:not(.active) .pill-count {
  background: var(--bg-subtle);
  color: var(--text-mute);
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.6rem 1rem;
  border-radius: var(--radius-md);
  background: var(--bg-input);
  border: 1px solid var(--border);
  min-width: 260px;
}

.search-box input {
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 600;
  width: 100%;
}

.refresh-btn {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.6rem 1rem;
  border-radius: var(--radius-md);
  background: var(--accent);
  color: #fff;
  font-size: 0.8rem;
  font-weight: 700;
}

.refresh-btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

.spinning {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.pod-avatar {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.pod-avatar svg {
  width: 18px;
  height: 18px;
}

.pod-avatar.is-running {
  color: var(--success);
  border-color: rgba(var(--success-rgb), 0.35);
}

.pod-avatar.is-pending {
  color: var(--warning);
}

.pod-avatar.is-failed {
  color: var(--error);
}

.name-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.pod-title {
  font-weight: 700;
  color: var(--text-main);
}

.pod-status {
  font-size: 0.75rem;
  color: var(--text-mute);
}

.ns-pill,
.image-chip,
.ready-label,
.restart-label,
.node-label,
.date-label {
  font-size: 0.8rem;
  font-weight: 600;
}

.ns-pill {
  display: inline-flex;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.images-cell {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.image-chip {
  display: inline-flex;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
}

.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.04em;
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.status-pill.is-running {
  color: var(--success);
  border-color: rgba(var(--success-rgb), 0.35);
}

.status-pill.is-pending {
  color: var(--warning);
}

.status-pill.is-failed {
  color: var(--error);
}

.empty-state {
  text-align: center;
  padding: 2rem 1rem;
  color: var(--text-mute);
}

.logs-btn {
  padding: 0.4rem 0.75rem;
  border-radius: var(--radius-md);
  background: var(--accent);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 700;
}

.logs-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

@media (max-width: 900px) {
  .pods-hero {
    flex-direction: column;
    align-items: stretch;
  }

  .panel-toolbar {
    flex-direction: column;
    align-items: stretch;
  }

  .toolbar-left,
  .toolbar-right,
  .search-box,
  .refresh-btn,
  .namespace-select select {
    width: 100%;
    min-width: 0;
  }
}
</style>
