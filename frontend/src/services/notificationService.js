import { apiFetch } from "../utils/apiFetch";
import { readApiError } from "../utils/authSession";

export async function fetchNotificationSettings() {
  const res = await apiFetch("/api/admin/notifications");
  if (!res.ok) {
    throw await readApiError(res, "Failed to load notification settings");
  }
  return res.json();
}

export async function saveNotificationSettings(payload) {
  const res = await apiFetch("/api/admin/notifications", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) {
    throw await readApiError(res, "Failed to save settings");
  }
  return res.json();
}

export async function testNotification(target = "all") {
  const res = await apiFetch("/api/admin/notifications/test", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ target }),
  });
  if (!res.ok) {
    throw await readApiError(res, "Test failed");
  }
  return res.json();
}
