<template>
  <div class="page-view alerts-view animate-fade-in">
    <section class="page-hero">
      <div class="page-hero-body">
        <div class="page-hero-copy">
          <span class="page-hero-eyebrow">Administration</span>
          <div class="page-hero-title-row">
            <span class="page-hero-mark">
              <BrandIcon name="notifications" :size="26" :colored="false" />
            </span>
            <h1>Alerts</h1>
          </div>
          <p class="page-hero-sub">
            Rule-based monitoring from logs, Docker events, Kubernetes warning events, and metrics. Routes to your configured notification channels.
          </p>
        </div>
        <div class="page-hero-stats">
          <div class="page-hero-stat success">
            <span class="page-hero-stat-val">{{ enabledCount }}</span>
            <span class="page-hero-stat-lbl">Enabled</span>
          </div>
          <div class="page-hero-stat">
            <span class="page-hero-stat-val">{{ rules.length }}</span>
            <span class="page-hero-stat-lbl">Rules</span>
          </div>
          <div class="page-hero-stat" :class="channels.length ? 'success' : 'warning'">
            <span class="page-hero-stat-val">{{ channels.length }}</span>
            <span class="page-hero-stat-lbl">Channels</span>
          </div>
        </div>
      </div>
      <div class="page-hero-mesh" aria-hidden="true"></div>
    </section>

    <div v-if="loading" class="loading-state">Loading alert rules...</div>

    <div v-else class="alerts-body">
      <section class="page-panel flush">
        <div class="page-toolbar alerts-toolbar">
          <div class="page-filter-pills">
            <button
              type="button"
              class="page-filter-pill"
              :class="{ active: tab === 'rules' }"
              @click="tab = 'rules'"
            >
              Rules
              <span class="pill-count">{{ rules.length }}</span>
            </button>
            <button
              type="button"
              class="page-filter-pill"
              :class="{ active: tab === 'history' }"
              @click="openHistory"
            >
              History
              <span v-if="historyLoaded" class="pill-count">{{ history.length }}</span>
            </button>
          </div>
          <button type="button" class="page-btn primary" @click="openCreate">
            <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5">
              <line x1="12" y1="5" x2="12" y2="19" />
              <line x1="5" y1="12" x2="19" y2="12" />
            </svg>
            New rule
          </button>
        </div>
      </section>

      <section v-if="tab === 'rules'" class="alerts-rules">
        <section class="page-panel flush rules-panel">
          <div class="section-head compact">
            <div>
              <h2>Alert rules</h2>
              <p class="section-sub">Toggle rules on or off without opening the editor.</p>
            </div>
          </div>

          <div v-if="!rules.length" class="empty-state-wrap">
            <div class="empty-state-content">
              <div class="empty-icon">
                <BrandIcon name="notifications" :size="28" :colored="false" />
              </div>
              <h3>No alert rules yet</h3>
              <p>Create a rule to start monitoring containers.</p>
              <button type="button" class="page-btn primary" @click="openCreate">Create rule</button>
            </div>
          </div>

          <div v-else class="premium-table-container embedded">
            <table class="premium-table alerts-table">
              <thead>
                <tr>
                  <th>Rule</th>
                  <th>Source</th>
                  <th>Severity</th>
                  <th>Destinations</th>
                  <th>Status</th>
                  <th class="col-actions">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="rule in rules" :key="rule.id">
                  <td data-label="Rule">
                    <div class="rule-name-cell">
                      <strong class="rule-name">{{ rule.name }}</strong>
                      <code class="rule-id">{{ rule.rule_id }}</code>
                    </div>
                  </td>
                  <td data-label="Source">
                    <span :class="['source-badge', sourceClass(rule.source_type)]">
                      {{ formatSourceType(rule.source_type) }}
                    </span>
                  </td>
                  <td data-label="Severity">
                    <span :class="['severity-badge', rule.severity]">{{ rule.severity }}</span>
                  </td>
                  <td data-label="Destinations">
                    <span class="dest-count" :class="{ muted: !rule.channel_ids?.length, warn: !rule.channel_ids?.length }">
                      {{ rule.channel_ids?.length || 0 }}
                      <span class="dest-label">
                        {{ rule.channel_ids?.length ? `channel${rule.channel_ids.length === 1 ? '' : 's'}` : 'auto on enable' }}
                      </span>
                    </span>
                  </td>
                  <td data-label="Status">
                    <div
                      class="premium-toggle"
                      :class="{ active: rule.enabled }"
                      role="switch"
                      :aria-checked="rule.enabled"
                      tabindex="0"
                      @click="toggleRule(rule)"
                      @keydown.enter.space.prevent="toggleRule(rule)"
                    >
                      <div class="toggle-rail">
                        <div class="toggle-handle"></div>
                      </div>
                      <span class="status-label">{{ rule.enabled ? 'On' : 'Off' }}</span>
                    </div>
                  </td>
                  <td class="col-actions" data-label="Actions">
                    <div class="action-group">
                      <button
                        type="button"
                        class="action-link"
                        :class="{ 'is-busy': testingRuleId === rule.id }"
                        :disabled="testingRuleId === rule.id || deletingRuleId === rule.id"
                        @click="testRule(rule)"
                      >
                        <span v-if="testingRuleId === rule.id" class="action-spinner" aria-hidden="true"></span>
                        {{ testingRuleId === rule.id ? 'Sending…' : 'Test' }}
                      </button>
                      <button
                        type="button"
                        class="action-link"
                        :class="{ 'is-active': editorOpen && editingId === rule.id }"
                        :disabled="testingRuleId === rule.id || deletingRuleId === rule.id"
                        @click="editRule(rule)"
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        class="action-link danger"
                        :class="{ 'is-busy': deletingRuleId === rule.id }"
                        :disabled="testingRuleId === rule.id || deletingRuleId === rule.id"
                        @click="removeRule(rule)"
                      >
                        <span v-if="deletingRuleId === rule.id" class="action-spinner" aria-hidden="true"></span>
                        {{ deletingRuleId === rule.id ? 'Deleting…' : 'Delete' }}
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </section>
      </section>

      <section v-else class="alerts-history page-panel flush">
        <div v-if="historyLoading" class="loading-state">Loading alert history...</div>
        <div v-else-if="!history.length" class="empty-state-wrap">
          <div class="empty-state-content">
            <div class="empty-icon">
              <BrandIcon name="notifications" :size="28" :colored="false" />
            </div>
            <h3>No alerts fired yet</h3>
            <p>When a rule triggers, the event will appear here with delivery status.</p>
          </div>
        </div>
        <div v-else class="history-list">
          <article v-for="item in history" :key="item.id" class="history-card">
            <div class="history-card-head">
              <span :class="['severity-badge', item.severity]">{{ item.severity }}</span>
              <div class="history-title-wrap">
                <strong>{{ item.rule_name || item.rule_id }}</strong>
                <span class="history-status" :class="item.status">{{ item.status }}</span>
              </div>
              <time class="history-time">{{ formatTimestamp(item.timestamp) }}</time>
            </div>
            <p class="history-message">{{ item.message }}</p>
            <div class="history-meta">
              <span v-if="item.container" class="meta-chip">
                <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2">
                  <rect x="2" y="7" width="20" height="14" rx="2" />
                  <path d="M16 7V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v2" />
                </svg>
                {{ item.container }}
              </span>
              <span class="meta-chip muted">{{ formatSourceType(item.source) }}</span>
            </div>
          </article>
        </div>
      </section>
    </div>

    <Teleport to="body">
      <Transition name="modal-bounce">
        <div v-if="editorOpen" class="modal-overlay editor-overlay" role="presentation" @click.self="closeEditor">
          <form class="alert-editor modal-card glass shadow-2xl" @submit.prevent="saveRule">
            <div class="modal-card-header">
              <div class="header-content">
                <div class="header-icon">
                  <BrandIcon name="notifications" :size="22" :colored="false" />
                </div>
                <div class="header-copy">
                  <div class="header-title-row">
                    <h3 class="modal-title">{{ editingId ? 'Edit alert rule' : 'Create alert rule' }}</h3>
                    <div
                      v-if="editingId"
                      class="premium-toggle header-toggle"
                      :class="{ active: form.enabled }"
                      role="switch"
                      :aria-checked="form.enabled"
                      tabindex="0"
                      @click.stop="form.enabled = !form.enabled"
                      @keydown.enter.space.prevent="form.enabled = !form.enabled"
                    >
                      <div class="toggle-rail">
                        <div class="toggle-handle"></div>
                      </div>
                      <span class="status-label">{{ form.enabled ? 'Enabled' : 'Disabled' }}</span>
                    </div>
                  </div>
                  <p class="modal-subtitle">
                    {{ editingId ? 'Update matching logic, scope, and destinations.' : 'Define what to watch and where to send alerts.' }}
                  </p>
                  <div v-if="editingId && form.rule_id" class="header-chips">
                    <code class="rule-id-chip">{{ form.rule_id }}</code>
                    <span :class="['severity-badge', form.severity]">{{ form.severity }}</span>
                    <span :class="['source-badge', sourceClass(form.source_type)]">
                      {{ formatSourceType(form.source_type) }}
                    </span>
                  </div>
                </div>
              </div>
              <button type="button" class="close-btn" aria-label="Close" @click="closeEditor">
                <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5">
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </button>
            </div>

            <div class="editor-shell">
              <nav class="editor-nav" aria-label="Rule sections">
                <button
                  v-for="section in editorSections"
                  :key="section.id"
                  type="button"
                  class="editor-nav-btn"
                  :class="{ active: activeSection === section.id }"
                  @click="scrollToSection(section.id)"
                >
                  <span class="nav-step">{{ section.step }}</span>
                  <span class="nav-copy">
                    <strong>{{ section.label }}</strong>
                    <span>{{ section.hint }}</span>
                  </span>
                </button>
              </nav>

              <div ref="editorBodyRef" class="modal-card-body editor-body" @scroll="syncActiveSection">
                <section id="section-basics" class="editor-section">
                  <div class="section-head">
                    <h4>Basics</h4>
                    <p>Name and severity for this rule.</p>
                  </div>
                  <div class="form-grid dual">
                    <div class="input-group">
                      <label class="label-caps" for="rule-id">Rule ID</label>
                      <input
                        id="rule-id"
                        v-model="form.rule_id"
                        class="premium-input"
                        required
                        pattern="[a-z0-9-]+"
                        placeholder="high-cpu"
                        :disabled="!!editingId"
                        :class="{ readonly: !!editingId }"
                      />
                      <p v-if="editingId" class="field-hint">Rule ID cannot be changed after creation.</p>
                    </div>
                    <div class="input-group">
                      <label class="label-caps" for="rule-name">Display name</label>
                      <input id="rule-name" v-model="form.name" class="premium-input" required placeholder="High CPU usage" />
                    </div>
                  </div>
                  <div class="input-group">
                    <label class="label-caps" for="rule-desc">Description</label>
                    <textarea id="rule-desc" v-model="form.description" class="premium-input textarea" rows="2" placeholder="Optional context for operators" />
                  </div>
                  <div class="input-group">
                    <label class="label-caps">Severity</label>
                    <div class="choice-row severity-row">
                      <button
                        v-for="option in severityOptions"
                        :key="option.value"
                        type="button"
                        class="choice-chip"
                        :class="[option.value, { active: form.severity === option.value }]"
                        @click="form.severity = option.value"
                      >
                        <strong>{{ option.label }}</strong>
                        <span>{{ option.hint }}</span>
                      </button>
                    </div>
                  </div>
                </section>

                <section id="section-trigger" class="editor-section">
                  <div class="section-head">
                    <h4>Trigger</h4>
                    <p>Choose what signal should fire this alert.</p>
                  </div>
                  <div class="input-group">
                    <label class="label-caps">Source type</label>
                    <div class="source-picker">
                      <button
                        v-for="option in sourceOptions"
                        :key="option.value"
                        type="button"
                        class="source-card"
                        :class="[sourceClass(option.value), { active: form.source_type === option.value }]"
                        @click="setSourceType(option.value)"
                      >
                        <span class="source-card-label">{{ option.label }}</span>
                        <span class="source-card-hint">{{ option.hint }}</span>
                      </button>
                    </div>
                  </div>

                  <div v-if="form.source_type === 'logs'" class="trigger-panel">
                    <div class="form-grid dual">
                      <div class="input-group">
                        <label class="label-caps">Log pattern</label>
                        <input v-model="logConfig.pattern" class="premium-input mono" placeholder="ERROR" />
                      </div>
                      <div class="input-group">
                        <label class="label-caps">Matches required</label>
                        <input v-model.number="logConfig.match_count" class="premium-input" type="number" min="1" />
                      </div>
                    </div>
                    <div class="form-grid dual align-end">
                      <div class="input-group">
                        <label class="label-caps">Window (seconds)</label>
                        <input v-model.number="logConfig.window_seconds" class="premium-input" type="number" min="1" />
                      </div>
                      <div
                        class="premium-toggle panel-toggle"
                        :class="{ active: logConfig.case_sensitive }"
                        role="switch"
                        :aria-checked="logConfig.case_sensitive"
                        tabindex="0"
                        @click="logConfig.case_sensitive = !logConfig.case_sensitive"
                        @keydown.enter.space.prevent="logConfig.case_sensitive = !logConfig.case_sensitive"
                      >
                        <div class="toggle-rail">
                          <div class="toggle-handle"></div>
                        </div>
                        <span class="toggle-copy">
                          <strong>Case sensitive</strong>
                          <span>Match pattern with exact casing</span>
                        </span>
                      </div>
                    </div>
                  </div>

                  <div v-else-if="form.source_type === 'events'" class="trigger-panel">
                    <div class="panel-toolbar">
                      <span class="label-caps">Docker events</span>
                      <button type="button" class="text-link" @click="selectAllEvents">Select all</button>
                    </div>
                    <div class="event-grid">
                      <label
                        v-for="event in eventOptions"
                        :key="event"
                        class="event-card"
                        :class="{ active: eventConfig.events.includes(event) }"
                      >
                        <input type="checkbox" :value="event" v-model="eventConfig.events" />
                        <span class="event-card-title">{{ eventOptionMeta[event].label }}</span>
                        <span class="event-card-desc">{{ eventOptionMeta[event].desc }}</span>
                      </label>
                    </div>
                    <div class="input-group">
                      <label class="label-caps">Min occurrences</label>
                      <input v-model.number="eventConfig.min_occurrences" class="premium-input" type="number" min="1" />
                    </div>
                  </div>

                  <div v-else-if="form.source_type === 'k8s_events'" class="trigger-panel">
                    <div class="panel-toolbar">
                      <span class="label-caps">Kubernetes warning events</span>
                      <button type="button" class="text-link" @click="selectAllK8sEvents">Select all</button>
                    </div>
                    <div class="event-grid">
                      <label
                        v-for="event in k8sEventOptions"
                        :key="event"
                        class="event-card"
                        :class="{ active: k8sEventConfig.events.includes(event) }"
                      >
                        <input type="checkbox" :value="event" v-model="k8sEventConfig.events" />
                        <span class="event-card-title">{{ k8sEventOptionMeta[event].label }}</span>
                        <span class="event-card-desc">{{ k8sEventOptionMeta[event].desc }}</span>
                      </label>
                    </div>
                    <div class="input-group">
                      <label class="label-caps">Min occurrences</label>
                      <input v-model.number="k8sEventConfig.min_occurrences" class="premium-input" type="number" min="1" />
                    </div>
                  </div>

                  <div v-else-if="form.source_type === 'metrics'" class="trigger-panel">
                    <div class="form-grid triple">
                      <div class="input-group">
                        <label class="label-caps">Metric</label>
                        <select v-model="metricConfig.metric" class="premium-input">
                          <option value="cpu">CPU %</option>
                          <option value="memory">Memory %</option>
                        </select>
                      </div>
                      <div class="input-group">
                        <label class="label-caps">Threshold %</label>
                        <input v-model.number="metricConfig.threshold" class="premium-input" type="number" min="1" max="100" />
                      </div>
                      <div class="input-group">
                        <label class="label-caps">Duration (min)</label>
                        <input v-model.number="metricConfig.duration_minutes" class="premium-input" type="number" min="1" />
                      </div>
                    </div>
                  </div>
                </section>

                <section id="section-scope" class="editor-section">
                  <div class="section-head">
                    <h4>Scope</h4>
                    <p>{{ isK8sSource ? 'Limit which pods or namespaces this rule applies to.' : 'Limit which containers this rule applies to.' }}</p>
                  </div>
                  <div class="input-group">
                    <label class="label-caps">{{ isK8sSource ? 'Target resources' : 'Target containers' }}</label>
                    <div class="scope-picker">
                      <button
                        v-for="option in scopeOptions"
                        :key="option.value"
                        type="button"
                        class="scope-choice"
                        :class="{ active: form.scope.type === option.value }"
                        @click="form.scope.type = option.value"
                      >
                        {{ option.label }}
                      </button>
                    </div>
                  </div>
                  <div v-if="form.scope.type === 'names'" class="input-group">
                    <label class="label-caps">{{ isK8sSource ? 'Resource names' : 'Container names' }}</label>
                    <input v-model="scopeNames" class="premium-input" :placeholder="isK8sSource ? 'default/api, worker' : 'api, worker'" />
                    <p class="field-hint">{{ isK8sSource ? 'Comma-separated pod names or namespace/pod values' : 'Comma-separated exact container names' }}</p>
                  </div>
                  <div v-if="form.scope.type === 'patterns'" class="input-group">
                    <label class="label-caps">Name patterns</label>
                    <input v-model="scopePatterns" class="premium-input mono" :placeholder="isK8sSource ? 'default/*, *api*' : 'backend-*, *api*'" />
                    <p class="field-hint">{{ isK8sSource ? 'Comma-separated glob patterns against namespace/pod' : 'Comma-separated glob patterns' }}</p>
                  </div>
                </section>

                <section id="section-destinations" class="editor-section">
                  <div class="section-head">
                    <h4>Destinations</h4>
                    <p>Notification channels to deliver fired alerts.</p>
                  </div>
                  <p v-if="!channels.length" class="hint warn channel-warn">
                    No active notification channels. Configure channels on the Notifications page first.
                  </p>
                  <div v-else class="channel-grid">
                    <label
                      v-for="channel in channels"
                      :key="channel.id"
                      class="channel-card"
                      :class="{ active: form.channel_ids.includes(channel.id) }"
                    >
                      <input type="checkbox" :value="channel.id" v-model="form.channel_ids" />
                      <span class="channel-card-icon" :class="channel.type">
                        <ChannelIcon :type="channel.type" :size="20" />
                      </span>
                      <span class="channel-card-copy">
                        <strong>{{ channel.name }}</strong>
                        <span>{{ channel.type }}</span>
                      </span>
                    </label>
                  </div>
                </section>

                <section id="section-throttle" class="editor-section">
                  <div class="section-head">
                    <h4>Throttling</h4>
                    <p>Reduce noise from repeated or grouped alerts.</p>
                  </div>
                  <div class="form-grid triple">
                    <div class="input-group">
                      <label class="label-caps">Cooldown (min)</label>
                      <input v-model.number="form.cooldown_minutes" class="premium-input" type="number" min="0" />
                    </div>
                    <div class="input-group">
                      <label class="label-caps">Max per hour</label>
                      <input v-model.number="form.max_per_hour" class="premium-input" type="number" min="0" />
                    </div>
                    <div class="input-group">
                      <label class="label-caps">Group window (min)</label>
                      <input v-model.number="form.group_window_minutes" class="premium-input" type="number" min="0" />
                    </div>
                  </div>
                  <div
                    class="premium-toggle panel-toggle recovery-toggle"
                    :class="{ active: form.recovery_enabled }"
                    role="switch"
                    :aria-checked="form.recovery_enabled"
                    tabindex="0"
                    @click="form.recovery_enabled = !form.recovery_enabled"
                    @keydown.enter.space.prevent="form.recovery_enabled = !form.recovery_enabled"
                  >
                    <div class="toggle-rail">
                      <div class="toggle-handle"></div>
                    </div>
                    <span class="toggle-copy">
                      <strong>Recovery notifications</strong>
                      <span>Send a follow-up when the condition clears</span>
                    </span>
                  </div>
                </section>
              </div>
            </div>

            <div class="modal-card-footer">
              <button type="button" class="page-btn" @click="closeEditor">Cancel</button>
              <button type="submit" class="page-btn primary" :disabled="saving">
                {{ saving ? 'Saving...' : editingId ? 'Save changes' : 'Create rule' }}
              </button>
            </div>
          </form>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref } from 'vue';
