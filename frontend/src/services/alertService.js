import { apiFetch } from '../utils/apiFetch';

export async function fetchAlerts() {
  const res = await apiFetch('/api/admin/alerts');
  if (!res.ok) throw new Error('Failed to load alerts');
  return res.json();
}

export async function fetchAlertHistory(limit = 100) {
  const res = await apiFetch(`/api/admin/alerts/history?limit=${limit}`);
  if (!res.ok) throw new Error('Failed to load alert history');
  return res.json();
}

export async function createAlertRule(payload) {
  const res = await apiFetch('/api/admin/alerts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Failed to create alert rule');
  return data;
}

export async function updateAlertRule(id, payload) {
  const res = await apiFetch(`/api/admin/alerts/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Failed to update alert rule');
  return data;
}

export async function deleteAlertRule(id) {
  const res = await apiFetch(`/api/admin/alerts/${id}`, { method: 'DELETE' });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error || 'Failed to delete alert rule');
  }
}

export async function createAlertFromTemplate(ruleKey, channelIds) {
  const res = await apiFetch(`/api/admin/alerts/templates/${ruleKey}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ channel_ids: channelIds }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Failed to create from template');
  return data;
}

export async function testAlertRule(ruleId) {
  const res = await apiFetch('/api/admin/alerts/test', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ rule_id: ruleId }),
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new Error(data.error || 'Failed to send test alert');
  return data;
}

export async function fetchNotifications() {
  const res = await apiFetch('/api/admin/notifications');
  if (!res.ok) throw new Error('Failed to load notification channels');
  return res.json();
}
