<template>
  <article class="metric-card variant-host">
    <div class="metric-glow" aria-hidden="true"></div>
    <div class="metric-top">
      <div class="metric-icon">
        <AppIcon name="server" />
      </div>
      <span class="metric-badge">Host</span>
    </div>
    <div class="metric-body host-body">
      <div class="host-stat">
        <div class="host-stat-head">
          <span>CPU</span>
          <strong :style="{ color: statColor(cpuPercent) }">{{ cpuPercent.toFixed(1) }}%</strong>
        </div>
        <div class="host-track">
          <div
            class="host-fill"
            :style="{ width: cpuPercent + '%', background: statColor(cpuPercent) }"
          ></div>
        </div>
        <span class="host-meta" v-if="sharedState.systemStats?.cores">
          {{ sharedState.systemStats.cores }} core{{ sharedState.systemStats.cores > 1 ? 's' : '' }}
        </span>
      </div>
      <div class="host-stat">
        <div class="host-stat-head">
          <span>Memory</span>
          <strong>{{ formatBytes(memUsed) }}</strong>
        </div>
        <div class="host-track">
          <div class="host-fill accent" :style="{ width: memPercent + '%' }"></div>
        </div>
        <span class="host-meta">{{ memPercent.toFixed(0) }}% of {{ formatBytes(memTotal) }}</span>
      </div>
    </div>
    <div class="metric-footer">{{ hostStatusLabel }}</div>
  </article>
</template>

<script setup>
import AppIcon from '../AppIcon.vue';
import { useDashboardShared } from '../../composables/useDashboardShared';

const {
  sharedState,
  cpuPercent,
  memUsed,
  memTotal,
  memPercent,
  hostStatusLabel,
  statColor,
  formatBytes,
} = useDashboardShared();
</script>

<style scoped>
@import '../../views/dashboard/dashboard.css';
</style>