import BrandIcon from '../components/BrandIcon.vue';
import ChannelIcon from '../components/ChannelIcon.vue';
import { showToast, kubernetesEnabled } from '../utils/sharedState';
import {
  createAlertRule,
  deleteAlertRule,
  fetchAlertHistory,
  fetchAlerts,
  fetchNotifications,
  testAlertRule,
  updateAlertRule,
} from '../services/alertService';

const loading = ref(true);
const historyLoading = ref(false);
const historyLoaded = ref(false);
const tab = ref('rules');
const rules = ref([]);
const history = ref([]);
const channels = ref([]);
const editorOpen = ref(false);
const editingId = ref(0);
const saving = ref(false);
const testingRuleId = ref(0);
const deletingRuleId = ref(0);
const activeSection = ref('basics');
const editorBodyRef = ref(null);

const eventOptions = ['start', 'stop', 'restart', 'exit_nonzero', 'unhealthy', 'oom'];

const k8sEventOptions = [
  'crash_loop_backoff',
  'image_pull_backoff',
  'failed_scheduling',
  'oom_killed',
  'backoff',
  'failed',
  'failed_mount',
  'evicted',
  'unhealthy',
];

const severityOptions = [
  { value: 'info', label: 'Info', hint: 'Awareness only' },
  { value: 'warning', label: 'Warning', hint: 'Needs attention' },
  { value: 'critical', label: 'Critical', hint: 'Immediate action' },
];

