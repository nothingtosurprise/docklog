<template>
  <div class="container-table">
    <div class="premium-table-container">
      <table class="premium-table">
        <thead>
          <tr>
            <th>Container Name</th>
            <th>Image & Tag</th>
            <th>Created</th>
            <th>Uptime</th>
            <th>State</th>
            <th class="text-right">Control</th>
          </tr>
        </thead>
        <tbody v-if="loading">
          <tr>
            <td colspan="6">
              <div class="table-loading">
                <div class="shimmer"></div>
              </div>
            </td>
          </tr>
        </tbody>
        <tbody v-else-if="filteredContainers.length > 0">
          <tr
            v-for="c in filteredContainers"
            :key="c.id"
            class="container-row"
          >
            <td data-label="Container Name">
              <div
                class="name-cell clickable group"
                @click="goToLogs(c.id)"
                @mouseenter="startLiveStats(c.id)"
                @mouseleave="stopLiveStats"
              >
                <div class="name-main">
                  <span class="container-title">{{ c.name }}</span>
                  <span class="container-id">{{ c.id.substring(0, 12) }}</span>
                </div>

                <!-- Stats Peek Hover (Live when hovering) -->
                <div v-if="c.state === 'running'" class="row-stats-peek">
                  <div class="r-stat">
                    <span
                      class="r-val"
                      :class="{ 'text-live': activeLiveId === c.id }"
                    >
                      {{
                        (activeLiveId === c.id ? liveStats.cpu : c.cpu)?.toFixed(2) || "0.00"
                      }}%
                    </span>
                  </div>
                  <div class="r-stat">
                    <span
                      class="r-val"
                      :class="{ 'text-live': activeLiveId === c.id }"
                    >
                      {{
                        formatBytes(activeLiveId === c.id ? liveStats.memory : c.memory)
                      }}
                    </span>
                  </div>
                </div>
              </div>
            </td>
            <td data-label="Image & Tag">
              <div class="image-cell">
                <span class="image-name">{{ c.image.split(":")[0] }}</span>
                <span class="image-tag">{{
                  c.image.split(":")[1] || "latest"
                }}</span>
              </div>
            </td>
            <td data-label="Created">
              <span class="date-label">{{ formatDate(c.created) }}</span>
            </td>
            <td data-label="Uptime">
              <span
                :class="[
                  'uptime-label',
                  c.state === 'running' ? 'is-running' : 'is-stopped',
                ]"
              >
                {{ c.status }}
              </span>
            </td>
            <td data-label="State">
              <div
                :class="[
                  'status-pill',
                  c.state === 'running' ? 'is-running' : 'is-stopped',
                ]"
              >
                <span class="pulse-dot"></span>
                {{ c.state.toUpperCase() }}
              </div>
            </td>
            <td class="text-right" data-label="Actions">
              <div class="action-group justify-end">
                <button
                  v-if="(sharedState.currentUser?.is_admin || sharedState.currentUser?.can_start) && c.state !== 'running'"
                  @click="triggerConfirm(c.id, 'start')"
                  class="icon-btn start"
                  data-tooltip="Start Container"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                  >
                    <polygon points="5 3 19 12 5 21 5 3"></polygon>
                  </svg>
                </button>
                <button
                  v-if="(sharedState.currentUser?.is_admin || sharedState.currentUser?.can_stop) && c.state === 'running'"
                  @click="triggerConfirm(c.id, 'stop')"
                  class="icon-btn stop"
                  data-tooltip="Stop Container"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    fill="currentColor"
                    stroke="currentColor"
                    stroke-width="3"
                  >
                    <rect x="6" y="6" width="12" height="12"></rect>
                  </svg>
                </button>
                <button
                  v-if="sharedState.currentUser?.is_admin || sharedState.currentUser?.can_restart"
                  @click="triggerConfirm(c.id, 'restart')"
                  class="icon-btn restart"
                  data-tooltip="Restart Container"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                  >
                    <path d="M23 4v6h-6"></path>
                    <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
                  </svg>
                </button>
                <button
                  @click="goToLogs(c.id)"
                  class="icon-btn logs"
                  data-tooltip="View Live Logs"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                  >
                    <path
                      d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"
                    ></path>
                    <path d="M8 9h8M8 13h5"></path>
                  </svg>
                </button>
                <button
                  v-if="sharedState.currentUser?.is_admin || sharedState.currentUser?.can_delete"
                  @click="triggerConfirm(c.id, 'remove')"
                  class="icon-btn stop"
                  data-tooltip="Delete Container"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="3"
                  >
                    <polyline points="3 6 5 6 21 6"></polyline>
                    <path
                      d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                    ></path>
                  </svg>
                </button>
              </div>
            </td>
          </tr>
        </tbody>
        <tbody v-else>
          <tr>
            <td colspan="6">
              <div class="empty-state-wrapper">
                <div class="empty-state-content">
                  <div class="empty-icon-box">
                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
                      <line x1="8" y1="21" x2="16" y2="21"></line>
                      <line x1="12" y1="17" x2="12" y2="21"></line>
                    </svg>
                  </div>
                  <h4 class="empty-title">No Containers Found</h4>
                  <p class="empty-text">
                    No containers match your search or you may not have access to any yet.
                  </p>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Unified Action Confirmation Modal -->
    <Teleport to="body">
      <Transition name="fade">
        <div v-if="showConfirm" class="modal-overlay">
          <div class="modal-content shadow-2xl">
            <div :class="['modal-icon', actionClass]">
              <svg v-if="pendingAction === 'start'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
              <svg v-else-if="pendingAction === 'stop'" viewBox="0 0 24 24" fill="currentColor" stroke="currentColor" stroke-width="2.5"><rect x="6" y="6" width="12" height="12"></rect></svg>
              <svg v-else-if="pendingAction === 'restart'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><path d="M23 4v6h-6"></path><path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path></svg>
              <svg v-else-if="pendingAction === 'remove'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                <polyline points="3 6 5 6 21 6"></polyline>
                <path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path>
              </svg>
            </div>
            <div class="modal-text-center">
              <h3>Confirm Operation</h3>
              <p>Are you sure you want to <strong>{{ pendingAction }}</strong> this container? This may affect active services.</p>
            </div>
            <div class="modal-divider"></div>
            <div class="modal-actions">
              <button @click="showConfirm = false" class="modal-btn cancel">Cancel</button>
              <button @click="executeAction" :class="['modal-btn confirm', actionClass]">Confirm {{ pendingAction }}</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { useContainers } from '../composables/useContainers';
