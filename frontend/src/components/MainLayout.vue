<template>
  <div class="app-layout">
    <header class="main-header glass">
      <div class="header-left">
        <!-- Back Button for non-dashboard pages -->
        <router-link
          v-if="route.path !== '/dashboard'"
          to="/dashboard"
          class="back-btn-header"
          data-tooltip="Back to Dashboard"
        >
          <svg
            viewBox="0 0 24 24"
            width="18"
            height="18"
            stroke="currentColor"
            stroke-width="3"
            fill="none"
          >
            <line x1="19" y1="12" x2="5" y2="12"></line>
            <polyline points="12 19 5 12 12 5"></polyline>
          </svg>
        </router-link>

        <router-link to="/dashboard" class="logo-link">
          <div class="logo-area">
            <div class="logo-box">
              <img src="/logo-icon.png?v=2" alt="DockLog" />
            </div>
            <h1>DockLog</h1>
          </div>
        </router-link>

        <button
          v-if="route.path !== '/health'"
          class="nav-icon-btn glass dashboard-menu-btn"
          @click="toggleDashboardSidebar"
          :data-tooltip="
            route.path === '/dashboard'
              ? 'Toggle Resources'
              : route.path === '/admin'
                ? 'Toggle Admin Menu'
                : 'Open Menu'
          "
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <line x1="4" y1="7" x2="20" y2="7"></line>
            <line x1="4" y1="12" x2="20" y2="12"></line>
            <line x1="4" y1="17" x2="20" y2="17"></line>
          </svg>
        </button>

        <div class="divider"></div>

        <!-- Global Search (only on dashboard) -->
        <div v-if="route.path === '/dashboard'" class="search-wrapper glass">
          <svg
            class="search-icon"
            viewBox="0 0 24 24"
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
            placeholder="Search resources..."
            class="search-input"
          />
        </div>
      </div>

      <div class="header-right">
        <div v-if="sharedState.systemStats" class="header-stats-group">
          <div class="stat-box glass">
            <div class="stat-label">SYS CPU</div>
            <div
              class="stat-value"
              :style="{ color: getStatColor(sharedState.systemStats.cpu) }"
            >
              {{ Number(sharedState.systemStats.cpu || 0).toFixed(1) }}%
            </div>
          </div>
          <div class="stat-box glass">
            <div class="stat-label">SYS MEM</div>
            <div class="stat-value">
              {{ Number(sharedState.systemStats.usedMemGB || 0).toFixed(1) }} GB
            </div>
          </div>
        </div>

        <button
          class="nav-icon-btn glass desktop-only"
          @click="toggleTheme"
          data-tooltip="Toggle Theme"
        >
          <svg
            v-if="sharedState.theme === 'dark'"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <circle cx="12" cy="12" r="5"></circle>
            <line x1="12" y1="1" x2="12" y2="3"></line>
            <line x1="12" y1="21" x2="12" y2="23"></line>
            <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
            <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
            <line x1="1" y1="12" x2="3" y2="12"></line>
            <line x1="21" y1="12" x2="23" y2="12"></line>
            <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
            <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
          </svg>
          <svg
            v-else
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
          >
            <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
          </svg>
        </button>

        <div
          class="user-profile glass"
          @click.stop="showUserMenu = !showUserMenu"
        >
          <div class="avatar">{{ userInitial }}</div>
          <span class="username desktop-only">{{
            sharedState.currentUser?.username
          }}</span>
          <svg
            :class="['chevron', { open: showUserMenu }]"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <polyline points="6 9 12 15 18 9"></polyline>
          </svg>

          <Transition name="fade">
            <div
              v-if="showUserMenu"
              class="user-menu glass shadow-xl"
              @click.stop
            >
              <div class="menu-header">
                <span class="role-badge">{{
                  sharedState.currentUser?.is_admin ? "ADMIN" : "USER"
                }}</span>
              </div>
              <button @click="openPasswordModal" class="menu-item">
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <rect
                    x="3"
                    y="11"
                    width="18"
                    height="11"
                    rx="2"
                    ry="2"
                  ></rect>
                  <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
                </svg>
                Change Password
              </button>

              <router-link
                v-if="sharedState.currentUser?.is_admin"
                to="/admin"
                class="menu-item"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                >
                  <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"></path>
                  <circle cx="9" cy="7" r="4"></circle>
                  <path d="M23 21v-2a4 4 0 0 0-3-3.87"></path>
                  <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
                </svg>
                Manage Users
              </router-link>

              <router-link
                v-if="sharedState.currentUser?.is_admin"
                to="/health"
                class="menu-item"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>
                </svg>
                System Health
              </router-link>

              <div class="menu-divider"></div>
              <button @click="logout" class="menu-item logout">
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"></path>
                  <polyline points="16 17 21 12 16 7"></polyline>
                  <line x1="21" y1="12" x2="9" y2="12"></line>
                </svg>
                Sign Out
              </button>
            </div>
          </Transition>
        </div>
      </div>
    </header>

    <div class="layout-body">
      <slot />
    </div>

    <!-- Global Toast Notification -->
    <Transition name="slide-up">
      <div
        v-if="sharedState.toast.visible"
        :class="['toast-notification', sharedState.toast.type]"
      >
        <div class="toast-icon">
          <svg
            v-if="sharedState.toast.type === 'success'"
            viewBox="0 0 24 24"
            width="22"
            height="22"
            stroke="currentColor"
            stroke-width="2.5"
            fill="none"
          >
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"></path>
            <polyline points="22 4 12 14.01 9 11.01"></polyline>
          </svg>
          <svg
            v-else
            viewBox="0 0 24 24"
            width="22"
            height="22"
            stroke="currentColor"
            stroke-width="2.5"
            fill="none"
          >
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="15" y1="9" x2="9" y2="15"></line>
            <line x1="9" y1="9" x2="15" y2="15"></line>
          </svg>
        </div>
        <div class="toast-content">
          <h4>{{ sharedState.toast.title }}</h4>
          <p>{{ sharedState.toast.message }}</p>
        </div>
        <button class="toast-close" @click="sharedState.toast.visible = false">
          <svg
            viewBox="0 0 24 24"
            width="18"
            height="18"
            stroke="currentColor"
            stroke-width="2.5"
            fill="none"
          >
            <line x1="18" y1="6" x2="6" y2="18"></line>
            <line x1="6" y1="6" x2="18" y2="18"></line>
          </svg>
        </button>
      </div>
    </Transition>

    <!-- Password Change Modal -->
    <Transition name="fade">
      <div v-if="sharedState.showPasswordModal" class="modal-overlay glass">
        <div class="modal-content glass shadow-2xl">
          <div class="modal-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2.5"
            >
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
          </div>
          <h3>
            {{
              sharedState.forcePasswordChange
                ? "Welcome to DockLog"
                : "Update Security"
            }}
          </h3>
          <p v-if="sharedState.forcePasswordChange" class="force-text-new">
            For security, please set a new password for your account to
            continue.
          </p>
          <div class="input-group">
            <input
              type="password"
              v-model="newPassword"
              placeholder="Enter new password"
              class="glass-input"
              @input="passwordError = ''"
            />
          </div>
          <div class="input-group">
            <input
              type="password"
              v-model="confirmPassword"
              placeholder="Confirm new password"
              class="glass-input"
              @input="passwordError = ''"
            />
            <p v-if="passwordError" class="input-error">{{ passwordError }}</p>
          </div>
          <div class="modal-actions">
            <button
              v-if="!sharedState.forcePasswordChange"
              @click="sharedState.showPasswordModal = false"
              class="modal-btn cancel"
            >
              Cancel
            </button>
            <button @click="updatePassword" class="modal-btn confirm">
              Update Password
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  sharedState,
  fetchCurrentUser,
  fetchSystemStats,
  showToast,
} from "../utils/sharedState";
import { secureStorage } from "../utils/storage";