const baseSourceOptions = [
  { value: 'logs', label: 'Logs', hint: 'Pattern matching in container output' },
  { value: 'events', label: 'Events', hint: 'Docker lifecycle and health signals' },
  { value: 'metrics', label: 'Metrics', hint: 'CPU or memory thresholds' },
];

const sourceOptions = computed(() => {
  const options = [...baseSourceOptions];
  if (kubernetesEnabled()) {
    options.splice(2, 0, {
      value: 'k8s_events',
      label: 'Kubernetes',
      hint: 'Warning events from pods and workloads',
    });
  }
  return options;
});

const scopeOptions = computed(() => {
  if (isK8sSource.value) {
    return [
      { value: 'all', label: 'All resources' },
      { value: 'names', label: 'By name' },
      { value: 'patterns', label: 'By pattern' },
    ];
  }
  return [
    { value: 'all', label: 'All containers' },
    { value: 'names', label: 'By name' },
    { value: 'patterns', label: 'By pattern' },
  ];
});

const eventOptionMeta = {
  start: { label: 'Start', desc: 'Container started' },
  stop: { label: 'Stop', desc: 'Container stopped' },
  restart: { label: 'Restart', desc: 'Container restarted' },
  exit_nonzero: { label: 'Exit nonzero', desc: 'Exited with a failure code' },
  unhealthy: { label: 'Unhealthy', desc: 'Health check failed' },
  oom: { label: 'OOM', desc: 'Killed for out-of-memory' },
};

