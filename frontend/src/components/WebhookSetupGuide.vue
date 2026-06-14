<template>
  <section class="webhook-guide" :class="{ wide, modal: isModal }">
    <div v-if="!isModal" class="guide-header">
      <div>
        <h2>How to create webhooks</h2>
        <p>
          DockLog needs an <strong>incoming webhook URL</strong> for each channel. Paste the URL in the field above,
          pick event types, then <strong>Save settings</strong> before using Test.
        </p>
      </div>
      <button type="button" class="page-btn secondary" @click="expandAll = !expandAll">
        {{ expandAll ? "Collapse all" : "Expand all" }}
      </button>
    </div>

    <div v-if="!isModal" class="guide-notes">
      <p><strong>Security:</strong> Webhook URLs are secrets. Anyone with the URL can post to your channel. Store them only in DockLog, not in git or public tickets.</p>
      <p><strong>HTTPS required:</strong> URLs must start with <code>https://</code>. HTTP webhooks are rejected.</p>
    </div>

    <div
      v-for="guide in visibleGuides"
      :key="guide.id"
      :id="`webhook-guide-${guide.id}`"
      class="guide-panel"
      :class="[guide.id, { 'is-static': isStaticPanel }]"
    >
      <button
        v-if="!isStaticPanel"
        type="button"
        class="guide-panel-toggle"
        :aria-expanded="isOpen(guide.id)"
        @click="toggle(guide.id)"
      >
        <span class="guide-panel-icon" :class="guide.id">
          <ChannelIcon :type="guide.id" :size="20" />
        </span>
        <span class="guide-panel-title">
          <strong>{{ guide.label }}</strong>
          <small>{{ guide.summary }}</small>
        </span>
        <span class="chevron" :class="{ open: isOpen(guide.id) }">›</span>
      </button>

      <div v-else-if="!hideChannelHead" class="guide-panel-static-head">
        <span class="guide-panel-icon" :class="guide.id">
          <ChannelIcon :type="guide.id" :size="22" />
        </span>
        <div class="guide-panel-title">
          <strong>{{ guide.label }}</strong>
          <small>{{ guide.summary }}</small>
        </div>
      </div>

      <div v-show="isOpen(guide.id)" class="guide-panel-body" :class="{ 'no-border': isStaticPanel }">
        <p class="guide-intro">{{ guide.intro }}</p>

        <ol class="guide-steps">
          <li v-for="(step, index) in guide.steps" :key="index">
            <span v-html="step"></span>
          </li>
        </ol>

        <div class="guide-url-box">
          <span class="guide-url-label">URL looks like</span>
          <code class="mono">{{ guide.urlExample }}</code>
        </div>

        <ul v-if="guide.tips?.length" class="guide-tips">
          <li v-for="(tip, index) in guide.tips" :key="index">{{ tip }}</li>
        </ul>

        <a
          v-if="guide.docsUrl"
          :href="guide.docsUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="guide-docs-link"
        >
          Official documentation →
        </a>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, reactive, ref } from "vue";
import ChannelIcon from "./ChannelIcon.vue";

const props = defineProps({
  channelId: { type: String, default: "" },
  modal: { type: Boolean, default: false },
});

const isModal = computed(() => props.modal);
const isStaticPanel = computed(() => props.modal && !!props.channelId);
const hideChannelHead = computed(() => isStaticPanel.value);

const expandAll = ref(false);
const openPanels = reactive({
  slack: true,
  teams: false,
  discord: false,
});

