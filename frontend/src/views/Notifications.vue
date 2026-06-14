<template>
  <div class="page-view notifications-view animate-fade-in">
    <section class="page-hero">
      <div class="page-hero-body">
        <div class="page-hero-copy">
          <span class="page-hero-eyebrow">Administration</span>
          <div class="page-hero-title-row">
            <span class="page-hero-mark">
              <BrandIcon name="notifications" :size="26" :colored="false" />
            </span>
            <h1>Notifications</h1>
          </div>
          <p class="page-hero-sub">
            Route DockLog alerts to Slack, Teams, or Discord. Each channel has its own webhook and event filters.
          </p>
        </div>
        <div class="page-hero-stats">
          <div class="page-hero-stat" :class="form.enabled ? 'success' : 'muted'">
            <span class="page-hero-stat-val">{{ form.enabled ? "On" : "Off" }}</span>
            <span class="page-hero-stat-lbl">Delivery</span>
          </div>
          <div class="page-hero-stat">
            <span class="page-hero-stat-val">{{ activeChannelCount }}</span>
            <span class="page-hero-stat-lbl">Active channels</span>
          </div>
          <div class="page-hero-stat" :class="validation.ready || !form.enabled ? 'success' : 'warning'">
            <span class="page-hero-stat-val">{{ validation.ready || !form.enabled ? "Ready" : "Fix" }}</span>
            <span class="page-hero-stat-lbl">Configuration</span>
          </div>
        </div>
      </div>
      <div class="page-hero-mesh" aria-hidden="true"></div>
    </section>

    <div v-if="loading" class="loading-state">Loading notification settings...</div>

    <form v-else class="notifications-form" @submit.prevent="saveSettings">
      <div class="notifications-layout">
        <aside class="notifications-sidebar page-panel flush">
          <div class="sidebar-block">
            <h3>Delivery</h3>
            <label class="delivery-toggle">
              <input
                type="checkbox"
                v-model="form.enabled"
                class="delivery-toggle-input"
              />
              <span class="delivery-toggle-copy">
                <span class="delivery-toggle-title">Enable notifications</span>
                <span class="delivery-toggle-hint">Master switch for all channels</span>
              </span>
            </label>
            <p v-if="!form.enabled && activeChannelCount > 0" class="hint warn delivery-warning">
              Delivery is off. Live container alerts will not send until you turn this on and save.
            </p>
          </div>

          <div class="sidebar-block">
            <h3>Readiness</h3>
            <ul class="checklist">
              <li :class="{ ok: form.enabled || activeChannelCount === 0 }">
                Delivery enabled for live alerts
              </li>
              <li :class="{ ok: !form.enabled || activeChannelCount > 0 }">
                At least one channel with a webhook
              </li>
              <li :class="{ ok: !form.enabled || eventRouteCount > 0 }">
                At least one channel routing events
              </li>
              <li :class="{ ok: !form.enabled || channelIssues.length === 0 }">
                No validation errors
              </li>
              <li :class="{ ok: !dirty }">
                {{ dirty ? "Unsaved changes" : "Saved configuration" }}
              </li>
            </ul>
          </div>

          <div class="sidebar-block">
            <h3>Channels</h3>
          <div class="channel-nav">
            <button
              v-for="channelType in availableChannelTypes"
              :key="channelType.type"
              type="button"
              class="channel-nav-btn"
              :class="{
                active: activeTab === channelType.type,
                [channelStatus(channelType)]: true,
              }"
              @click="activeTab = channelType.type"
            >
              <span class="nav-icon" :class="channelType.type">
                <ChannelIcon :type="channelType.type" :size="18" />
              </span>
              <span class="nav-label">{{ channelType.label }}</span>
              <span class="nav-status">{{ statusLabel(channelType) }}</span>
            </button>
          </div>
          </div>

          <button type="button" class="page-btn secondary full-width" @click="openGuideModal()">
          Webhook setup guide
        </button>
      </aside>

      <div class="notifications-main">
        <div
          v-for="channelType in availableChannelTypes"
          v-show="activeTab === channelType.type"
          :key="channelType.type"
          class="channel-editor page-panel"
        >
          <div class="editor-header">
            <div class="editor-icon" :class="channelType.type">
              <ChannelIcon :type="channelType.type" :size="24" />
            </div>
            <div>
              <h2>{{ channelType.label }}</h2>
              <p>{{ channelType.description }}</p>
            </div>
            <span class="badge" :class="statusBadgeClass(channelType)">{{ statusLabel(channelType) }}</span>
          </div>

          <div v-if="isConfigured(channelType.type) && !channelForms[channelType.type]?.clear" class="stored-url">
            <span class="stored-label">Saved webhook</span>
            <code class="mono">{{ maskedConfig(channelType.type) }}</code>
            <span class="stored-hint">Leave the field empty to keep this URL. Paste a new URL to replace it.</span>
          </div>

            <label class="toggle-row channel-toggle">
            <input
              type="checkbox"
              v-model="channelForms[channelType.type].enabled"
              class="toggle-row-input"
              @change="onChannelEnabledChange(channelType.type)"
            />
            <span class="toggle-row-label">Enable {{ channelType.label }}</span>
          </label>

          <div
            v-for="field in channelType.config_fields"
            :key="field.key"
            class="input-group"
          >
            <label :for="`${channelType.type}-${field.key}`">{{ field.label }}</label>
            <input
              :id="`${channelType.type}-${field.key}`"
              v-model="channelForms[channelType.type].config[field.key]"
              :type="field.secret ? 'password' : 'text'"
              class="premium-input"
              :class="{ invalid: fieldError(channelType.type, field.key) }"
              :placeholder="field.placeholder"
              autocomplete="off"
              @input="touch()"
              @blur="markTouched(channelType.type, field.key)"
            />
            <p v-if="fieldError(channelType.type, field.key)" class="field-error">
              {{ fieldError(channelType.type, field.key) }}
            </p>
          </div>

          <button type="button" class="text-link" @click="openGuideModal(channelType.type)">
            How to create a {{ channelType.label }} webhook →
          </button>

          <div class="event-block">
            <div class="event-block-head">
              <h3>Events for this channel</h3>
              <button
                type="button"
                class="text-link"
                @click="setAllEvents(channelType.type, true)"
              >
                Select all
              </button>
            </div>
            <div class="event-grid">
              <label
                v-for="eventType in eventTypes"
                :key="`${channelType.type}-${eventType.key}`"
                class="event-card"
                :class="{ active: channelForms[channelType.type].events[eventType.key] }"
              >
                <input
                  type="checkbox"
                  v-model="channelForms[channelType.type].events[eventType.key]"
                  @change="touch()"
                />
                <span class="event-card-title">{{ eventType.label }}</span>
                <span class="event-card-desc">{{ eventType.description }}</span>
              </label>
            </div>
            <p v-if="fieldError(channelType.type, 'events')" class="field-error">
              {{ fieldError(channelType.type, "events") }}
            </p>
          </div>

          <div class="editor-actions">
            <button
              type="button"
              class="page-btn secondary"
              :disabled="testing || !canTestChannel(channelType.type)"
              @click="sendTest(channelType.type)"
            >
              Test {{ channelType.label }}
            </button>
            <button
              v-if="isConfigured(channelType.type)"
              type="button"
              class="page-btn danger-outline"
              @click="toggleRemove(channelType.type)"
            >
              {{ channelForms[channelType.type].clear ? "Undo remove" : "Remove integration" }}
            </button>
          </div>
          <p v-if="canTestChannel(channelType.type) && dirty" class="hint warn">
            Save settings before testing so DockLog uses the latest webhook URL.
          </p>
        </div>

        <div v-for="channelType in upcomingChannelTypes" :key="channelType.type" class="channel-editor page-panel muted-panel">
          <div class="editor-header compact">
            <div class="editor-icon" :class="channelType.type">
              <ChannelIcon :type="channelType.type" :size="22" />
            </div>
            <div>
              <h2>{{ channelType.label }}</h2>
              <p class="hint">{{ channelType.description }} (coming soon)</p>
            </div>
          </div>
        </div>
      </div>
      </div>

      <footer class="save-bar" :class="{ invalid: validation.issues.length > 0 }">
        <div class="save-bar-copy">
          <strong v-if="form.enabled && validation.ready">Configuration is valid</strong>
          <strong v-else-if="form.enabled">Fix issues before saving</strong>
          <strong v-else>Notifications are disabled</strong>
          <ul v-if="validation.issues.length" class="save-issues">
            <li v-for="(issue, index) in validation.issues" :key="index">{{ issue }}</li>
          </ul>
          <p v-else-if="dirty" class="hint">You have unsaved changes.</p>
        </div>
        <div class="save-bar-actions">
          <button
            type="button"
            class="page-btn secondary"
            :disabled="testing || saving || activeChannelCount === 0"
            @click="sendTest('all')"
          >
            Test all
          </button>
          <button type="submit" class="page-btn primary" :disabled="saving || !validation.valid">
            {{ saving ? "Saving..." : "Save settings" }}
          </button>
        </div>
      </footer>
    </form>

    <Teleport to="body">
      <Transition name="modal-bounce">
        <div
          v-if="guideModalOpen"
          class="modal-overlay"
          role="presentation"
          @click.self="closeGuideModal"
        >
          <div
            class="modal-card guide-modal glass shadow-2xl"
            role="dialog"
            aria-modal="true"
            :aria-labelledby="guideModalChannel ? 'guide-modal-title' : undefined"
          >
            <div class="modal-card-header">
              <div class="header-content">
                <div
                  v-if="guideModalChannel"
                  class="header-icon channel"
                  :class="guideModalChannel"
                >
                  <ChannelIcon :type="guideModalChannel" :size="22" />
                </div>
                <div v-else class="header-icon">
                  <BrandIcon name="notifications" :size="22" :colored="false" />
                </div>
                <div>
                  <h3 id="guide-modal-title" class="modal-title">{{ guideModalTitle }}</h3>
                  <p class="modal-subtitle">Step-by-step webhook setup</p>
                </div>
              </div>
              <button type="button" class="close-btn" aria-label="Close" @click="closeGuideModal">
                ×
              </button>
            </div>
            <div class="modal-card-body guide-modal-body">
              <div v-if="guideModalChannel" class="guide-modal-notes">
                <p><strong>Security:</strong> Webhook URLs are secrets. Store them only in DockLog.</p>
                <p><strong>HTTPS required:</strong> URLs must start with <code>https://</code>.</p>
              </div>
              <WebhookSetupGuide :channel-id="guideModalChannel" modal />
            </div>
            <div class="modal-card-footer">
              <button type="button" class="page-btn primary" @click="closeGuideModal">
                Done
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from "vue";
import BrandIcon from "../components/BrandIcon.vue";
import ChannelIcon from "../components/ChannelIcon.vue";
import WebhookSetupGuide from "../components/WebhookSetupGuide.vue";
import {
  fetchNotificationSettings,
  saveNotificationSettings,
  testNotification,
} from "../services/notificationService";
import { apiErrorMessage } from "../utils/authSession";
import {
  channelStatus,
  countActiveChannels,
  countEventRoutes,
  validateNotificationSettings,
} from "../utils/notificationValidation";
import { showToast } from "../utils/sharedState";