const k8sEventOptionMeta = {
  crash_loop_backoff: { label: 'Crash loop', desc: 'CrashLoopBackOff' },
  image_pull_backoff: { label: 'Image pull', desc: 'ImagePullBackOff or ErrImagePull' },
  failed_scheduling: { label: 'Scheduling', desc: 'FailedScheduling' },
  oom_killed: { label: 'OOM killed', desc: 'Container OOMKilled' },
  backoff: { label: 'Backoff', desc: 'Generic BackOff restarts' },
  failed: { label: 'Failed', desc: 'Container or pod failed' },
  failed_mount: { label: 'Mount failed', desc: 'Volume mount failed' },
  evicted: { label: 'Evicted', desc: 'Pod evicted from node' },
  unhealthy: { label: 'Unhealthy', desc: 'Probe or lifecycle hook failure' },
};

const editorSections = computed(() => [
  { id: 'basics', step: '1', label: 'Basics', hint: 'Name and severity' },
  { id: 'trigger', step: '2', label: 'Trigger', hint: 'What fires the alert' },
  {
    id: 'scope',
    step: '3',
    label: 'Scope',
    hint: form.source_type === 'k8s_events' ? 'Which pods or namespaces' : 'Which containers',
  },
  { id: 'destinations', step: '4', label: 'Destinations', hint: 'Where to send' },
  { id: 'throttle', step: '5', label: 'Throttling', hint: 'Noise control' },
]);

const form = reactive({
  rule_id: '',
  name: '',
  description: '',
  severity: 'warning',
  enabled: true,
  source_type: 'logs',
  scope: { type: 'all', containers: [], patterns: [] },
  channel_ids: [],
  cooldown_minutes: 15,
  max_per_hour: 20,
  group_window_minutes: 5,
  recovery_enabled: false,
});

const logConfig = reactive({ pattern: 'ERROR', match_count: 10, window_seconds: 120, case_sensitive: false });
const eventConfig = reactive({ events: ['restart'], min_occurrences: 1, window_seconds: 300 });
const k8sEventConfig = reactive({ events: ['crash_loop_backoff'], min_occurrences: 1, window_seconds: 300 });
const metricConfig = reactive({ metric: 'cpu', operator: 'gt', threshold: 90, duration_minutes: 5 });
const scopeNames = ref('');
const scopePatterns = ref('');

const enabledCount = computed(() => rules.value.filter((rule) => rule.enabled).length);
const isK8sSource = computed(() => form.source_type === 'k8s_events');

function formatSourceType(source) {
  const map = {
    logs: 'Logs',
    events: 'Events',
    metrics: 'Metrics',
    k8s_events: 'Kubernetes',
    log: 'Logs',
    metric: 'Metrics',
    event: 'Events',
  };
  return map[source] || source || 'Unknown';
}

function sourceClass(source) {
  const key = (source || '').replace(/s$/, '');
  if (key === 'log') return 'logs';
  if (key === 'metric') return 'metrics';
  if (key === 'event') return 'events';
  if (source === 'k8s_events') return 'k8s_events';
  return source || 'logs';
}

