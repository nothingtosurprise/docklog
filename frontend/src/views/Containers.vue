<template>
  <div class="containers-view animate-fade-in">
    <div class="view-header">
      <div class="header-info">
        <h1>Container Management</h1>
        <p class="text-mute">
          Monitor and control your containerized ecosystem
        </p>
      </div>
      <div class="header-actions">
        <div class="search-box glass">
          <svg
            viewBox="0 0 24 24"
            width="18"
            height="18"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <circle cx="11" cy="11" r="8"></circle>
            <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
          </svg>
          <input
            type="text"
            v-model="sharedState.searchQuery"
            placeholder="Filter by name or image..."
          />
        </div>
        <button
          class="premium-btn primary refresh-trigger"
          @click="fetchContainers"
          :disabled="loading"
        >
          <svg
            viewBox="0 0 24 24"
            width="16"
            height="16"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            :class="{ rotating: loading }"
          >
            <polyline points="23 4 23 10 17 10"></polyline>
            <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
          </svg>
          Refresh
        </button>
      </div>
    </div>

    <ContainerTable class="mt-8" />
  </div>
</template>

<script setup>
import ContainerTable from "../components/ContainerTable.vue";
import { useContainers } from "../composables/useContainers";
import { sharedState } from "../utils/sharedState";

const { fetchContainers, loading } = useContainers();
</script>

<style scoped>
.view-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  margin-bottom: 2rem;
}

.header-actions {
  display: flex;
  gap: 1.5rem;
  align-items: center;
}

.rotating {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

button.premium-btn.primary.refresh-trigger {
  max-width: 120px;
}

@media (max-width: 1024px) {
  .view-header {
    flex-direction: column;
    align-items: stretch;
    gap: 1.25rem;
  }
  .header-actions {
    width: 100%;
    align-items: stretch;
    gap: 1rem;
  }
  .search-box {
    min-width: 0 !important;
    width: 100%;
  }
  .premium-btn {
    width: 100%;
    justify-content: center;
  }
}
</style>