const loading = ref(true);
const saving = ref(false);
const testing = ref(false);
const dirty = ref(false);
const guideModalOpen = ref(false);
const guideModalChannel = ref("");
const activeTab = ref("slack");
const touchedFields = reactive({});

const guideLabels = {
  slack: "Slack",
  teams: "Microsoft Teams",
  discord: "Discord",
};

const guideModalTitle = computed(() => {
  if (!guideModalChannel.value) return "Webhook setup guide";
  const label = guideLabels[guideModalChannel.value] || guideModalChannel.value;
  return `How to create a ${label} webhook`;
});

const channelTypes = ref([]);
const eventTypes = ref([]);
const configuredChannels = ref([]);

const form = reactive({ enabled: false });
const channelForms = reactive({});
const savedSnapshot = ref("");

const availableChannelTypes = computed(() =>
  channelTypes.value.filter((c) => c.available),
);
const upcomingChannelTypes = computed(() =>
  channelTypes.value.filter((c) => !c.available),
);

const isConfigured = (type) =>
  configuredChannels.value.some((c) => c.type === type && c.configured);

const validation = computed(() =>
  validateNotificationSettings({
    enabled: form.enabled,
    channelTypes: channelTypes.value,
    channelForms,
    isConfigured,
  }),
);

const activeChannelCount = computed(() =>
  countActiveChannels(channelTypes.value, channelForms, isConfigured),
);