function formatTimestamp(value) {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function resetConfig() {
  if (form.source_type === 'logs') {
    logConfig.pattern = 'ERROR';
  }
}

function setSourceType(value) {
  form.source_type = value;
  resetConfig();
}

function selectAllEvents() {
  eventConfig.events = [...eventOptions];
}

function selectAllK8sEvents() {
  k8sEventConfig.events = [...k8sEventOptions];
}

function scrollToSection(id) {
  activeSection.value = id;
  const el = document.getElementById(`section-${id}`);
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' });
  }
}

function syncActiveSection() {
  const container = editorBodyRef.value;
  if (!container) return;
  const offset = container.scrollTop + 120;
  const sections = editorSections.value;
  for (let i = sections.length - 1; i >= 0; i -= 1) {
    const section = sections[i];
    const el = document.getElementById(`section-${section.id}`);
    if (el && el.offsetTop <= offset) {
      activeSection.value = section.id;
      break;
    }
  }
}

function buildScope() {
  const scope = { type: form.scope.type };
  if (form.scope.type === 'names') {
    scope.containers = scopeNames.value.split(',').map((v) => v.trim()).filter(Boolean);
  }
  if (form.scope.type === 'patterns') {
    scope.patterns = scopePatterns.value.split(',').map((v) => v.trim()).filter(Boolean);
  }
  return scope;
}

function buildConfig() {
  if (form.source_type === 'logs') {
    return {
      patterns: [{ pattern: logConfig.pattern }],
      match_count: logConfig.match_count,
      window_seconds: logConfig.window_seconds,
      case_sensitive: logConfig.case_sensitive,
    };
  }
  if (form.source_type === 'events') {
    return { ...eventConfig };
  }
  if (form.source_type === 'k8s_events') {
    return { ...k8sEventConfig };
  }
  return { ...metricConfig };
}

function buildRuleUpdatePayload(rule, overrides = {}) {
  const channelIds = overrides.channel_ids ?? rule.channel_ids ?? [];
  return {
    rule_id: rule.rule_id,
    name: rule.name,
    description: rule.description || '',
    severity: rule.severity,
    enabled: overrides.enabled ?? rule.enabled,
    source_type: rule.source_type,
    config: rule.config,
    scope: rule.scope || { type: 'all' },
    channel_ids: channelIds,
    cooldown_minutes: rule.cooldown_minutes,
    max_per_hour: rule.max_per_hour,
    group_window_minutes: rule.group_window_minutes,
    recovery_enabled: rule.recovery_enabled,
  };
}

function configuredChannelIds() {
  return channels.value.map((channel) => channel.id);
}

function buildPayload() {
  return {
    rule_id: form.rule_id,
    name: form.name,
    description: form.description,
    severity: form.severity,
    enabled: form.enabled,
    source_type: form.source_type,
    config: buildConfig(),
    scope: buildScope(),
    channel_ids: form.channel_ids,
    cooldown_minutes: form.cooldown_minutes,
    max_per_hour: form.max_per_hour,
    group_window_minutes: form.group_window_minutes,
    recovery_enabled: form.recovery_enabled,
  };
}

async function load() {
  loading.value = true;
  try {
    const [alertsData, notificationData] = await Promise.all([fetchAlerts(), fetchNotifications()]);
    rules.value = alertsData.rules || [];
    channels.value = (notificationData.channels || []).filter((c) => c.configured);
  } catch (err) {
    showToast('Could not load alerts', err.message || 'Failed to load alerts', 'error');
  } finally {
    loading.value = false;
  }
}

async function openHistory() {
  tab.value = 'history';
  historyLoading.value = true;
  try {
    const data = await fetchAlertHistory();
    history.value = data.history || [];
    historyLoaded.value = true;
  } catch (err) {
    showToast('Could not load history', err.message || 'Failed to load history', 'error');
  } finally {
    historyLoading.value = false;
  }
}

function openCreate() {
  editingId.value = 0;
  activeSection.value = 'basics';
  Object.assign(form, {
    rule_id: '', name: '', description: '', severity: 'warning', enabled: true,
    source_type: 'logs', channel_ids: [], cooldown_minutes: 15, max_per_hour: 20,
    group_window_minutes: 5, recovery_enabled: false,
  });
  form.scope = { type: 'all' };
  scopeNames.value = '';
  scopePatterns.value = '';
  editorOpen.value = true;
  nextTick(() => editorBodyRef.value?.scrollTo({ top: 0 }));
}

function editRule(rule) {
  editingId.value = rule.id;
  activeSection.value = 'basics';
  Object.assign(form, {
    rule_id: rule.rule_id,
    name: rule.name,
    description: rule.description,
    severity: rule.severity,
    enabled: rule.enabled,
    source_type: rule.source_type,
    channel_ids: [...(rule.channel_ids || [])],
    cooldown_minutes: rule.cooldown_minutes,
    max_per_hour: rule.max_per_hour,
    group_window_minutes: rule.group_window_minutes,
    recovery_enabled: rule.recovery_enabled,
  });
  form.scope = { ...(rule.scope || { type: 'all' }) };
  scopeNames.value = (rule.scope?.containers || []).join(', ');
  scopePatterns.value = (rule.scope?.patterns || []).join(', ');
  if (rule.source_type === 'logs' && rule.config) {
    Object.assign(logConfig, {
      pattern: rule.config.patterns?.[0]?.pattern || 'ERROR',
      match_count: rule.config.match_count || 10,
      window_seconds: rule.config.window_seconds || 120,
      case_sensitive: !!rule.config.case_sensitive,
    });
  }
  if (rule.source_type === 'events' && rule.config) Object.assign(eventConfig, rule.config);
  if (rule.source_type === 'k8s_events' && rule.config) Object.assign(k8sEventConfig, rule.config);
  if (rule.source_type === 'metrics' && rule.config) Object.assign(metricConfig, rule.config);
  editorOpen.value = true;
  nextTick(() => editorBodyRef.value?.scrollTo({ top: 0 }));
}

function closeEditor() {
  editorOpen.value = false;
}

async function saveRule() {
  saving.value = true;
  try {
    const payload = buildPayload();
    if (editingId.value) {
      await updateAlertRule(editingId.value, payload);
      showToast('Rule updated', 'Alert rule saved', 'success');
    } else {
      await createAlertRule(payload);
      showToast('Rule created', 'Alert rule saved', 'success');
    }
    closeEditor();
    await load();
  } catch (err) {
    showToast('Could not save rule', err.message || 'Failed to save rule', 'error');
  } finally {
    saving.value = false;
  }
}

