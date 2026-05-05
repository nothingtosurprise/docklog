<template>
  <div class="admin-page">
    <div
      :class="[
        'sidebar-backdrop',
        { active: isCompact && sharedState.adminSidebarOpen },
      ]"
      @click="closeSidebar"
    ></div>

    <div class="admin-content-wrapper">
      <button
        v-if="!isCompact && !sharedState.adminSidebarOpen"
        class="sidebar-expand-btn"
        @click="sharedState.adminSidebarOpen = true"
        title="Expand Sidebar"
      >
        <svg
          viewBox="0 0 24 24"
          width="16"
          height="16"
          fill="none"
          stroke="currentColor"
          stroke-width="3"
        >
          <polyline points="9 18 15 12 9 6"></polyline>
        </svg>
      </button>

      <aside
        :class="[
          'sidebar',
          'glass',
          { collapsed: !sharedState.adminSidebarOpen },
        ]"
      >
        <div class="sidebar-content">
          <div class="sidebar-section">
            <div
              style="
                display: flex;
                justify-content: space-between;
                align-items: center;
                margin-bottom: 1rem;
              "
            >
              <div class="section-label" style="margin-bottom: 0">
                SYSTEM OVERVIEW
              </div>
              <button
                v-if="!isCompact"
                @click="sharedState.adminSidebarOpen = false"
                class="return-link"
                style="
                  padding: 0.4rem;
                  border-radius: 8px;
                  width: 32px;
                  height: 32px;
                "
                title="Collapse Sidebar"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="14"
                  height="14"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                >
                  <polyline points="15 18 9 12 15 6"></polyline>
                </svg>
              </button>
            </div>
            <div class="stats-pills">
              <div class="stat-pill glass">
                <span class="p-val">{{ totalContainers }}</span>
                <span class="p-lbl">TOTAL</span>
              </div>
              <div class="stat-pill glass">
                <span class="p-val">{{ runningContainers }}</span>
                <span class="p-lbl">ACTIVE</span>
              </div>
            </div>
          </div>

          <div class="sidebar-spacer"></div>

          <div class="sidebar-footer">
            <div class="admin-notice glass">
              <div class="notice-icon">
                <svg
                  viewBox="0 0 24 24"
                  width="16"
                  height="16"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
                >
                  <circle cx="12" cy="12" r="10"></circle>
                  <line x1="12" y1="16" x2="12" y2="12"></line>
                  <line x1="12" y1="8" x2="12.01" y2="8"></line>
                </svg>
              </div>
              <div class="notice-text">
                <strong>Admin Mode</strong>
                <span>Session is protected</span>
              </div>
            </div>
            <router-link to="/dashboard" class="return-link glass">
              <svg
                viewBox="0 0 24 24"
                width="18"
                height="18"
                stroke="currentColor"
                stroke-width="2.5"
                fill="none"
              >
                <polyline points="15 18 9 12 15 6"></polyline>
              </svg>
              Back to Monitoring
            </router-link>
          </div>
        </div>
      </aside>

      <main class="main-view">
        <AdminPanel :token="token" @close="$router.push('/dashboard')" />
      </main>
    </div>


  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import AdminPanel from "../components/AdminPanel.vue";
import { secureStorage } from "../utils/storage";
import { sharedState, fetchSystemStats } from "../utils/sharedState";

const router = useRouter();
const token = secureStorage.getItem("token");
const totalContainers = ref(0);
const runningContainers = ref(0);
const isCompact = ref(window.innerWidth <= 1024);

let systemStatsInterval = null;
let resizeHandler = null;

