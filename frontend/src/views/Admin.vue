<template>
  <div class="page-view admin-view animate-fade-in">
    <section class="page-hero">
      <div class="page-hero-body">
        <div class="page-hero-copy">
          <span class="page-hero-eyebrow">Administration</span>
          <h1>Staff management</h1>
          <p class="page-hero-sub">Manage user accounts, roles, and container permissions.</p>
        </div>
        <div class="page-hero-actions">
          <button @click="userManagerRef?.openCreateModal()" class="page-btn primary">
            <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="3">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            Add user
          </button>
        </div>
      </div>
      <div class="page-hero-mesh" aria-hidden="true"></div>
    </section>

    <section class="page-metrics animate-slide-up">
      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon">
            <AppIcon name="users" />
          </div>
          <span class="badge badge-dim">Staff</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Accounts</span>
          <span class="stat-value">{{ staffUsersCount }}</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon success">
            <AppIcon name="checkCircle" />
          </div>
          <span class="badge badge-success">Active</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">System status</span>
          <span class="stat-value">Online</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon">
            <AppIcon name="shield" />
          </div>
          <span class="badge badge-dim">Audit</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Audit events</span>
          <span class="stat-value">{{ auditLogsCount || "-" }}</span>
        </div>
      </div>

      <div class="page-metric-card">
        <div class="stat-header">
          <div class="stat-icon warning">
            <AppIcon name="alert" />
          </div>
          <span class="badge badge-warning">Alerts</span>
        </div>
        <div class="stat-content">
          <span class="stat-label">Security alerts</span>
          <span class="stat-value">0</span>
        </div>
      </div>
    </section>

    <section class="page-panel">
      <UserManager ref="userManagerRef" :token="token" embedded @update-count="handleStaffCountUpdate" />
    </section>
  </div>
</template>

<script setup>
import { ref } from "vue";
import AppIcon from "../components/AppIcon.vue";
import UserManager from "../components/UserManager.vue";
import { secureStorage } from "../utils/storage";

const userManagerRef = ref(null);
const token = secureStorage.getItem("token");
const staffUsersCount = ref(0);
const auditLogsCount = ref(0);

const handleStaffCountUpdate = (count) => {
  staffUsersCount.value = count;
};
</script>