async function toggleRule(rule) {
  const nextEnabled = !rule.enabled;
  let channelIds = [...(rule.channel_ids || [])];
  if (nextEnabled && channelIds.length === 0) {
    channelIds = configuredChannelIds();
    if (!channelIds.length) {
      showToast(
        'No notification channel',
        'Save a webhook on the Notifications page first, then try again.',
        'error',
      );
      return;
    }
  }
  try {
    await updateAlertRule(rule.id, buildRuleUpdatePayload(rule, { enabled: nextEnabled, channel_ids: channelIds }));
    await load();
    showToast(nextEnabled ? 'Rule enabled' : 'Rule disabled', rule.name, 'success');
  } catch (err) {
    showToast('Could not update rule', err.message || 'Failed to update rule', 'error');
  }
}

async function removeRule(rule) {
  if (!confirm(`Delete rule "${rule.name}"?`)) return;
  deletingRuleId.value = rule.id;
  try {
    await deleteAlertRule(rule.id);
    showToast('Rule deleted', rule.name, 'success');
    await load();
  } catch (err) {
    showToast('Could not delete rule', err.message || 'Failed to delete rule', 'error');
  } finally {
    deletingRuleId.value = 0;
  }
}

async function testRule(rule) {
  testingRuleId.value = rule.id;
  try {
    await testAlertRule(rule.id);
    showToast('Test sent', 'Check your notification channel', 'success');
  } catch (err) {
    showToast('Test failed', err.message || 'Failed to send test alert', 'error');
  } finally {
    testingRuleId.value = 0;
  }
}

onMounted(load);
</script>

<style scoped>
.alerts-body {
  display: grid;
  gap: 1rem;
}

.alerts-toolbar {
  margin-bottom: 0;
}

.alerts-rules {
  display: grid;
  gap: 1rem;
}

.section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.section-head.compact {
  margin-bottom: 0.75rem;
}

.section-head h2 {
  margin: 0;
  font-size: 1rem;
  font-weight: 800;
  color: var(--text-main);
}

.section-sub {
  margin: 0.2rem 0 0;
  font-size: 0.82rem;
  color: var(--text-mute);
}

.rules-panel {
  padding-bottom: 1rem;
}

.rule-name-cell {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
}

.rule-name {
  font-size: 0.92rem;
  font-weight: 800;
  color: var(--text-main);
}

.rule-id {
  font-family: var(--font-mono);
  font-size: 0.72rem;
  color: var(--text-mute);
  background: var(--bg-input);
  padding: 0.15rem 0.45rem;
  border-radius: 6px;
  border: 1px solid var(--border);
  width: fit-content;
}

.severity-badge,
.source-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.2rem 0.55rem;
  border-radius: 999px;
  font-size: 0.68rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.severity-badge.info { background: rgba(8, 145, 178, 0.14); color: #22d3ee; }
.severity-badge.warning { background: rgba(245, 158, 11, 0.14); color: #fbbf24; }
.severity-badge.critical { background: rgba(239, 68, 68, 0.14); color: #f87171; }

.source-badge.logs { background: rgba(8, 145, 178, 0.12); color: var(--accent); }
.source-badge.events { background: rgba(139, 92, 246, 0.12); color: #a78bfa; }
.source-badge.k8s_events { background: rgba(59, 130, 246, 0.12); color: #60a5fa; }
.source-badge.metrics { background: rgba(245, 158, 11, 0.12); color: #fbbf24; }

.dest-count {
  display: inline-flex;
  align-items: baseline;
  gap: 0.25rem;
  font-variant-numeric: tabular-nums;
  font-weight: 800;
  color: var(--text-main);
}

.dest-count.muted {
  color: var(--text-mute);
}

.dest-count.warn .dest-label {
  color: var(--warning);
}

.dest-label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--text-mute);
}

.col-actions {
  text-align: right;
  white-space: nowrap;
}

.action-group {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
}

.action-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  min-width: 3.75rem;
  padding: 0.38rem 0.7rem;
  border-radius: 8px;
  border: 1px solid color-mix(in srgb, var(--accent) 32%, var(--border));
  background: color-mix(in srgb, var(--accent) 10%, var(--bg-input));
  font-size: 0.76rem;
  font-weight: 700;
  color: var(--accent);
  cursor: pointer;
  transition:
    background 0.15s ease,
    border-color 0.15s ease,
    transform 0.1s ease,
    box-shadow 0.15s ease,
    opacity 0.15s ease;
}

.action-link:hover:not(:disabled) {
  background: color-mix(in srgb, var(--accent) 18%, var(--bg-input));
  border-color: color-mix(in srgb, var(--accent) 48%, var(--border));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--accent) 12%, transparent);
}

.action-link:active:not(:disabled) {
  transform: scale(0.96);
}

.action-link:focus-visible {
  outline: 2px solid color-mix(in srgb, var(--accent) 55%, transparent);
  outline-offset: 2px;
}

.action-link.is-active {
  background: color-mix(in srgb, var(--accent) 22%, var(--bg-input));
  border-color: var(--accent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 25%, transparent);
}

.action-link:disabled {
  opacity: 0.72;
  cursor: wait;
}

.action-link.is-busy {
  opacity: 0.9;
}

.action-link.danger {
  color: var(--error);
  border-color: color-mix(in srgb, var(--error) 35%, var(--border));
  background: color-mix(in srgb, var(--error) 10%, var(--bg-input));
}

.action-link.danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--error) 18%, var(--bg-input));
  border-color: color-mix(in srgb, var(--error) 50%, var(--border));
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--error) 12%, transparent);
}

.action-link.danger:focus-visible {
  outline-color: color-mix(in srgb, var(--error) 55%, transparent);
}

.action-spinner {
  width: 0.72rem;
  height: 0.72rem;
  border: 2px solid color-mix(in srgb, currentColor 28%, transparent);
  border-top-color: currentColor;
  border-radius: 50%;
  animation: action-spin 0.7s linear infinite;
  flex-shrink: 0;
}

@keyframes action-spin {
  to {
    transform: rotate(360deg);
  }
}

.premium-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.6rem;
  cursor: pointer;
  user-select: none;
}

.toggle-rail {
  width: 36px;
  height: 20px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 20px;
  position: relative;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}

.premium-toggle.active .toggle-rail {
  background: var(--success);
  border-color: var(--success);
  box-shadow: 0 0 12px rgba(16, 185, 129, 0.2);
}

.toggle-handle {
  width: 14px;
  height: 14px;
  background: #fff;
  border-radius: 50%;
  position: absolute;
  top: 2px;
  left: 2px;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
}

.premium-toggle.active .toggle-handle {
  transform: translateX(16px);
}

