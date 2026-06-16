<template>
  <div class="pod-table" :class="{ embedded }">
    <div class="premium-table-container" :class="{ embedded }">
      <table class="premium-table pods-table">
        <thead>
          <tr>
            <th>Pod</th>
            <th>Namespace</th>
            <th>Ready</th>
            <th>Restarts</th>
            <th>Phase</th>
            <th class="text-right">Actions</th>
          </tr>
        </thead>
        <tbody v-if="loading">
          <tr>
            <td colspan="6">
              <div class="table-loading"><div class="shimmer"></div></div>
            </td>
          </tr>
        </tbody>
        <tbody v-else-if="visiblePods.length">
          <tr v-for="pod in visiblePods" :key="pod.uid || `${pod.namespace}/${pod.name}`">
            <td>
              <button class="pod-link" type="button" @click="goToPod(pod)">
                {{ pod.name }}
              </button>
              <div class="sub-text">{{ pod.status }}</div>
            </td>
            <td><span class="ns-pill">{{ pod.namespace }}</span></td>
            <td>{{ pod.ready }}</td>
            <td>{{ pod.restarts }}</td>
            <td>
              <span :class="['status-pill', phaseClass(pod.phase)]">{{ pod.phase }}</span>
            </td>
            <td class="text-right action-cell">
              <div class="action-group justify-end" @click.stop>
                <div class="action-cluster primary-actions">
                  <button
                    v-if="userCanStart(sharedState.currentUser) && !isPodRunning(pod)"
                    type="button"
                    class="icon-btn start"
                    data-tooltip="Start"
                    @click="runPodAction(pod, 'start')"
                  >
                    <AppIcon name="play" :size="16" :stroke-width="2.75" />
                  </button>
                  <button
                    v-if="userCanStop(sharedState.currentUser) && isPodRunning(pod)"
                    type="button"
                    class="icon-btn stop"
                    data-tooltip="Stop"
                    @click="runPodAction(pod, 'stop')"
                  >
                    <AppIcon name="stopOutline" :size="16" :stroke-width="2.75" />
                  </button>
                  <button
                    v-if="userCanRestart(sharedState.currentUser)"
                    type="button"
                    class="icon-btn restart"
                    data-tooltip="Restart"
                    @click="runPodAction(pod, 'restart')"
                  >
                    <AppIcon name="refresh" :size="16" :stroke-width="2.75" />
                  </button>
                </div>
                <div class="action-cluster secondary-actions">
                  <button
                    v-if="userCanShell(sharedState.currentUser)"
                    type="button"
                    class="icon-btn shell"
                    data-tooltip="Shell"
                    @click="goToShell(pod)"
                  >
                    <AppIcon name="terminal" :size="16" />
                  </button>
                  <button
                    type="button"
                    class="icon-btn logs"
                    data-tooltip="Logs"
                    @click="goToLogs(pod)"
                  >
                    <AppIcon name="logsBubble" :size="16" />
                  </button>
                  <button
                    v-if="userCanDelete(sharedState.currentUser)"
                    type="button"
                    class="icon-btn delete"
                    data-tooltip="Delete"
                    @click="runPodAction(pod, 'remove')"
                  >
                    <AppIcon name="trash" :size="16" />
                  </button>
                </div>
              </div>
            </td>
          </tr>
        </tbody>
        <tbody v-else>
          <tr>
            <td colspan="6" class="empty-cell">No pods found in this namespace.</td>
          </tr>
        </tbody>
      </table>
      <div v-if="!loading && totalCount" class="table-footer">
        <span>
          Showing {{ visiblePods.length }} of {{ totalCount }}
          pod{{ totalCount === 1 ? '' : 's' }}
        </span>
        <router-link
          v-if="showViewAllLink"
          :to="viewAllTo"
          class="view-all-link"
        >
          {{ viewAllLabel }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import AppIcon from './AppIcon.vue';
import { logsRouteForPod } from '../utils/logRoutes';
import { apiFetch } from '../utils/apiFetch';
import { secureStorage } from '../utils/storage';
import {
  sharedState,
  showToast,
  userCanStart,
  userCanStop,
  userCanRestart,
  userCanDelete,
  userCanShell,
} from '../utils/sharedState';

const props = defineProps({
  pods: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  embedded: { type: Boolean, default: false },
  maxRows: { type: Number, default: 0 },
  viewAllTo: { type: [String, Object], default: '' },
  viewAllLabel: { type: String, default: 'View all pods' },
});

const emit = defineEmits(['refresh']);

const router = useRouter();

const totalCount = computed(() => props.pods.length);

const visiblePods = computed(() => {
  if (!props.maxRows || props.maxRows <= 0) {
    return props.pods;
  }
  return props.pods.slice(0, props.maxRows);
});

const showViewAllLink = computed(
  () => props.maxRows > 0 && totalCount.value > props.maxRows && !!props.viewAllTo,
);

const phaseClass = (phase) => {
  const normalized = String(phase || '').toLowerCase();
  if (normalized === 'running') return 'is-running';
  if (normalized === 'pending') return 'is-pending';
  if (normalized === 'failed' || normalized === 'unknown') return 'is-failed';
  return 'is-neutral';
};

const isPodRunning = (pod) => String(pod.phase || '').toLowerCase() === 'running';

const goToPod = (pod) => {
  router.push({
    path: `/kubernetes/pods/${encodeURIComponent(pod.namespace)}/${encodeURIComponent(pod.name)}`,
    query: { namespace: pod.namespace, tab: 'pods' },
  });
};

const goToLogs = (pod) => {
  router.push(logsRouteForPod(pod));
};

const goToShell = (pod) => {
  router.push({ path: '/shell', query: { p: `${pod.namespace}/${pod.name}` } });
};

const runPodAction = async (pod, action) => {
  const confirmed = window.confirm(`Are you sure you want to ${action} pod ${pod.name}?`);
  if (!confirmed) return;

  try {
    const token = secureStorage.getItem('token');
    const formData = new URLSearchParams();
    formData.set('action', action);
    const res = await apiFetch(
      `/api/namespaces/${encodeURIComponent(pod.namespace)}/pods/${encodeURIComponent(pod.name)}/action`,
      {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: formData.toString(),
      },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => ({}));
      throw new Error(data.error || `Failed to ${action} pod`);
    }
    showToast('Success', `Pod ${action} requested.`, 'success');
    emit('refresh');
  } catch (err) {
    showToast('Action failed', err.message || `Failed to ${action} pod`, 'error');
  }
};
</script>