const eventRouteCount = computed(() =>
  countEventRoutes(channelTypes.value, channelForms, isConfigured),
);

const channelIssues = computed(() => validation.value.issues);

const defaultEvents = () => ({
  notify_container_actions: true,
  notify_security_events: true,
  notify_admin_actions: true,
  notify_health_events: false,
});

const ensureChannelForm = (type) => {
  if (!channelForms[type]) {
    channelForms[type] = {
      enabled: false,
      config: {},
      events: defaultEvents(),
      clear: false,
    };
  }
  return channelForms[type];
};

const touch = () => {
  dirty.value = snapshot() !== savedSnapshot.value;
};

const snapshot = () =>
  JSON.stringify({
    enabled: form.enabled,
    channels: channelTypes.value.map((ct) => ({
      type: ct.type,
      ...channelForms[ct.type],
    })),
  });

const markTouched = (type, field) => {
  if (!touchedFields[type]) touchedFields[type] = {};
  touchedFields[type][field] = true;
  touch();
};

const fieldError = (type, field) => {
  const show = touchedFields[type]?.[field] || validation.value.issues.length > 0;
  if (!show) return "";
  return validation.value.fieldErrors[type]?.[field] || "";
};

const channelHasWebhook = (type) => {
  const ct = channelTypes.value.find((c) => c.type === type);
  const entry = channelForms[type];
  if (!ct || !entry || entry.clear) return false;
  const hasNew = (ct.config_fields || []).some((f) => (entry.config[f.key] || "").trim());
  return isConfigured(type) || hasNew;
};