.status-label {
  font-size: 0.72rem;
  font-weight: 800;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.02em;
  min-width: 1.5rem;
}

.premium-toggle.active .status-label {
  color: var(--success);
}

.empty-state-wrap {
  display: grid;
  place-items: center;
  padding: 3rem 1.5rem;
}

.empty-state-content {
  max-width: 360px;
  text-align: center;
  display: grid;
  gap: 0.75rem;
  justify-items: center;
}

.empty-icon {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  background: var(--accent-soft);
  color: var(--accent);
  display: grid;
  place-items: center;
}

.empty-state-content h3 {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 800;
  color: var(--text-main);
}

.empty-state-content p {
  margin: 0;
  font-size: 0.86rem;
  color: var(--text-mute);
  line-height: 1.5;
}

.history-list {
  display: grid;
  gap: 0.75rem;
}

.history-card {
  padding: 1rem 1.1rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-input);
  transition: border-color 0.2s ease;
}

.history-card:hover {
  border-color: var(--border-active);
}

.history-card-head {
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  flex-wrap: wrap;
  margin-bottom: 0.55rem;
}

.history-title-wrap {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  min-width: 0;
  flex: 1;
}

.history-title-wrap strong {
  font-size: 0.92rem;
  font-weight: 800;
  color: var(--text-main);
}

.history-status {
  font-size: 0.72rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-mute);
}

.history-status.sent,
.history-status.delivered {
  color: var(--success);
}

.history-status.failed,
.history-status.error {
  color: var(--error);
}

.history-time {
  margin-left: auto;
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--text-mute);
  white-space: nowrap;
}

.history-message {
  margin: 0;
  font-size: 0.86rem;
  line-height: 1.5;
  color: var(--text-dim);
}

.history-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.65rem;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.25rem 0.55rem;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 700;
  background: var(--bg-subtle);
  border: 1px solid var(--border);
  color: var(--text-dim);
}

.meta-chip.muted {
  color: var(--text-mute);
}

.loading-state {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--text-mute);
  font-weight: 600;
}

.hint {
  font-size: 0.82rem;
  color: var(--text-mute);
}

.hint.warn {
  color: var(--warning);
}

/* Editor modal */
.editor-overlay {
  padding: 1rem;
}

.alert-editor {
  width: min(920px, 100%);
  max-height: min(92vh, 960px);
  display: flex;
  flex-direction: column;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 24px;
  overflow: hidden;
}

.modal-card-header {
  padding: 1.35rem 1.5rem;
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 1rem;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.header-content {
  display: flex;
  gap: 0.85rem;
  align-items: flex-start;
  min-width: 0;
  flex: 1;
}

.header-copy {
  min-width: 0;
  flex: 1;
}

.header-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.header-icon {
  width: 44px;
  height: 44px;
  background: var(--accent-soft);
  color: var(--accent);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.modal-title {
  margin: 0;
  font-size: 1.15rem;
  font-weight: 800;
  color: var(--text-main);
}

.modal-subtitle {
  margin: 0.2rem 0 0;
  font-size: 0.82rem;
  color: var(--text-mute);
  line-height: 1.45;
}

.header-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
  margin-top: 0.65rem;
}

.rule-id-chip {
  font-family: var(--font-mono);
  font-size: 0.72rem;
  color: var(--text-mute);
  background: var(--bg-input);
  padding: 0.2rem 0.5rem;
  border-radius: 6px;
  border: 1px solid var(--border);
}

.header-toggle {
  flex-shrink: 0;
}

.header-toggle .status-label {
  min-width: auto;
}

.close-btn {
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--bg-input);
  color: var(--text-dim);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}

.close-btn:hover {
  color: var(--text-main);
  border-color: var(--border-active);
  background: var(--bg-subtle);
}

.editor-shell {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  min-height: 0;
  flex: 1 1 auto;
}

.editor-nav {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 1rem;
  border-right: 1px solid var(--border);
  background: var(--bg-subtle);
}

.editor-nav-btn {
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  width: 100%;
  padding: 0.7rem 0.75rem;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease;
}

.editor-nav-btn:hover {
  background: var(--bg-card);
  border-color: var(--border);
}

.editor-nav-btn.active {
  background: var(--bg-card);
  border-color: rgba(var(--accent-rgb), 0.35);
  box-shadow: 0 8px 20px -14px var(--shadow);
}

.nav-step {
  width: 22px;
  height: 22px;
  border-radius: 999px;
  display: grid;
  place-items: center;
  font-size: 0.68rem;
  font-weight: 800;
  color: var(--text-mute);
  background: var(--bg-input);
  border: 1px solid var(--border);
  flex-shrink: 0;
}

.editor-nav-btn.active .nav-step {
  background: var(--accent);
  border-color: transparent;
  color: #fff;
}

.nav-copy {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.nav-copy strong {
  font-size: 0.8rem;
  font-weight: 800;
  color: var(--text-main);
}

.nav-copy span {
  font-size: 0.72rem;
  color: var(--text-mute);
  line-height: 1.35;
}

.editor-body {
  padding: 1.25rem 1.5rem 1.5rem;
  overflow-y: auto;
  min-height: 0;
  display: grid;
  gap: 1rem;
  align-content: start;
  scroll-behavior: smooth;
}

.editor-section {
  display: grid;
  gap: 0.9rem;
  padding: 1rem 1.05rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-input);
  scroll-margin-top: 0.75rem;
}

.section-head h4 {
  margin: 0;
  font-size: 0.95rem;
  font-weight: 800;
  color: var(--text-main);
}

.section-head p {
  margin: 0.2rem 0 0;
  font-size: 0.8rem;
  color: var(--text-mute);
  line-height: 1.45;
}

.form-grid {
  display: grid;
  gap: 0.75rem;
}