const route = useRoute();
const router = useRouter();
const showUserMenu = ref(false);
let statsInterval = null;
let userInterval = null;

const newPassword = ref("");
const confirmPassword = ref("");
const passwordError = ref("");

const userInitial = computed(
  () => sharedState.currentUser?.username?.charAt(0).toUpperCase() || "A",
);

const toggleTheme = () => {
  sharedState.theme = sharedState.theme === "dark" ? "light" : "dark";
  secureStorage.setItem("theme", sharedState.theme);
  document.documentElement.setAttribute("data-theme", sharedState.theme);
};

const getStatColor = (val) => {
  const v = parseFloat(val);
  if (v > 80) return "var(--error)";
  if (v > 50) return "#f59e0b";
  return "var(--success)";
};

const toggleDashboardSidebar = () => {
  if (route.path === "/dashboard") {
    sharedState.dashboardSidebarOpen = !sharedState.dashboardSidebarOpen;
    return;
  }

  if (route.path === "/admin") {
    sharedState.adminSidebarOpen = !sharedState.adminSidebarOpen;
    return;
  }

  sharedState.dashboardSidebarOpen = true;
  router.push("/dashboard");
};

const logout = () => {
  secureStorage.removeItem("token");
  secureStorage.removeItem("user");
  sharedState.currentUser = null;
  sharedState.showPasswordModal = false;
  sharedState.forcePasswordChange = false;
  router.push("/login");
};

const openPasswordModal = () => {
  newPassword.value = "";
  confirmPassword.value = "";
  passwordError.value = "";
  sharedState.showPasswordModal = true;
  showUserMenu.value = false;
};

const updatePassword = async () => {
  if (newPassword.value.length < 6) {
    passwordError.value = "Password must be at least 6 characters";
    return;
  }
  if (newPassword.value !== confirmPassword.value) {
    passwordError.value = "Passwords do not match";
    return;
  }

  try {
    const token = secureStorage.getItem("token");
    const formData = new FormData();
    formData.append("password", newPassword.value);
    const res = await fetch("/api/user/change-password", {
      method: "POST",
      headers: { Authorization: `Bearer ${token}` },
      body: formData,
    });
    if (res.ok) {
      sharedState.showPasswordModal = false;
      showToast("Success", "Password updated successfully", "success");
      // If forced, clear the flag
      if (sharedState.forcePasswordChange) {
        sharedState.forcePasswordChange = false;
      }
    } else {
      const data = await res.json();
      passwordError.value = data.error || "Failed to update password";
    }
  } catch (err) {
    passwordError.value = "Connection error";
  }
};

