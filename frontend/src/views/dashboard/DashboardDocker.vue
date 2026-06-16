<template>
  <div class="dashboard">
    <!-- Hero -->
    <section class="dash-hero animate-slide-up">
      <div class="hero-body">
        <div class="hero-copy">
          <span class="hero-eyebrow">
            <span class="live-dot" aria-hidden="true"></span>
            Live overview
          </span>
          <h1 class="hero-title">{{ greeting }}, {{ username }}</h1>
          <p class="hero-sub">
            {{ containers.length }} container{{ containers.length === 1 ? "" : "s" }} across your fleet
            <span v-if="runningCount" class="hero-accent"> · {{ runningCount }} running</span>
          </p>
        </div>
        <div class="hero-actions">
          <router-link to="/health" class="dash-action-btn">
            <AppIcon name="activity" />
            Diagnostics
          </router-link>
          <router-link to="/containers" class="dash-action-btn">
            <AppIcon name="containers" />
            All containers
          </router-link>
          <button class="dash-action-btn primary" @click="refresh" :disabled="loading">
            <AppIcon name="refresh" :class="{ spinning: loading }" />
            Refresh
          </button>
        </div>
      </div>
      <div class="hero-mesh" aria-hidden="true"></div>
    </section>

    <!-- Metrics -->
    <section class="metrics-bento animate-slide-up" style="animation-delay: 0.05s">
      <article class="metric-card variant-fleet">
        <div class="metric-glow" aria-hidden="true"></div>
        <div class="metric-top">
          <div class="metric-icon">
            <AppIcon name="containers" />
          </div>
          <span class="metric-badge">Fleet</span>
        </div>
        <div class="metric-body">
          <div class="metric-main">
            <span class="metric-value">{{ containers.length }}</span>
            <span class="metric-label">Total containers</span>
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
          <span>{{ runningCount }} running · {{ stoppedCount }} stopped</span>
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
              <span v-for="n in 4" :key="n" class="pulse-bar" :style="{ height: pulseHeights[n - 1] + '%', animationDelay: n * 0.12 + 's' }"></span>
            </div>
          </div>
        </div>
        <div class="metric-footer success-text">
          {{ runningRatio }}% of fleet is healthy
        </div>
      </article>

      <article class="metric-card variant-idle">
        <div class="metric-glow" aria-hidden="true"></div>
        <div class="metric-top">
          <div class="metric-icon muted">
            <AppIcon name="stopOutline" />
          </div>
          <span class="metric-badge dim">Idle</span>
        </div>
        <div class="metric-body">
          <div class="metric-main">
            <span class="metric-value">{{ stoppedCount }}</span>
            <span class="metric-label">Stopped</span>
          </div>
          <div class="metric-visual">
            <div class="donut muted" :style="{ '--pct': stoppedRatio }">
              <svg viewBox="0 0 36 36">
                <circle class="donut-track" cx="18" cy="18" r="15.5" />
                <circle class="donut-fill muted" cx="18" cy="18" r="15.5" />
              </svg>
              <span class="donut-label">{{ stoppedRatio }}%</span>
            </div>
          </div>
        </div>
        <div class="metric-footer" :class="{ warn: stoppedCount > 0 }">
          {{ stoppedCount ? `${stoppedCount} container${stoppedCount === 1 ? '' : 's'} offline` : "All services operational" }}
        </div>
      </article>

      <DashboardHostCard />
    </section>

    <!-- Container table -->
    <section class="table-panel animate-slide-up" style="animation-delay: 0.1s">
      <div class="panel-toolbar">
        <div class="toolbar-left">
          <h2>Container overview</h2>
          <p class="toolbar-sub">Manage workloads and jump into logs</p>
        </div>
        <div class="toolbar-right">
          <div class="filter-pills">
            <button
              v-for="f in filters"
              :key="f.value"
              class="filter-pill"
              :class="{ active: stateFilter === f.value }"
              @click="stateFilter = f.value"
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
      <ContainerTable
        :state-filter="stateFilter"
        embedded
        :max-rows="5"
        view-all-to="/containers"
      />
    </section>
  </div>
</template>

<script setup>
import { ref, computed } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import ContainerTable from "../../components/ContainerTable.vue";
import DashboardHostCard from "../../components/dashboard/DashboardHostCard.vue";
import { useContainers } from "../../composables/useContainers";
import { useDashboardShared } from "../../composables/useDashboardShared";
import { sharedState } from "../../utils/sharedState";

const stateFilter = ref("all");
const { containers, loading, runningCount, stoppedCount, fetchContainers } = useContainers();
const { username, greeting } = useDashboardShared();

const runningRatio = computed(() => {
  if (!containers.value.length) return 0;
  return Math.round((runningCount.value / containers.value.length) * 100);
});

const stoppedRatio = computed(() => {
  if (!containers.value.length) return 0;
  return Math.round((stoppedCount.value / containers.value.length) * 100);
});

const pulseHeights = computed(() => {
  const base = Math.max(35, Math.min(95, runningRatio.value));
  return [base * 0.55, base * 0.85, base, base * 0.7];
});

const filters = computed(() => [
  { label: "All", value: "all", count: containers.value.length },
  { label: "Running", value: "running", count: runningCount.value },
  { label: "Stopped", value: "stopped", count: stoppedCount.value },
]);

const refresh = () => fetchContainers();
</script>

<style scoped>
@import './dashboard.css';
</style>