.form-grid.dual {
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.form-grid.triple {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.form-grid.align-end {
  align-items: end;
}

.input-group {
  display: grid;
  gap: 0.45rem;
}

.label-caps {
  font-size: 0.68rem;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-mute);
}

.alert-editor .premium-input {
  width: 100%;
  padding: 0.75rem 0.95rem;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 12px;
  color: var(--text-main);
  font-size: 0.86rem;
  font-weight: 600;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
}

.alert-editor .premium-input:focus {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px rgba(var(--accent-rgb), 0.12);
  background: var(--bg-subtle);
}

.alert-editor .premium-input.readonly,
.alert-editor .premium-input:disabled {
  opacity: 0.72;
  cursor: not-allowed;
}

.alert-editor .premium-input.mono {
  font-family: var(--font-mono);
  font-size: 0.82rem;
}

.alert-editor .premium-input.textarea {
  min-height: 76px;
  resize: vertical;
  font-family: inherit;
  line-height: 1.45;
}

.field-hint {
  margin: 0;
  font-size: 0.76rem;
  color: var(--text-mute);
}

.choice-row {
  display: grid;
  gap: 0.55rem;
}

.severity-row {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.choice-chip {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  padding: 0.75rem 0.85rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-card);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
}

.choice-chip strong {
  font-size: 0.82rem;
  font-weight: 800;
  color: var(--text-main);
}

.choice-chip span {
  font-size: 0.72rem;
  color: var(--text-mute);
}

.choice-chip.info.active {
  border-color: rgba(8, 145, 178, 0.45);
  background: rgba(8, 145, 178, 0.1);
}

.choice-chip.warning.active {
  border-color: rgba(245, 158, 11, 0.45);
  background: rgba(245, 158, 11, 0.1);
}

.choice-chip.critical.active {
  border-color: rgba(239, 68, 68, 0.45);
  background: rgba(239, 68, 68, 0.1);
}

.source-picker {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 0.55rem;
}

.source-card {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  padding: 0.85rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-card);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease, transform 0.2s ease;
}

.source-card:hover {
  transform: translateY(-1px);
}

.source-card-label {
  font-size: 0.86rem;
  font-weight: 800;
  color: var(--text-main);
}

.source-card-hint {
  font-size: 0.72rem;
  line-height: 1.4;
  color: var(--text-mute);
}

.source-card.logs.active {
  border-color: rgba(8, 145, 178, 0.45);
  background: rgba(8, 145, 178, 0.08);
}

.source-card.events.active {
  border-color: rgba(139, 92, 246, 0.45);
  background: rgba(139, 92, 246, 0.08);
}

.source-card.k8s_events.active {
  border-color: rgba(59, 130, 246, 0.45);
  background: rgba(59, 130, 246, 0.08);
}

.source-card.metrics.active {
  border-color: rgba(245, 158, 11, 0.45);
  background: rgba(245, 158, 11, 0.08);
}

.trigger-panel {
  display: grid;
  gap: 0.85rem;
  padding-top: 0.15rem;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
}

.text-link {
  padding: 0;
  border: none;
  background: none;
  color: var(--accent);
  font-size: 0.8rem;
  font-weight: 700;
  cursor: pointer;
  text-decoration: underline;
}

.event-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  gap: 0.65rem;
}

.event-card {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  padding: 0.85rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-card);
  cursor: pointer;
  position: relative;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.event-card input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.event-card.active {
  border-color: rgba(var(--accent-rgb), 0.45);
  background: var(--accent-soft);
}

.event-card-title {
  font-size: 0.82rem;
  font-weight: 800;
  color: var(--text-main);
}

.event-card-desc {
  font-size: 0.74rem;
  color: var(--text-mute);
  line-height: 1.35;
}

.scope-picker {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.45rem;
  padding: 0.35rem;
  border-radius: var(--radius-md);
  background: var(--bg-card);
  border: 1px solid var(--border);
}

.scope-choice {
  padding: 0.65rem 0.75rem;
  border: none;
  border-radius: calc(var(--radius-md) - 2px);
  background: transparent;
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--text-dim);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
}

.scope-choice.active {
  background: var(--bg-subtle);
  color: var(--accent);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.channel-warn {
  margin: 0;
}

.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(210px, 1fr));
  gap: 0.65rem;
}

.channel-card {
  display: flex;
  align-items: center;
  gap: 0.7rem;
  padding: 0.85rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-card);
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.channel-card input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.channel-card.active {
  border-color: rgba(var(--accent-rgb), 0.45);
  background: var(--accent-soft);
}

.channel-card-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: var(--bg-subtle);
  flex-shrink: 0;
}

.channel-card-icon.slack { background: rgba(224, 30, 90, 0.12); }
.channel-card-icon.teams { background: rgba(0, 120, 212, 0.12); }
.channel-card-icon.discord { background: rgba(88, 101, 242, 0.12); }
.channel-card-icon.custom { background: rgba(99, 102, 241, 0.12); }

.channel-card-copy {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
  min-width: 0;
}

.channel-card-copy strong {
  font-size: 0.84rem;
  font-weight: 800;
  color: var(--text-main);
}

.channel-card-copy span {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--text-mute);
  text-transform: lowercase;
}

.panel-toggle {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 0.85rem 0.95rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border);
  background: var(--bg-card);
}

.recovery-toggle {
  margin-top: 0.15rem;
}

.toggle-copy {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.toggle-copy strong {
  font-size: 0.84rem;
  font-weight: 800;
  color: var(--text-main);
}

.toggle-copy span {
  font-size: 0.76rem;
  color: var(--text-mute);
  line-height: 1.4;
}

.modal-card-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: flex-end;
  gap: 0.65rem;
  flex-shrink: 0;
  background: var(--bg-subtle);
}

.page-hero-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.page-hero-mark {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  background: var(--accent-soft);
  color: var(--accent);
  flex-shrink: 0;
}

.modal-bounce-enter-active,
.modal-bounce-leave-active {
  transition: opacity 0.25s ease;
}

.modal-bounce-enter-active .alert-editor,
.modal-bounce-leave-active .alert-editor {
  transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1), opacity 0.25s ease;
}

.modal-bounce-enter-from,
.modal-bounce-leave-to {
  opacity: 0;
}

.modal-bounce-enter-from .alert-editor,
.modal-bounce-leave-to .alert-editor {
  transform: scale(0.96) translateY(8px);
  opacity: 0;
}

@media (max-width: 900px) {
  .form-grid.dual,
  .form-grid.triple,
  .severity-row,
  .source-picker,
  .scope-picker {
    grid-template-columns: 1fr;
  }

  .editor-shell {
    grid-template-columns: 1fr;
  }

  .editor-nav {
    flex-direction: row;
    overflow-x: auto;
    border-right: none;
    border-bottom: 1px solid var(--border);
    padding: 0.75rem;
  }

  .editor-nav-btn {
    min-width: 150px;
    flex-shrink: 0;
  }

  .nav-copy span {
    display: none;
  }

  .header-title-row {
    flex-direction: column;
    align-items: flex-start;
  }

  .history-time {
    margin-left: 0;
    width: 100%;
  }

  .premium-table.alerts-table thead {
    display: none;
  }

  .premium-table.alerts-table tr {
    display: grid;
    gap: 0.5rem;
    padding: 1rem;
    border-bottom: 1px solid var(--border);
  }

  .premium-table.alerts-table td {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0;
    border: none;
  }

  .premium-table.alerts-table td::before {
    content: attr(data-label);
    font-size: 0.68rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--text-mute);
    flex-shrink: 0;
  }

  .col-actions {
    justify-content: flex-start;
  }

  .col-actions::before {
    display: none;
  }
}
</style>
