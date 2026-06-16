<template>
  <div class="dashboard">
    <div v-if="!sharedState.k8sAvailable" class="k8s-unavailable-banner">
      <strong>Kubernetes is not connected.</strong>
      <p>{{ sharedState.k8sError || 'Mount a kubeconfig or deploy DockLog in-cluster with a ServiceAccount.' }}</p>
    </div>

    <section class="dash-hero animate-slide-up">
      <div class="hero-body">
        <div class="hero-copy">
          <span class="hero-eyebrow">
            <span class="live-dot" aria-hidden="true"></span>
            Cluster overview
          </span>
          <h1 class="hero-title">{{ greeting }}, {{ username }}</h1>
          <p class="hero-sub">
            {{ podTotal }} pod{{ podTotal === 1 ? '' : 's' }} in {{ selectedNamespace || 'namespace' }}
            <span v-if="runningCount" class="hero-accent"> · {{ runningCount }} running</span>
          </p>
        </div>
        <div class="hero-actions">
          <router-link to="/health" class="dash-action-btn">
            <AppIcon name="activity" />
            Diagnostics
          </router-link>
          <router-link to="/kubernetes" class="dash-action-btn">
            <AppIcon name="box" />
            Kubernetes
          </router-link>
          <button class="dash-action-btn primary" @click="refresh" :disabled="loading">
            <AppIcon name="refresh" :class="{ spinning: loading }" />
            Refresh
          </button>
        </div>
      </div>
      <div class="hero-mesh hero-mesh-k8s" aria-hidden="true"></div>
    </section>

    <section class="metrics-bento animate-slide-up" style="animation-delay: 0.05s">
      <article class="metric-card variant-fleet">
        <div class="metric-glow" aria-hidden="true"></div>
        <div class="metric-top">
          <div class="metric-icon">
            <AppIcon name="containers" />
          </div>
          <span class="metric-badge">Pods</span>
        </div>
        <div class="metric-body">
          <div class="metric-main">
            <span class="metric-value">{{ podTotal }}</span>
            <span class="metric-label">Total pods</span>
          </div>
          <div class="metric-visual">
            <div class="donut" :style="{ '--pct': runningRatio }">
              <svg viewBox="0 0 36 36">
                <circle class="donut-track" cx="18" cy="18" r="15.5" />
                <circle class="donut-fill success" cx="18" cy="18" r="15.5" />
              </svg>
              <span class="donut-label">{{ runningRatio }}%</span>
            </div>
          </div>
        </div>
        <div class="metric-footer">
          <div class="metric-bar">
            <div class="metric-bar-fill success" :style="{ width: runningRatio + '%' }"></div>
          </div>
          <span>{{ runningCount }} running · {{ pendingCount }} pending · {{ failedCount }} failed</span>
        </div>
      </article>

      <article class="metric-card variant-live">
        <div class="metric-glow" aria-hidden="true"></div>
        <div class="metric-top">
          <div class="metric-icon success">
            <AppIcon name="activity" />
          </div>
          <span class="metric-badge success">Live</span>
        </div>
        <div class="metric-body">
          <div class="metric-main">
            <span class="metric-value">{{ runningCount }}</span>
            <span class="metric-label">Running now</span>
          </div>
          <div class="metric-visual">
            <div class="pulse-stack">
              <span
                v-for="n in 4"
                :key="n"
                class="pulse-bar"
                :style="{ height: pulseHeights[n - 1] + '%', animationDelay: n * 0.12 + 's' }"
              ></span>
            </div>
          </div>
        </div>
        <div class="metric-footer success-text">{{ runningRatio }}% of pods are healthy</div>
      </article>

      <article class="metric-card variant-idle">
        <div class="metric-glow" aria-hidden="true"></div>
        <div class="metric-top">
          <div class="metric-icon" :class="warningCount > 0 ? 'warning' : 'muted'">
            <AppIcon name="bell" />
          </div>
          <span class="metric-badge" :class="warningCount > 0 ? 'warning-badge' : 'dim'">Cluster</span>
        </div>
        <div class="metric-body">
          <div class="metric-main">
            <span class="metric-value">{{ overview?.deployments ?? 0 }}</span>
            <span class="metric-label">Deployments</span>
          </div>
          <div class="metric-visual metric-stat-pill">{{ overview?.services ?? 0 }} svc</div>
        </div>
        <div class="metric-footer" :class="{ warn: warningCount > 0 }">
          {{ warningCount > 0 ? `${warningCount} warning events` : `${overview?.hpas ?? 0} HPAs configured` }}
        </div>
      </article>

      <DashboardHostCard />
    </section>

    <section class="table-panel animate-slide-up" style="animation-delay: 0.1s">
      <div class="panel-toolbar">
        <div class="toolbar-left">
          <h2>Pod overview</h2>
          <p class="toolbar-sub">Browse pods and jump into logs</p>
        </div>
        <div class="toolbar-right">
          <select v-model="selectedNamespace" class="namespace-select" :disabled="namespacesLoading">
            <option v-if="namespacesLoading" value="">Loading...</option>
            <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
          </select>
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
          <div class="search-box glass">
            <AppIcon name="search" :size="16" />
            <input type="text" v-model="sharedState.searchQuery" placeholder="Search..." />
          </div>
        </div>
      </div>
      <PodTable
        :pods="filteredPods"
        :loading="loading"
        embedded
        :max-rows="5"
        :view-all-to="kubernetesPodsLink"
        @refresh="refresh"
      />
    </section>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import AppIcon from '../../components/AppIcon.vue';