import { sharedState } from '../utils/sharedState';

const {
  loading, filteredContainers, activeLiveId, liveStats,
  showConfirm, pendingAction, actionClass,
  startLiveStats, stopLiveStats, goToLogs, triggerConfirm, executeAction,
  formatBytes, formatDate,
} = useContainers();
</script>

<style scoped>
.premium-table tr {
  cursor: pointer;
  transition: all 0.2s;
}

.premium-table tr.active td {
  background: var(--accent-soft);
  color: var(--accent);
}

.action-group {
  display: flex;
  gap: 0.5rem;
}

.icon-btn {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.icon-btn:hover {
  background: var(--bg-card);
  transform: translateY(-2px);
  color: var(--text-main);
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
}

.icon-btn.start:hover { color: var(--success); border-color: var(--success); box-shadow: 0 4px 12px rgba(var(--success-rgb), 0.2); }
.icon-btn.stop:hover { color: var(--error); border-color: var(--error); box-shadow: 0 4px 12px rgba(var(--error-rgb), 0.2); }
.icon-btn.restart:hover { color: var(--warning); border-color: var(--warning); box-shadow: 0 4px 12px rgba(var(--warning-rgb), 0.2); }
.icon-btn.logs:hover {
  color: var(--accent);
  border-color: var(--accent);
  box-shadow: 0 4px 12px rgba(var(--accent-rgb), 0.2);
}

/* Row Stats Peek */
.row-stats-peek {
  display: flex;
  gap: 1rem;
  opacity: 0;
  transform: translateX(-10px);
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  background: var(--bg-card);
  padding: 0.4rem 0.8rem;
  border-radius: 10px;
  border: 1px solid var(--border);
  pointer-events: none;
  margin-left: 1rem;
}

.name-cell:hover .row-stats-peek {
  opacity: 1;
  transform: translateX(0);
}

.r-stat {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.r-val {
  font-size: 0.75rem;
  font-weight: 900;
  color: var(--accent);
  font-family: var(--font-mono);
  transition: color 0.3s;
}

.text-live {
  color: var(--success) !important;
  text-shadow: 0 0 8px rgba(var(--success-rgb), 0.4);
}

.r-lab {
  font-size: 0.55rem;
  font-weight: 800;
  color: var(--text-mute);
  text-transform: uppercase;
}

.name-cell {
  display: flex;
  align-items: center;
  min-width: 0;
}

.name-main {
  display: flex;
  flex-direction: column;
}

.image-cell {
  display: flex;
  flex-direction: column;
  text-align: left;
  min-width: 0;
}

.status-pill {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.7rem;
  font-weight: 950;
  padding: 0.4rem 0.8rem;
  border-radius: 10px;
  width: fit-content;
  border: 1px solid rgba(var(--accent-rgb), 0.2);
  background: rgba(var(--accent-rgb), 0.05);
  color: var(--accent);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.is-running {
  color: var(--success) !important;
}
.is-running .pulse-dot {
  background: var(--success);
  box-shadow: 0 0 8px var(--success);
  animation: pulse-mini 2s infinite;
}

.is-stopped {
  color: var(--error) !important;
}
.is-stopped .pulse-dot {
  background: var(--error);
}

@keyframes pulse-mini {
  0% { transform: scale(0.95); opacity: 1; }
  50% { transform: scale(1.1); opacity: 0.7; }
  100% { transform: scale(0.95); opacity: 1; }
}

@media (max-width: 850px) {
  .premium-table thead {
    display: none;
  }
  .premium-table tbody tr {
    display: block;
    padding: 1.25rem;
    margin-bottom: 1.25rem;
    background: var(--bg-card);
    border: 1px solid var(--border);
    border-radius: 20px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
  }
  .premium-table tbody tr td {
    display: flex;
    flex-direction: column;
    padding: 0.6rem 0;
    border: none;
    text-align: left !important;
    gap: 0.35rem;
  }
  .premium-table tbody tr td::before {
    content: attr(data-label);
    display: block;
    font-size: 0.65rem;
    font-weight: 800;
    color: var(--text-mute);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }
}

.table-loading {
  padding: 2rem 1rem;
}

.table-loading .shimmer {
  min-height: 180px;
  border-radius: var(--radius-lg);
}

.empty-state-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 280px;
}

.empty-state-content {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.empty-icon-box {
  width: 72px;
  height: 72px;
  background: var(--bg-input);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-mute);
  opacity: 0.6;
  margin-bottom: 1.25rem;
  border: 1px dashed var(--border);
}

.empty-title {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--text-main);
  margin-bottom: 0.5rem;
}

.empty-text {
  font-size: 0.9rem;
  color: var(--text-mute);
  max-width: 320px;
  line-height: 1.6;
}
</style>