const canTestChannel = (type) =>
  channelForms[type]?.enabled && channelHasWebhook(type) && !dirty.value;

const statusLabel = (channelType) => {
  const status = channelStatus(channelType, channelForms[channelType.type], isConfigured);
  const map = {
    active: "Active",
    paused: "Paused",
    incomplete: "Needs URL",
    "no-events": "No events",
    disconnected: "Not set up",
  };
  return map[status] || status;
};

const statusBadgeClass = (channelType) => {
  const status = channelStatus(channelType, channelForms[channelType.type], isConfigured);
  if (status === "active") return "badge-success";
  if (status === "paused") return "badge-dim";
  if (status === "incomplete" || status === "no-events") return "badge-warning";
  return "badge-dim";
};

const maskedConfig = (type) => {
  const channel = configuredChannels.value.find((c) => c.type === type);
  if (!channel?.config_masked) return "";
  return channel.config_masked.webhook_url || Object.values(channel.config_masked).find(Boolean) || "";
};

const resetChannelForms = (data) => {
  for (const channelType of data.channel_types || []) {
    const existing = (data.channels || []).find((c) => c.type === channelType.type);
    const entry = ensureChannelForm(channelType.type);
    entry.enabled = existing?.enabled ?? false;
    entry.clear = false;
    entry.config = {};
    entry.events = {
      notify_container_actions: existing?.events?.notify_container_actions ?? true,
      notify_security_events: existing?.events?.notify_security_events ?? true,
      notify_admin_actions: existing?.events?.notify_admin_actions ?? true,
      notify_health_events: existing?.events?.notify_health_events ?? false,
    };
    for (const field of channelType.config_fields || []) {
      entry.config[field.key] = "";
    }
  }
  savedSnapshot.value = snapshot();
  dirty.value = false;
  Object.keys(touchedFields).forEach((k) => delete touchedFields[k]);
};

const onChannelEnabledChange = (type) => {
  if (channelForms[type]?.enabled) {
    form.enabled = true;
  }
  touch();
};

const setAllEvents = (type, value) => {
  const entry = channelForms[type];
  if (!entry) return;
  entry.events.notify_container_actions = value;
  entry.events.notify_security_events = value;
  entry.events.notify_admin_actions = value;
  entry.events.notify_health_events = value;
  touch();
};