const handleGlobalClick = () => {
  showUserMenu.value = false;
};

onMounted(async () => {
  window.addEventListener("click", handleGlobalClick);
  const session = await fetchCurrentUser();
  if (session.status === "forbidden") {
    router.replace("/login");
    return;
  }
  sharedState.forcePasswordChange =
    sharedState.currentUser?.password_changed === false;
  sharedState.showPasswordModal =
    sharedState.currentUser?.password_changed === false;
  await fetchSystemStats();
  statsInterval = setInterval(fetchSystemStats, 5000);
  userInterval = setInterval(async () => {
    const current = await fetchCurrentUser();
    if (current.status === "forbidden") {
      clearInterval(userInterval);
      if (statsInterval) clearInterval(statsInterval);
      router.replace("/login");
    }
  }, 2000);

  window.addEventListener("online", handleOnline);
  window.addEventListener("offline", handleOffline);
});

const handleOnline = () => {
  showToast("Back Online", "Your internet connection has been restored", "success");
};

const handleOffline = () => {
  showToast("Offline", "You are currently disconnected", "error");
};

watch(
  () => sharedState.currentUser?.password_changed,
  (passwordChanged) => {
    if (passwordChanged === false) {
      sharedState.forcePasswordChange = true;
      sharedState.showPasswordModal = true;
    }
  },
);

onUnmounted(() => {
  window.removeEventListener("click", handleGlobalClick);
  window.removeEventListener("online", handleOnline);
  window.removeEventListener("offline", handleOffline);
  if (statsInterval) clearInterval(statsInterval);
  if (userInterval) clearInterval(userInterval);
});
</script>

<style scoped>
.app-layout {
  height: 100vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.layout-body {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
  min-height: 0;
}

.main-header {
  height: 72px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 2rem;
  position: sticky;
  top: 0;
  z-index: 2000;
}

.header-left,
.header-right {
  display: flex;
  align-items: center;
  gap: 1.25rem;
  flex: 1;
  min-width: 0;
}

.header-right {
  justify-content: flex-end;
  margin-left: auto;
}

.header-stats-group {
  display: flex;
  gap: 1rem;
}

.stat-box {
  padding: 0.5rem 1rem;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  min-width: 100px;
}

.stat-label {
  font-size: 0.65rem;
  font-weight: 800;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.stat-value {
  font-size: 0.9rem;
  font-weight: 900;
  font-family: var(--font-mono);
}

.logo-link {
  text-decoration: none;
}

.logo-area {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.logo-box {
  width: 36px;
  height: 36px;
  background: var(--accent);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 8px 16px rgba(99, 102, 241, 0.2);
}

.logo-box img {
  width: 22px;
  height: 22px;
  filter: brightness(0) invert(1);
}

.logo-area h1 {
  font-size: 1.25rem;
  font-weight: 950;
  color: var(--text-main);
  letter-spacing: -0.04em;
  margin: 0;
}

.divider {
  width: 1px;
  height: 24px;
  background: var(--border);
}

.search-wrapper {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 1rem;
  border-radius: 14px;
  width: 280px;
  transition: all 0.3s;
  min-width: 0;
}

.search-wrapper:focus-within {
  width: 320px;
  border-color: var(--accent);
  background: var(--bg-card);
}

.dashboard-menu-btn {
  display: none;
}

.search-icon {
  width: 18px;
  height: 18px;
  color: var(--text-mute);
}

.search-input {
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 600;
  width: 100%;
  outline: none;
}

.back-btn-header {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-main);
  transition: all 0.2s;
  text-decoration: none;
}

.back-btn-header:hover {
  transform: translateX(-4px);
  background: var(--bg-card);
  border-color: var(--accent);
}

@media (max-width: 1024px) {
  .main-header {
    padding: 0 1rem;
    gap: 0.75rem;
  }

  .dashboard-menu-btn {
    display: flex;
  }

  .header-left,
  .header-right {
    gap: 0.75rem;
  }

  .header-stats-group {
    display: none;
  }

  .search-wrapper {
    width: 220px;
  }

  .search-wrapper:focus-within {
    width: 240px;
  }
}

@media (max-width: 900px) {
  .logo-area h1,
  .divider {
    display: none;
  }

  .header-left {
    gap: 0.6rem;
  }

  .search-wrapper {
    width: 180px;
  }

  .search-wrapper:focus-within {
    width: 200px;
  }
}

@media (max-width: 768px) {
  .header-left {
    gap: 0.5rem;
  }

  .search-wrapper {
    display: none;
  }

  .header-right {
    gap: 0.5rem;
  }
}
</style>
