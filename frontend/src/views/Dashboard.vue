<template>
  <div class="dashboard-container">
    <div class="summary-grid animate-slide-up">
      <div class="premium-stat-card">
        <div class="stat-header">
          <div class="stat-icon">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>
              <polyline points="2 17 12 22 22 17"></polyline>
              <polyline points="2 12 12 17 22 12"></polyline>
            </svg>
          </div>
          <span class="badge badge-dim">Total</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">TOTAL CONTAINERS</span>
          <span class="stat-value">{{ containers.length }}</span>
        </div>
      </div>

      <div class="premium-stat-card">
        <div class="stat-header">
          <div class="stat-icon success">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"></polyline>
            </svg>
          </div>
          <span class="badge badge-success">Active</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">RUNNING</span>
          <span class="stat-value">{{ runningCount }}</span>
        </div>
      </div>

      <div class="premium-stat-card">
        <div class="stat-header">
          <div class="stat-icon error">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
              <path d="M18.36 6.64a9 9 0 1 1-12.73 0"></path>
              <line x1="12" y1="2" x2="12" y2="12"></line>
            </svg>
          </div>
          <span class="badge badge-error">Stopped</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">STOPPED</span>
          <span class="stat-value">{{ stoppedCount }}</span>
        </div>
      </div>
    </div>

    <div class="dashboard-grid full-width">
      <div class="grid-section">
        <div class="section-header">
          <h3>Container Overview</h3>
          <div class="header-actions">
            <div class="search-box glass">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="3">
                <circle cx="11" cy="11" r="8"></circle>
                <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
              </svg>
              <input type="text" v-model="sharedState.searchQuery" placeholder="Search..." />
            </div>
          </div>
        </div>
        <ContainerTable />
      </div>
    </div>
  </div>
</template>

<script setup>
import ContainerTable from "../components/ContainerTable.vue";
import { useContainers } from "../composables/useContainers";
import { sharedState } from "../utils/sharedState";

const { containers, runningCount, stoppedCount } = useContainers();
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  padding-bottom: 1.5rem;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 2rem;
  align-items: start;
}

.dashboard-grid.full-width {
  grid-template-columns: 1fr;
}

.grid-section {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  min-width: 0;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.section-header h3 {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--text-main);
  letter-spacing: -0.02em;
  margin: 0;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.5rem 0.85rem;
  border-radius: 10px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  width: 180px;
}

.search-box input {
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 0.75rem;
  font-weight: 700;
  width: 100%;
  outline: none;
}

.search-box svg {
  color: var(--text-mute);
}

@media (max-width: 850px) {
  .summary-grid {
    grid-template-columns: repeat(2, 1fr) !important;
    gap: 1rem !important;
  }
  .dashboard-grid {
    grid-template-columns: 1fr !important;
    gap: 1.5rem !important;
  }
  .section-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 1.25rem;
  }
  .header-actions {
    width: 100%;
  }
  .search-box {
    width: 100% !important;
    min-width: 0 !important;
  }
}

@media (max-width: 480px) {
  .summary-grid {
    grid-template-columns: 1fr !important;
  }
}
</style>