const openGuideModal = (type = "") => {
  guideModalChannel.value = type || "";
  guideModalOpen.value = true;
};

const closeGuideModal = () => {
  guideModalOpen.value = false;
  guideModalChannel.value = "";
};

const toggleRemove = (type) => {
  const entry = channelForms[type];
  entry.clear = !entry.clear;
  if (entry.clear) entry.enabled = false;
  touch();
};

const loadSettings = async () => {
  loading.value = true;
  try {
    const data = await fetchNotificationSettings();
    channelTypes.value = data.channel_types || [];
    eventTypes.value = data.event_types || [];
    configuredChannels.value = data.channels || [];
    form.enabled = data.enabled;
    resetChannelForms(data);
    if (!availableChannelTypes.value.some((c) => c.type === activeTab.value)) {
      activeTab.value = availableChannelTypes.value[0]?.type || "slack";
    }
  } catch (e) {
    showToast("Error", apiErrorMessage(e, "Failed to load settings"), "error");
  } finally {
    loading.value = false;
  }
};

const buildPayload = () => {
  const payload = { enabled: form.enabled, channels: [] };

  for (const channelType of channelTypes.value) {
    if (!channelType.available) continue;
    const entry = channelForms[channelType.type];
    if (!entry) continue;

    const update = {
      type: channelType.type,
      enabled: entry.enabled,
      clear: entry.clear,
      config: {},
      events: { ...entry.events },
    };

    let hasInput = false;
    for (const field of channelType.config_fields) {
      const value = (entry.config[field.key] || "").trim();
      if (value) {
        update.config[field.key] = value;
        hasInput = true;
      }
    }

    if (entry.clear || hasInput || isConfigured(channelType.type) || entry.enabled) {
      payload.channels.push(update);
    }
  }

  return payload;
};

const saveSettings = async () => {
  const check = validateNotificationSettings({
    enabled: form.enabled,
    channelTypes: channelTypes.value,
    channelForms,
    isConfigured,
  });

  if (!check.valid) {
    showToast("Cannot save", check.issues[0] || "Fix validation errors", "error");
    return;
  }

  saving.value = true;
  try {
    const data = await saveNotificationSettings(buildPayload());
    channelTypes.value = data.channel_types || [];
    eventTypes.value = data.event_types || [];
    configuredChannels.value = data.channels || [];
    resetChannelForms(data);
    showToast("Saved", "Notification settings updated", "success");
  } catch (e) {
    showToast("Save failed", apiErrorMessage(e, "Failed to save settings"), "error");
  } finally {
    saving.value = false;
  }
};

const sendTest = async (target) => {
  if (!form.enabled) {
    showToast(
      "Delivery is off",
      "Turn on Enable notifications and save before testing live alerts",
      "warning",
    );
    return;
  }
  if (dirty.value) {
    showToast("Save first", "Save settings before sending a test notification", "warning");
    return;
  }

  testing.value = true;
  try {
    const data = await testNotification(target);
    showToast("Sent", data.message || "Test notification sent", "success");
  } catch (e) {
    showToast("Test failed", apiErrorMessage(e, "Test failed"), "error");
  } finally {
    testing.value = false;
  }
};

watch(
  () => form.enabled,
  () => touch(),
);

onMounted(loadSettings);
</script>

<style scoped>
.notifications-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.notifications-layout {
  display: grid;
  grid-template-columns: minmax(240px, 300px) minmax(0, 1fr);
  gap: 1.25rem;
  align-items: start;
}

.notifications-sidebar {
  position: sticky;
  top: 1rem;
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.sidebar-block {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.sidebar-block h3 {
  margin: 0;
  font-size: 0.78rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-mute);
}

.delivery-toggle {
  display: flex;
  align-items: flex-start;
  gap: 0.65rem;
  width: 100%;
  margin: 0;
  padding: 0.8rem 0.85rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-input);
  cursor: pointer;
}

.delivery-toggle-input {
  width: 1rem;
  height: 1rem;
  margin: 0.1rem 0 0;
  flex: 0 0 1rem;
  accent-color: var(--accent);
  cursor: pointer;
}

.delivery-toggle-copy {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  min-width: 0;
  flex: 1;
}