<style scoped>
.pod-link {
  font-weight: 800;
  color: var(--text-main);
  text-align: left;
  padding: 0;
  background: none;
  border: none;
}

.pod-link:hover {
  color: var(--accent);
}

.sub-text {
  font-size: 0.72rem;
  color: var(--text-mute);
  margin-top: 0.15rem;
}

.ns-pill {
  display: inline-flex;
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
  background: var(--bg-subtle);
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--text-dim);
}

.status-pill {
  display: inline-flex;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 800;
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.status-pill.is-running { color: var(--success); }
.status-pill.is-pending { color: var(--warning); }
.status-pill.is-failed { color: var(--error); }
.status-pill.is-neutral { color: var(--text-mute); }

.empty-cell {
  text-align: center;
  color: var(--text-mute);
  padding: 2rem 1rem !important;
}

.action-cell {
  min-width: 330px;
}

.action-group {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.45rem;
}

.justify-end {
  justify-content: flex-end;
}

.action-cluster {
  display: flex;
  gap: 0.35rem;
  padding: 0.25rem;
  border-radius: var(--radius-md);
  background: var(--bg-subtle);
  border: 1px solid var(--border-subtle);
}

.secondary-actions {
  background: transparent;
  border-color: transparent;
  padding: 0.25rem 0;
}

.icon-btn {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  border: 1px solid transparent;
  background: var(--bg-input);
  color: var(--text-dim);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.icon-btn:hover {
  transform: translateY(-1px);
  color: var(--text-main);
  border-color: var(--border);
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.08);
}

.icon-btn.start:hover {
  color: var(--success);
  border-color: rgba(var(--success-rgb), 0.35);
  background: rgba(var(--success-rgb), 0.08);
}

.icon-btn.stop:hover {
  color: var(--warning);
  border-color: rgba(var(--warning-rgb), 0.35);
  background: rgba(var(--warning-rgb), 0.08);
}

.icon-btn.restart:hover,
.icon-btn.logs:hover {
  color: var(--accent);
  border-color: rgba(var(--accent-rgb), 0.35);
  background: var(--accent-soft);
}

.icon-btn.shell:hover,
.icon-btn.shell:focus-visible {
  color: #8b5cf6;
  border-color: rgba(139, 92, 246, 0.4);
  background: rgba(139, 92, 246, 0.1);
  box-shadow: 0 4px 14px rgba(139, 92, 246, 0.15);
}

.icon-btn.delete:hover {
  color: var(--error);
  border-color: rgba(var(--error-rgb), 0.35);
  background: rgba(var(--error-rgb), 0.08);
}

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
  padding: 0.75rem 1.25rem;
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--text-mute);
  border-top: 1px solid var(--border);
  background: var(--bg-subtle);
}

.view-all-link {
  color: var(--accent);
  font-weight: 700;
  text-decoration: none;
}

.view-all-link:hover {
  text-decoration: underline;
}
</style>