const fetchStats = async () => {
  try {
    const res = await fetch("/api/containers", {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (res.ok) {
      const data = await res.json();
      totalContainers.value = data.length;
      runningContainers.value = data.filter(
        (c) => c.state === "running",
      ).length;
    }
  } catch (err) {
    console.error(err);
  }
};

const closeSidebar = () => {
  sharedState.adminSidebarOpen = false;
};

onMounted(async () => {
  resizeHandler = () => {
    const compact = window.innerWidth <= 1024;
    isCompact.value = compact;
    sharedState.adminSidebarOpen = !compact;
  };
  window.addEventListener("resize", resizeHandler);
  fetchStats();
  await fetchSystemStats();
  systemStatsInterval = setInterval(fetchSystemStats, 5000);
});

onUnmounted(() => {
  if (systemStatsInterval) clearInterval(systemStatsInterval);
  if (resizeHandler) window.removeEventListener("resize", resizeHandler);
});
</script>

<style scoped>
.admin-page {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-width: 0;
}

/* Admin Specific Layout */
.admin-content-wrapper {
  display: flex;
  flex: 1;
  overflow: hidden;
  min-width: 0;
}
.sidebar {
  width: 280px;
  min-width: 280px;
  height: calc(100vh - 72px);
  background: var(--bg-sidebar);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
  position: relative;
  z-index: 10;
  overflow: hidden;
}
.sidebar.collapsed {
  width: 0;
  min-width: 0;
  border-right: none;
}
.sidebar-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 1.5rem 1.25rem;
}
.sidebar-section {
  margin-bottom: 1.5rem;
}
.section-label {
  font-size: 0.65rem;
  font-weight: 900;
  color: var(--text-mute);
  letter-spacing: 0.12em;
  margin-bottom: 1rem;
  padding: 0 0.5rem;
}
.stats-pills {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.6rem;
}
.stat-pill {
  padding: 1rem;
  border-radius: 14px;
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  transition: all 0.2s;
}
.stat-pill:hover {
  transform: translateY(-2px);
  border-color: var(--text-mute);
}
.p-val {
  font-size: 1.25rem;
  font-weight: 900;
  color: var(--text-main);
  font-family: var(--font-mono);
}
.p-lbl {
  font-size: 0.6rem;
  font-weight: 850;
  color: var(--text-mute);
}
.sidebar-spacer {
  flex: 1;
}
.sidebar-footer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.admin-notice {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  background: rgba(99, 102, 241, 0.05);
  border: 1px solid rgba(99, 102, 241, 0.1);
}
.notice-icon {
  width: 24px;
  height: 24px;
  border-radius: 8px;
  background: var(--accent);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.notice-icon svg {
  width: 14px;
  height: 14px;
}
.notice-text {
  display: flex;
  flex-direction: column;
}
.notice-text strong {
  font-size: 0.8rem;
  font-weight: 800;
  color: var(--text-main);
}
.notice-text span {
  font-size: 0.65rem;
  color: var(--text-dim);
}
.return-link {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  padding: 0.85rem;
  border-radius: 14px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-main);
  text-decoration: none;
  font-weight: 800;
  font-size: 0.85rem;
  transition: all 0.2s;
}
.return-link svg {
  width: 16px;
  height: 16px;
}
.return-link:hover {
  transform: translateY(-2px);
  background: var(--bg-card);
  border-color: var(--text-mute);
}
.main-view {
  flex: 1;
  overflow: hidden;
  background: var(--bg-main);
  position: relative;
  min-width: 0;
}

.sidebar-toggle {
  display: none;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1024px) {
  .admin-content-wrapper {
    flex-direction: column;
  }
  .sidebar-toggle {
    display: inline-flex;
    position: relative;
    top: auto;
    left: auto;
    z-index: 1;
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.65rem 0.9rem;
    border-radius: 999px;
    border: 1px solid var(--border);
    background: var(--bg-sidebar);
    color: var(--text-main);
    font-size: 0.8rem;
    font-weight: 800;
    box-shadow: 0 10px 30px var(--shadow);
    margin: 0.75rem 0 0.75rem 0.75rem;
  }
  .sidebar {
    position: fixed;
    top: 72px;
    left: 0;
    bottom: 0;
    width: min(320px, 88vw);
    min-width: 0;
    height: calc(100vh - 72px);
    border-right: 1px solid var(--border);
    border-bottom: none;
    transform: translateX(-102%);
    transition: transform 0.28s cubic-bezier(0.23, 1, 0.32, 1);
    z-index: 1650;
  }
  .sidebar:not(.collapsed) {
    transform: translateX(0);
  }
  .sidebar-backdrop.active {
    z-index: 1600;
  }
  .sidebar-content {
    padding: 1rem;
    min-height: 0;
  }
  .stats-pills {
    grid-template-columns: repeat(2, 1fr);
  }
  .sidebar-spacer {
    display: none;
  }
  .stats-pills {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 480px) {
  .sidebar-toggle {
    margin: 0.5rem 0 0.5rem 0.5rem;
    padding: 0.6rem 0.8rem;
  }
  .sidebar {
    width: min(300px, 92vw);
  }
}
</style>