.delivery-toggle-title {
  font-size: 0.9rem;
  font-weight: 600;
  color: var(--text-main);
  line-height: 1.3;
}

.delivery-toggle-hint {
  font-size: 0.78rem;
  font-weight: 500;
  color: var(--text-mute);
  line-height: 1.35;
}

.checklist {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.55rem;
  font-size: 0.84rem;
  color: var(--text-mute);
}

.checklist li {
  position: relative;
  padding-left: 1.35rem;
}

.checklist li::before {
  content: "○";
  position: absolute;
  left: 0;
  color: var(--text-mute);
}

.checklist li.ok {
  color: var(--text-main);
}

.checklist li.ok::before {
  content: "✓";
  color: var(--success, #10b981);
}

.channel-nav {
  display: flex;
  flex-direction: column;
  gap: 0.45rem;
}

.channel-nav-btn {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 0.55rem;
  width: 100%;
  padding: 0.65rem 0.75rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-input);
  color: var(--text-main);
  cursor: pointer;
  text-align: left;
}

.page-hero-title-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.page-hero-mark {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--accent) 14%, var(--bg-input));
  color: var(--accent);
  flex-shrink: 0;
}

.nav-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  flex-shrink: 0;
}

.nav-icon.slack { background: rgba(224, 30, 90, 0.12); }
.nav-icon.teams { background: rgba(0, 120, 212, 0.12); }
.nav-icon.discord { background: rgba(88, 101, 242, 0.12); }
.nav-icon.email { background: rgba(100, 116, 139, 0.12); }

.editor-header.compact {
  margin-bottom: 0;
}

.channel-nav-btn.active {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, var(--bg-input));
}

.nav-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-mute);
}

.channel-nav-btn.status-active {
  border-color: color-mix(in srgb, #10b981 35%, var(--border));
}

.channel-nav-btn.status-incomplete,
.channel-nav-btn.status-no-events {
  border-color: color-mix(in srgb, #f59e0b 35%, var(--border));
}

.nav-label {
  font-size: 0.88rem;
  font-weight: 600;
}

.nav-status {
  font-size: 0.72rem;
  color: var(--text-mute);
}

.notifications-main {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  min-width: 0;
}

.guide-wrap {
  margin-bottom: 0.25rem;
}

/* Overlay uses global .modal-overlay from style.css (z-index: 5000) */

.guide-modal {
  width: 100%;
  max-width: 640px;
  max-height: min(88vh, 900px);
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
  align-items: center;
  min-width: 0;
}

.header-icon {
  width: 42px;
  height: 42px;
  background: var(--bg-subtle);
  color: var(--accent);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.header-icon.channel.slack { background: rgba(224, 30, 90, 0.12); }
.header-icon.channel.teams { background: rgba(0, 120, 212, 0.12); }
.header-icon.channel.discord { background: rgba(88, 101, 242, 0.12); }

.modal-title {
  margin: 0;
  font-size: 1.05rem;
  font-weight: 800;
  color: var(--text-main);
}

.modal-subtitle {
  margin: 0.15rem 0 0;
  font-size: 0.8rem;
  color: var(--text-mute);
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-mute);
  font-size: 1.5rem;
  line-height: 1;
  cursor: pointer;
  padding: 0.15rem 0.35rem;
}

.guide-modal-body {
  padding: 1.25rem 1.5rem 1.75rem;
  overflow-y: auto;
  flex: 1 1 auto;
  min-height: 0;
  overscroll-behavior: contain;
}

.guide-modal-notes {
  margin-bottom: 1rem;
  padding: 0.85rem 1rem;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--accent) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 20%, transparent);
  font-size: 0.82rem;
  color: var(--text-mute);
  line-height: 1.55;
}

.guide-modal-notes p {
  margin: 0;
}

.guide-modal-notes p + p {
  margin-top: 0.4rem;
}

.guide-modal-notes code {
  font-size: 0.8rem;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  background: var(--bg-input);
}

.modal-card-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
}

.modal-bounce-enter-active,
.modal-bounce-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.modal-bounce-enter-from,
.modal-bounce-leave-to {
  opacity: 0;
}