import PodTable from '../../components/PodTable.vue';
import DashboardHostCard from '../../components/dashboard/DashboardHostCard.vue';
import { useKubernetes } from '../../composables/useKubernetes';
import { useDashboardShared } from '../../composables/useDashboardShared';

const {
  pods,
  overview,
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
} = useKubernetes();

const { username, greeting, sharedState } = useDashboardShared();

const podTotal = computed(() => pods.value.length);
const warningCount = computed(() => overview.value?.warning_events ?? 0);

const runningRatio = computed(() => {
  if (!podTotal.value) return 0;
  return Math.round((runningCount.value / podTotal.value) * 100);
});

const pulseHeights = computed(() => {
  const base = Math.max(35, Math.min(95, runningRatio.value));
  return [base * 0.55, base * 0.85, base, base * 0.7];
});

const filters = computed(() => [
  { label: 'All', value: 'all', count: pods.value.length },
  { label: 'Running', value: 'running', count: runningCount.value },
  { label: 'Pending', value: 'pending', count: pendingCount.value },
  { label: 'Failed', value: 'failed', count: failedCount.value },
]);

const kubernetesPodsLink = computed(() => ({
  path: '/kubernetes',
  query: { tab: 'pods', namespace: selectedNamespace.value },
}));
</script>

<style scoped>
@import './dashboard.css';

.k8s-unavailable-banner {
  padding: 1rem 1.25rem;
  border-radius: var(--radius-lg);
  border: 1px solid rgba(var(--warning-rgb), 0.35);
  background: rgba(var(--warning-rgb), 0.08);
  margin-bottom: 0.25rem;
}

.k8s-unavailable-banner strong {
  display: block;
  margin-bottom: 0.35rem;
}

.k8s-unavailable-banner p {
  margin: 0;
  color: var(--text-dim);
  font-size: 0.88rem;
}

.hero-mesh-k8s {
  background:
    radial-gradient(ellipse 80% 60% at 100% 0%, rgba(59, 130, 246, 0.16), transparent 55%),
    radial-gradient(ellipse 50% 40% at 0% 100%, rgba(var(--success-rgb), 0.08), transparent 50%);
}

.metric-icon.warning {
  background: rgba(var(--warning-rgb), 0.1);
  color: var(--warning);
  border-color: rgba(var(--warning-rgb), 0.15);
}

.metric-badge.warning-badge {
  background: rgba(var(--warning-rgb), 0.1);
  color: var(--warning);
  border-color: rgba(var(--warning-rgb), 0.15);
}

.metric-stat-pill {
  font-size: 0.68rem;
  font-weight: 800;
  padding: 0.35rem 0.55rem;
  border-radius: 999px;
  background: var(--bg-subtle);
  color: var(--text-dim);
  border: 1px solid var(--border);
}

.namespace-select {
  min-width: 140px;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-main);
  font-size: 0.8rem;
  font-weight: 600;
}
</style>