const guides = [
  {
    id: "slack",
    label: "Slack",
    summary: "Incoming Webhook app",
    intro:
      "Create a Slack app with Incoming Webhooks enabled, then attach it to the channel where DockLog alerts should appear.",
    urlExample: "https://hooks.slack.com/services/…/…/…",
    docsUrl: "https://api.slack.com/messaging/webhooks",
    steps: [
      "Open <a href=\"https://api.slack.com/apps\" target=\"_blank\" rel=\"noopener\">api.slack.com/apps</a> and sign in to your workspace.",
      "Click <strong>Create New App</strong> → <strong>From scratch</strong>. Name it (e.g. <em>DockLog</em>) and select your workspace.",
      "In the app settings sidebar, open <strong>Incoming Webhooks</strong> and turn <strong>Activate Incoming Webhooks</strong> on.",
      "Click <strong>Add New Webhook to Workspace</strong>, choose the channel (e.g. <em>#alerts</em> or <em>#docklog</em>), and allow the app.",
      "Copy the generated <strong>Webhook URL</strong>. It starts with <code>https://hooks.slack.com/services/</code>.",
      "Paste the URL into the Slack card on this page, enable the channel, select events, and click <strong>Save settings</strong>.",
    ],
    tips: [
      "You can create multiple webhooks for different channels; each URL is unique.",
      "To rotate a leaked URL, remove the webhook in Slack app settings and create a new one, then update DockLog.",
      "Workspace admins may need to approve the app before webhooks work.",
    ],
  },
  {
    id: "teams",
    label: "Microsoft Teams",
    summary: "Incoming webhook connector",
    intro:
      "Add an Incoming Webhook to a Teams channel. Newer Teams tenants may use Workflows instead of classic Connectors. Both produce a URL you can paste here.",
    urlExample: "https://outlook.office.com/webhook/…  or  https://….webhook.office.com/webhookb2/…",
    docsUrl: "https://learn.microsoft.com/en-us/microsoftteams/platform/webhooks-and-connectors/how-to/add-incoming-webhook",
    steps: [
      "In Microsoft Teams, open the <strong>channel</strong> where alerts should be posted.",
      "Click <strong>⋯</strong> (More options) next to the channel name → <strong>Connectors</strong> or <strong>Workflows</strong> (depends on your Teams version).",
      "<strong>Classic Connectors:</strong> search for <strong>Incoming Webhook</strong>, click <strong>Add</strong>, name it (e.g. <em>DockLog</em>), optionally upload an icon, then <strong>Create</strong>. Copy the URL.",
      "<strong>Workflows (newer Teams):</strong> create a workflow triggered by <strong>When a webhook request is received</strong>, add a <strong>Post message in a chat or channel</strong> action, save, then copy the webhook URL from the trigger step.",
      "Paste the URL into the Microsoft Teams card on this page, enable the channel, select events, and click <strong>Save settings</strong>.",
    ],
    tips: [
      "The URL is tied to one channel. Create separate webhooks for #ops and #security if needed.",
      "If Connectors are disabled by your org admin, ask them to allow Incoming Webhooks or use a Workflow-based URL.",
      "Message format is sent as a Teams MessageCard; DockLog formats alerts automatically.",
    ],
  },
  {
    id: "discord",
    label: "Discord",
    summary: "Channel webhook",
    intro:
      "Discord webhooks post messages to a single text channel. You need <strong>Manage Webhooks</strong> permission on the server.",
    urlExample: "https://discord.com/api/webhooks/…/…",
    docsUrl: "https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks",
    steps: [
      "Open Discord and go to the <strong>server</strong> where you want alerts.",
      "Click the server name → <strong>Server Settings</strong> → <strong>Integrations</strong> → <strong>Webhooks</strong> (or right-click a channel → <strong>Edit Channel</strong> → <strong>Integrations</strong> → <strong>Webhooks</strong>).",
      "Click <strong>New Webhook</strong> or <strong>Create Webhook</strong>.",
      "Set a name (e.g. <em>DockLog</em>), choose the <strong>target channel</strong>, and optionally set an avatar.",
      "Click <strong>Copy Webhook URL</strong>. It starts with <code>https://discord.com/api/webhooks/</code>.",
      "Paste the URL into the Discord card on this page, enable the channel, select events, and click <strong>Save settings</strong>.",
    ],
    tips: [
      "Each webhook posts to one channel only; create multiple webhooks for different channels.",
      "To revoke access, delete the webhook in Discord. DockLog will fail to deliver until you save a new URL.",
      "Do not share the URL publicly; it allows posting messages as the webhook identity.",
    ],
  },
];

const visibleGuides = computed(() => {
  if (!props.channelId) return guides;
  return guides.filter((guide) => guide.id === props.channelId);
});

const isOpen = (id) => {
  if (isStaticPanel.value) return true;
  if (props.modal && !props.channelId) return true;
  return expandAll.value || openPanels[id];
};

const toggle = (id) => {
  if (expandAll.value) {
    expandAll.value = false;
    Object.keys(openPanels).forEach((key) => {
      openPanels[key] = key === id;
    });
    return;
  }
  openPanels[id] = !openPanels[id];
};