.modal-bounce-enter-from .guide-modal,
.modal-bounce-leave-to .guide-modal {
  transform: scale(0.96) translateY(8px);
}

.channel-editor {
  padding: 1.35rem;
}

.muted-panel {
  opacity: 0.7;
}

.editor-header {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1.25rem;
}

.editor-header h2 {
  margin: 0;
  font-size: 1.15rem;
}

.editor-header p {
  margin: 0.2rem 0 0;
  font-size: 0.85rem;
  color: var(--text-mute);
}

.editor-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  flex-shrink: 0;
}

.editor-icon.slack { background: rgba(224, 30, 90, 0.12); }
.editor-icon.teams { background: rgba(0, 120, 212, 0.12); }
.editor-icon.discord { background: rgba(88, 101, 242, 0.12); }
.editor-icon.email { background: rgba(100, 116, 139, 0.12); }

.stored-url {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.85rem;
  margin-bottom: 1rem;
  border-radius: var(--radius-md);
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.stored-label {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-mute);
}

.stored-hint {
  font-size: 0.78rem;
  color: var(--text-mute);
}

.toggle-row {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  width: fit-content;
  max-width: 100%;
  font-size: 0.9rem;
  font-weight: 600;
  margin: 0 0 1rem;
  cursor: pointer;
}

.toggle-row-input {
  width: 1rem;
  height: 1rem;
  margin: 0;
  flex: 0 0 1rem;
  accent-color: var(--accent);
  cursor: pointer;
}

.toggle-row-label {
  line-height: 1.35;
  color: var(--text-main);
}

.input-group {
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  margin-bottom: 0.85rem;
}

.input-group label {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.premium-input.invalid {
  border-color: var(--error);
}

.field-error {
  margin: 0;
  font-size: 0.8rem;
  color: var(--error);
}

.text-link {
  padding: 0;
  border: none;
  background: none;
  color: var(--accent);
  font-size: 0.84rem;
  font-weight: 600;
  cursor: pointer;
  text-decoration: underline;
  margin-bottom: 1rem;
}

.event-block {
  margin-top: 0.5rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border);
}

.event-block-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 0.75rem;
}

.event-block-head h3 {
  margin: 0;
  font-size: 0.95rem;
}

.event-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 0.75rem;
}

.event-card {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  padding: 0.85rem;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg-input);
  cursor: pointer;
  position: relative;
}

.event-card input {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.event-card.active {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 8%, var(--bg-input));
}

.event-card-title {
  font-size: 0.88rem;
  font-weight: 700;
}

.event-card-desc {
  font-size: 0.76rem;
  color: var(--text-mute);
  line-height: 1.45;
}

.editor-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  margin-top: 1.25rem;
}

.page-btn.danger-outline {
  border: 1px solid color-mix(in srgb, var(--error) 40%, transparent);
  color: var(--error);
  background: transparent;
}

.hint {
  margin: 0.75rem 0 0;
  font-size: 0.82rem;
  color: var(--text-mute);
}

.hint.warn {
  color: #f59e0b;
}

.save-bar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 1rem 1.25rem;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border);
  background: var(--bg-card);
  position: sticky;
  bottom: 0.75rem;
  z-index: 2;
}

.save-bar.invalid {
  border-color: color-mix(in srgb, var(--error) 35%, var(--border));
}

.save-bar-copy strong {
  display: block;
  margin-bottom: 0.35rem;
}

.save-issues {
  margin: 0;
  padding-left: 1.1rem;
  color: var(--error);
  font-size: 0.84rem;
}

.save-bar-actions {
  display: flex;
  gap: 0.75rem;
  flex-shrink: 0;
}

.full-width {
  width: 100%;
}

.mono {
  font-family: "JetBrains Mono", monospace;
  font-size: 0.8rem;
  word-break: break-all;
}

.loading-state {
  padding: 2rem;
  color: var(--text-mute);
}

@media (max-width: 960px) {
  .notifications-layout {
    grid-template-columns: 1fr;
  }

  .notifications-sidebar {
    position: static;
  }

  .channel-nav {
    flex-direction: row;
    flex-wrap: wrap;
  }

  .channel-nav-btn {
    flex: 1 1 calc(50% - 0.25rem);
  }
}
</style>
