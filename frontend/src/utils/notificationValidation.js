const PLACEHOLDER_MARKERS = [
  /\.\.\.$/,
  /\/\.\.\./,
  /xxx+/i,
  /your[-_]?webhook/i,
  /paste[-_]?here/i,
  /example\.com/i,
];

const WEBHOOK_HOST_HINTS = {
  slack: ["hooks.slack.com", "hooks.slack-gov.com"],
  teams: ["webhook.office.com", "outlook.office.com"],
  discord: ["discord.com", "discordapp.com"],
};

export function validateWebhookUrl(raw, channelType) {
  const url = (raw || "").trim();
  if (!url) {
    return { valid: false, message: "Webhook URL is required" };
  }
  if (!url.startsWith("https://")) {
    return { valid: false, message: "URL must start with https://" };
  }
  if (url.length > 2048) {
    return { valid: false, message: "URL is too long (max 2048 characters)" };
  }
  for (const pattern of PLACEHOLDER_MARKERS) {
    if (pattern.test(url)) {
      return { valid: false, message: "Replace the placeholder with your real webhook URL" };
    }
  }

  let hostname = "";
  try {
    hostname = new URL(url).hostname.toLowerCase();
  } catch {
    return { valid: false, message: "Enter a valid webhook URL" };
  }

  const allowedHosts = WEBHOOK_HOST_HINTS[channelType];
  if (allowedHosts && !allowedHosts.some((host) => hostname === host || hostname.endsWith(`.${host}`))) {
    const label = channelType === "slack" ? "Slack" : channelType === "teams" ? "Teams" : "Discord";
    return {
      valid: false,
      message: `This does not look like a ${label} webhook URL`,
    };
  }

  return { valid: true, message: "" };
}

export function channelHasStoredWebhook(type, isConfigured) {
  return isConfigured(type);
}

export function channelWebhookValue(channelTypeMeta, entry) {
  const field = (channelTypeMeta?.config_fields || []).find((f) => f.key === "webhook_url");
  if (!field) return "";
  return (entry?.config?.[field.key] || "").trim();
}

export function channelHasWebhook(channelTypeMeta, entry, isConfigured, type) {
  if (!entry || entry.clear) return false;
  const typed = channelWebhookValue(channelTypeMeta, entry);
  if (typed) return true;
  return isConfigured(type);
}

export function channelHasAnyEvent(entry) {
  if (!entry?.events) return false;
  return (
    entry.events.notify_container_actions ||
    entry.events.notify_security_events ||
    entry.events.notify_admin_actions ||
    entry.events.notify_health_events
  );
}

export function validateChannelEntry(channelTypeMeta, entry, isConfigured) {
  const issues = [];
  const type = channelTypeMeta.type;
  const label = channelTypeMeta.label;

  if (!entry) return issues;

  if (entry.clear) return issues;

  const webhookInput = channelWebhookValue(channelTypeMeta, entry);
  const stored = isConfigured(type);
  const active = entry.enabled;

  if (webhookInput) {
    const check = validateWebhookUrl(webhookInput, type);
    if (!check.valid) {
      issues.push({ scope: type, field: "webhook_url", message: `${label}: ${check.message}` });
    }
  } else if (active && !stored) {
    issues.push({
      scope: type,
      field: "webhook_url",
      message: `${label}: add a webhook URL or disable this channel`,
    });
  }

  if (active && channelHasWebhook(channelTypeMeta, entry, isConfigured, type) && !channelHasAnyEvent(entry)) {
    issues.push({
      scope: type,
      field: "events",
      message: `${label}: select at least one event type`,
    });
  }

  return issues;
}

export function validateNotificationSettings({
  enabled,
  channelTypes,
  channelForms,
  isConfigured,
}) {
  const issues = [];
  const fieldErrors = {};

  for (const channelType of channelTypes || []) {
    if (!channelType.available) continue;
    const entry = channelForms[channelType.type];
    const channelIssues = validateChannelEntry(channelType, entry, isConfigured);
    for (const issue of channelIssues) {
      issues.push(issue.message);
      if (!fieldErrors[issue.scope]) fieldErrors[issue.scope] = {};
      fieldErrors[issue.scope][issue.field] = issue.message.replace(/^[^:]+:\s*/, "");
    }
  }

  const channelIssuesOnly = [...new Set(issues)];

  if (!enabled) {
    const activeWithWebhook = (channelTypes || []).filter((ct) => {
      if (!ct.available) return false;
      const entry = channelForms[ct.type];
      return entry?.enabled && !entry.clear && channelHasWebhook(ct, entry, isConfigured, ct.type);
    });
    const deliveryIssues = [];
    if (activeWithWebhook.length > 0) {
      deliveryIssues.push("Turn on Delivery to send live alerts to your active channels");
    }
    return {
      valid: channelIssuesOnly.length === 0,
      issues: [...channelIssuesOnly, ...deliveryIssues],
      fieldErrors,
      ready: channelIssuesOnly.length === 0,
    };
  }

  const activeWithWebhook = (channelTypes || []).filter((ct) => {
    if (!ct.available) return false;
    const entry = channelForms[ct.type];
    return entry?.enabled && !entry.clear && channelHasWebhook(ct, entry, isConfigured, ct.type);
  });

  if (activeWithWebhook.length === 0) {
    issues.push("Enable at least one channel with a valid webhook URL");
  }

  const routesEvents = activeWithWebhook.some((ct) => {
    const entry = channelForms[ct.type];
    return channelHasAnyEvent(entry);
  });

  if (activeWithWebhook.length > 0 && !routesEvents) {
    issues.push("Select at least one event type on an active channel");
  }

  const unique = [...new Set(issues)];
  return {
    valid: unique.length === 0,
    issues: unique,
    fieldErrors,
    ready: unique.length === 0,
  };
}

export function countActiveChannels(channelTypes, channelForms, isConfigured) {
  let count = 0;
  for (const ct of channelTypes || []) {
    if (!ct.available) continue;
    const entry = channelForms[ct.type];
    if (entry?.enabled && !entry.clear && channelHasWebhook(ct, entry, isConfigured, ct.type)) {
      count += 1;
    }
  }
  return count;
}

export function countEventRoutes(channelTypes, channelForms, isConfigured) {
  let count = 0;
  for (const ct of channelTypes || []) {
    if (!ct.available) continue;
    const entry = channelForms[ct.type];
    if (!entry?.enabled || entry.clear) continue;
    if (!channelHasWebhook(ct, entry, isConfigured, ct.type)) continue;
    if (!channelHasAnyEvent(entry)) continue;
    count += 1;
  }
  return count;
}

export function channelStatus(channelTypeMeta, entry, isConfigured) {
  const type = channelTypeMeta.type;
  if (!entry || entry.clear) return "disconnected";
  if (!channelHasWebhook(channelTypeMeta, entry, isConfigured, type)) return "incomplete";
  if (!entry.enabled) return "paused";
  if (!channelHasAnyEvent(entry)) return "no-events";
  return "active";
}