const openGuide = (id) => {
  expandAll.value = false;
  Object.keys(openPanels).forEach((key) => {
    openPanels[key] = key === id;
  });
};

defineExpose({ openGuide });
</script>

<style scoped>
.webhook-guide {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.guide-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.guide-header h2 {
  margin: 0 0 0.35rem;
  font-size: 1.1rem;
}

.guide-header p {
  margin: 0;
  font-size: 0.88rem;
  color: var(--text-mute);
  line-height: 1.55;
  max-width: 52rem;
}

.guide-notes {
  padding: 0.85rem 1rem;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--accent) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent) 20%, transparent);
  font-size: 0.82rem;
  color: var(--text-mute);
  line-height: 1.55;
}

.guide-notes p {
  margin: 0;
}

.guide-notes p + p {
  margin-top: 0.45rem;
}

.guide-notes code {
  font-size: 0.8rem;
  padding: 0.1rem 0.35rem;
  border-radius: 4px;
  background: var(--bg-input);
}

.guide-panel {
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.guide-panel-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding: 0.9rem 1rem;
  background: var(--bg-input);
  border: none;
  cursor: pointer;
  text-align: left;
  color: var(--text-main);
}

.guide-panel-toggle:hover {
  background: color-mix(in srgb, var(--accent) 6%, var(--bg-input));
}

.guide-panel-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.guide-panel-icon.slack { background: rgba(224, 30, 90, 0.12); }
.guide-panel-icon.teams { background: rgba(0, 120, 212, 0.12); }
.guide-panel-icon.discord { background: rgba(88, 101, 242, 0.12); }

.guide-panel-title {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.guide-panel-title strong {
  font-size: 0.95rem;
}

.guide-panel-title small {
  font-size: 0.8rem;
  color: var(--text-mute);
  font-weight: 500;
}

.chevron {
  font-size: 1.4rem;
  line-height: 1;
  color: var(--text-mute);
  transform: rotate(90deg);
  transition: transform 0.2s ease;
}

.chevron.open {
  transform: rotate(-90deg);
}

.guide-panel-body {
  padding: 1rem 1.1rem 1.15rem;
  border-top: 1px solid var(--border);
  font-size: 0.88rem;
  line-height: 1.6;
  color: var(--text-main);
}

.guide-intro {
  margin: 0 0 0.85rem;
  color: var(--text-mute);
}

.guide-steps {
  margin: 0 0 1rem;
  padding-left: 1.25rem;
}

.guide-steps li {
  margin-bottom: 0.55rem;
}

.guide-steps :deep(a) {
  color: var(--accent);
  text-decoration: underline;
}

.guide-steps :deep(code) {
  font-size: 0.8rem;
  padding: 0.1rem 0.3rem;
  border-radius: 4px;
  background: var(--bg-input);
}

.guide-url-box {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 0.75rem 0.85rem;
  border-radius: var(--radius-md);
  background: var(--bg-input);
  border: 1px solid var(--border);
  margin-bottom: 0.85rem;
}

.guide-url-label {
  font-size: 0.72rem;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--text-mute);
}

.guide-url-box code {
  font-size: 0.78rem;
  word-break: break-all;
}

.guide-tips {
  margin: 0 0 0.5rem;
  padding-left: 1.15rem;
  color: var(--text-mute);
  font-size: 0.82rem;
}

.guide-tips li {
  margin-bottom: 0.35rem;
}

.guide-docs-link {
  font-size: 0.85rem;
  font-weight: 600;
  color: var(--accent);
  text-decoration: none;
}

.guide-docs-link:hover {
  text-decoration: underline;
}

.webhook-guide.modal {
  background: transparent;
  border: none;
  border-radius: 0;
  padding: 0;
  gap: 0.85rem;
}

.guide-panel-static-head {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  padding-bottom: 0.25rem;
}

.guide-panel-body.no-border {
  border-top: none;
  padding: 0;
}

.guide-panel.is-static {
  border: none;
  border-radius: 0;
  overflow: visible;
}

.guide-panel.is-static + .guide-panel.is-static {
  margin-top: 0.5rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--border);
}

.mono {
  font-family: "JetBrains Mono", monospace;
}

.wide {
  grid-column: 1 / -1;
}
</style>
