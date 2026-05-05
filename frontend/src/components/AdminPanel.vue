<template>
  <div class="admin-panel">
    <div class="panel-container">
      <header class="panel-header">
        <div class="header-left">
          <div class="title-group">
            <h1>
              {{ activeTab === "staff" ? "Staff Control" : "Security Audit" }}
            </h1>
            <p>
              {{
                activeTab === "staff"
                  ? "Manage account access and security policies"
                  : "Monitor system actions and security events"
              }}
            </p>
          </div>
        </div>

        <div class="header-center">
          <div class="tab-switcher glass">
            <button
              @click="activeTab = 'staff'"
              :class="['tab-btn', { active: activeTab === 'staff' }]"
            >
              Staff
            </button>
            <button
              @click="activeTab = 'audit'"
              :class="['tab-btn', { active: activeTab === 'audit' }]"
            >
              Audit Logs
            </button>
          </div>
        </div>

        <div class="header-actions">
          <button
            v-if="activeTab === 'staff'"
            @click="showCreateModal = true"
            class="action-btn primary"
          >
            <svg
              viewBox="0 0 24 24"
              width="18"
              height="18"
              stroke="currentColor"
              stroke-width="2.5"
              fill="none"
            >
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <line x1="5" y1="12" x2="19" y2="12"></line>
            </svg>
            New User
          </button>
        </div>
      </header>

      <div v-if="activeTab === 'staff'" class="main-user-card glass">
        <!-- Desktop Table -->
        <table class="refined-table desktop-only">
          <thead>
            <tr>
              <th>User Account</th>
              <th>Role</th>
              <th>Global Rights</th>
              <th>Status</th>
              <th class="text-right">Manage</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in staffUsers" :key="u.id">
              <td>
                <div class="user-cell">
                  <div class="mini-avatar">
                    {{ u.username.charAt(0).toUpperCase() }}
                  </div>
                  <div class="u-meta">
                    <span class="u-name">{{ u.username }}</span>
                    <span class="u-id">ID: #{{ u.id }}</span>
                  </div>
                </div>
              </td>
              <td>
                <span :class="['badge', u.is_admin ? 'admin' : 'staff']">
                  {{ u.is_admin ? "ADMIN" : "STAFF" }}
                </span>
              </td>
              <td>
                <div v-if="!u.is_admin" class="perm-preview">
                  <span v-if="u.can_start" class="p-chip">START</span>
                  <span v-if="u.can_stop" class="p-chip">STOP</span>
                  <span v-if="u.can_restart" class="p-chip">RESTART</span>
                  <span v-if="u.can_delete" class="p-chip">DELETE</span>
                  <span
                    v-if="
                      !u.can_start &&
                      !u.can_stop &&
                      !u.can_restart &&
                      !u.can_delete
                    "
                    class="p-none"
                    >READ-ONLY</span
                  >
                </div>
                <span v-else class="all-access">SYSTEM OWNER</span>
              </td>
              <td>
                <button
                  @click="toggleUserStatus(u)"
                  :class="['status-pill-toggle', { active: u.is_active }]"
                  v-if="!u.is_admin"
                >
                  <span class="dot"></span>
                  {{ u.is_active ? "Active" : "Inactive" }}
                </button>
                <span v-else class="status-pill-toggle active"
                  ><span class="dot"></span> Active</span
                >
              </td>
              <td class="text-right">
                <div class="action-dropdown-wrap" v-if="!u.is_admin">
                  <button
                    @click.stop="toggleMenu(u.id)"
                    :class="['dots-btn', { active: activeMenuId === u.id }]"
                  >
                    <svg
                      viewBox="0 0 24 24"
                      width="20"
                      height="20"
                      stroke="currentColor"
                      stroke-width="2.5"
                      fill="none"
                    >
                      <circle cx="12" cy="12" r="1.5"></circle>
                      <circle cx="12" cy="5" r="1.5"></circle>
                      <circle cx="12" cy="19" r="1.5"></circle>
                    </svg>
                  </button>
                  <Transition name="slide-fade">
                    <div
                      v-if="activeMenuId === u.id"
                      class="dropdown-menu glass"
                    >
                      <button @click="openPermissions(u)" class="menu-item">
                        <svg
                          viewBox="0 0 24 24"
                          width="16"
                          height="16"
                          stroke="currentColor"
                          stroke-width="2"
                          fill="none"
                        >
                          <path
                            d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"
                          ></path>
                        </svg>
                        Edit Permissions
                      </button>
                      <button @click="openPasswordReset(u)" class="menu-item">
                        <svg
                          viewBox="0 0 24 24"
                          width="16"
                          height="16"
                          stroke="currentColor"
                          stroke-width="2"
                          fill="none"
                        >
                          <path
                            d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3L15.5 7.5z"
                          ></path>
                        </svg>
                        Change Password
                      </button>
                      <div class="menu-sep"></div>
                      <button
                        @click="deleteUser(u.id)"
                        class="menu-item danger"
                      >
                        <svg
                          viewBox="0 0 24 24"
                          width="16"
                          height="16"
                          stroke="currentColor"
                          stroke-width="2"
                          fill="none"
                        >
                          <polyline points="3 6 5 6 21 6"></polyline>
                          <path
                            d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                          ></path>
                        </svg>
                        Terminate Account
                      </button>
                    </div>
                  </Transition>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Pagination Controls -->
        <div v-if="totalPages > 1" class="pagination-bar glass">
          <button
            @click="changePage(currentPage - 1)"
            :disabled="currentPage === 1"
            class="p-btn"
          >
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
            Prev
          </button>
          <div class="page-numbers">
            <button
              v-for="p in totalPages"
              :key="p"
              @click="changePage(p)"
              :class="['page-btn', { active: p === currentPage }]"
            >
              {{ p }}
            </button>
          </div>
          <button
            @click="changePage(currentPage + 1)"
            :disabled="currentPage === totalPages"
            class="p-btn"
          >
            Next
            <svg
              viewBox="0 0 24 24"
              width="18"
              height="18"
              stroke="currentColor"
              stroke-width="2.5"
              fill="none"
            >
              <polyline points="9 18 15 12 9 6"></polyline>
            </svg>
          </button>
        </div>

        <!-- Mobile Cards -->
        <div class="mobile-only mobile-card-list">
          <div v-for="u in staffUsers" :key="u.id" class="mobile-card glass">
            <div class="card-header">
              <div class="user-cell">
                <div class="mini-avatar">
                  {{ u.username.charAt(0).toUpperCase() }}
                </div>
                <div class="u-meta">
                  <span class="u-name">{{ u.username }}</span>
                  <span class="u-id">ID: #{{ u.id }}</span>
                </div>
              </div>
              <div class="card-actions" v-if="!u.is_admin">
                <button
                  @click.stop="toggleMenu(u.id)"
                  :class="['dots-btn', { active: activeMenuId === u.id }]"
                >
                  <svg
                    viewBox="0 0 24 24"
                    width="20"
                    height="20"
                    stroke="currentColor"
                    stroke-width="2.5"
                    fill="none"
                  >
                    <circle cx="12" cy="12" r="1.5"></circle>
                    <circle cx="12" cy="5" r="1.5"></circle>
                    <circle cx="12" cy="19" r="1.5"></circle>
                  </svg>
                </button>
                <Transition name="slide-fade">
                  <div
                    v-if="activeMenuId === u.id"
                    class="dropdown-menu glass mobile-dropdown"
                  >
                    <button @click="openPermissions(u)" class="menu-item">
                      <svg
                        viewBox="0 0 24 24"
                        width="16"
                        height="16"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                      >
                        <path
                          d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"
                        ></path>
                      </svg>
                      Edit Permissions
                    </button>
                    <button @click="openPasswordReset(u)" class="menu-item">
                      <svg
                        viewBox="0 0 24 24"
                        width="16"
                        height="16"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                      >
                        <path
                          d="M21 2l-2 2m-7.61 7.61a5.5 5.5 0 1 1-7.778 7.778 5.5 5.5 0 0 1 7.777-7.777zm0 0L15.5 7.5m0 0l3 3L22 7l-3-3L15.5 7.5z"
                        ></path>
                      </svg>
                      Change Password
                    </button>
                    <div class="menu-sep"></div>
                    <button @click="deleteUser(u.id)" class="menu-item danger">
                      <svg
                        viewBox="0 0 24 24"
                        width="16"
                        height="16"
                        stroke="currentColor"
                        stroke-width="2"
                        fill="none"
                      >
                        <polyline points="3 6 5 6 21 6"></polyline>
                        <path
                          d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"
                        ></path>
                      </svg>
                      Terminate Account
                    </button>
                  </div>
                </Transition>
              </div>
            </div>
            <div class="card-body">
              <div class="card-info-row">
                <span class="info-label">Role</span>
                <span :class="['badge', u.is_admin ? 'admin' : 'staff']">
                  {{ u.is_admin ? "ADMIN" : "STAFF" }}
                </span>
              </div>
              <div class="card-info-row">
                <span class="info-label">Rights</span>
                <div v-if="!u.is_admin" class="perm-preview">
                  <span v-if="u.can_start" class="p-chip">START</span>
                  <span v-if="u.can_stop" class="p-chip">STOP</span>
                  <span v-if="u.can_restart" class="p-chip">RESTART</span>
                  <span v-if="u.can_delete" class="p-chip">DELETE</span>
                  <span
                    v-if="
                      !u.can_start &&
                      !u.can_stop &&
                      !u.can_restart &&
                      !u.can_delete
                    "
                    class="p-none"
                    >READ-ONLY</span
                  >
                </div>
                <span v-else class="all-access">SYSTEM OWNER</span>
              </div>
              <div class="card-info-row">
                <span class="info-label">Status</span>
                <button
                  @click="toggleUserStatus(u)"
                  :class="['status-pill-toggle', { active: u.is_active }]"
                  v-if="!u.is_admin"
                >
                  <span class="dot"></span>
                  {{ u.is_active ? "Active" : "Inactive" }}
                </button>
                <span v-else class="status-pill-toggle active"
                  ><span class="dot"></span> Active</span
                >
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="activeTab === 'audit'" class="main-user-card audit-card glass">
        <div class="audit-toolbar">
          <div class="search-box glass">
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              stroke="currentColor"
              stroke-width="2.5"
              fill="none"
            >
              <circle cx="11" cy="11" r="8"></circle>
              <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
            </svg>
            <input
              type="text"
              v-model="auditSearch"
              placeholder="Filter logs..."
            />
          </div>
          <button
            @click="fetchAuditLogs"
            class="refresh-btn glass"
            :disabled="loadingLogs"
          >
            <svg
              :class="{ rotating: loadingLogs }"
              viewBox="0 0 24 24"
              width="16"
              height="16"
              stroke="currentColor"
              stroke-width="2.5"
              fill="none"
            >
              <polyline points="23 4 23 10 17 10"></polyline>
              <path d="M20.49 15a9 9 0 1 1-2.12-9.36L23 10"></path>
            </svg>
          </button>
        </div>

        <!-- Desktop Audit Table -->
        <div class="audit-table-wrap desktop-only">
          <table class="refined-table audit-table">
            <thead>
              <tr>
                <th>Time</th>
                <th>User</th>
                <th>Action</th>
                <th>Resource</th>
                <th>Status</th>
                <th>Message</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in filteredLogs" :key="log.id">
                <td class="time-col">{{ formatAuditTime(log.timestamp) }}</td>
                <td class="user-col">
                  <div class="user-info">
                    <div class="mini-avatar">
                      {{ log.username[0].toUpperCase() }}
                    </div>
                    <span>{{ log.username }}</span>
                  </div>
                </td>
                <td>
                  <span class="action-tag">{{ log.action.toUpperCase() }}</span>
                </td>
                <td class="res-col">{{ log.resource }}</td>
                <td>
                  <span :class="['status-pill', log.status.toLowerCase()]">
                    {{ log.status }}
                  </span>
                </td>
                <td class="msg-col">{{ log.message }}</td>
              </tr>
              <tr v-if="filteredLogs.length === 0">
                <td colspan="6" class="empty-state">No matching logs found.</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Mobile Audit Cards -->
        <div class="mobile-only mobile-audit-list">
          <div
            v-for="log in filteredLogs"
            :key="log.id"
            class="mobile-card glass audit-item"
          >
            <div class="card-header">
              <div class="user-info">
                <div class="mini-avatar">
                  {{ log.username[0].toUpperCase() }}
                </div>
                <div class="u-meta">
                  <span class="u-name">{{ log.username }}</span>
                  <span class="u-id">{{ formatAuditTime(log.timestamp) }}</span>
                </div>
              </div>
              <span :class="['status-pill', log.status.toLowerCase()]">
                {{ log.status }}
              </span>
            </div>
            <div class="card-body">
              <div class="audit-main-action">
                <span class="action-tag">{{ log.action.toUpperCase() }}</span>
                <span class="res-target">{{ log.resource }}</span>
              </div>
              <p class="audit-msg">{{ log.message }}</p>
            </div>
          </div>
          <div v-if="filteredLogs.length === 0" class="empty-state p-8">
            No matching logs found.
          </div>
        </div>
      </div>

      <!-- Modals -->
    </div>
    <TransitionGroup name="fade">
      <!-- Create Modal -->
      <div
        v-if="showCreateModal"
        class="modal-overlay"
        @click.self="showCreateModal = false"
      >
        <div class="modal-box glass shadow-2xl">
          <div class="modal-header">
            <h3>New Staff Member</h3>
            <p>Set credentials and base permissions</p>
          </div>
          <form @submit.prevent="createUser" class="modal-form">
            <div class="form-row">
              <div class="input-field">
                <label>Username</label>
                <input
                  v-model="newUser.username"
                  type="text"
                  placeholder="e.g. john_doe"
                  required
                />
              </div>
              <div class="input-field">
                <label>Initial Password</label>
                <input
                  v-model="newUser.password"
                  type="password"
                  placeholder="••••••••"
                  required
                  maxlength="72"
                />
                <p class="field-hint">Use 72 characters or fewer.</p>
              </div>
            </div>

            <div class="modal-section-title">Container Visibility</div>
            <div class="access-level-picker glass mb-6">
              <button
                type="button"
                @click="setAccessLevel('restricted')"
                :class="['al-btn', { active: !isFullAccess }]"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="14"
                  height="14"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
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
                Restricted
              </button>
              <button
                type="button"
                @click="setAccessLevel('full')"
                :class="['al-btn', { active: isFullAccess }]"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="14"
                  height="14"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
                >
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
                </svg>
                Full Access
              </button>
            </div>

            <Transition name="fade">
              <div v-if="!isFullAccess" class="input-field mt-2 mb-6">
                <label>Allowed Containers Pattern</label>
                <input
                  v-model="newUser.allowed_containers"
                  type="text"
                  placeholder="e.g. redis, backend-*"
                />
                <p class="field-hint">
                  Supports wildcards (e.g. redis, backend*) or regex
                </p>
              </div>
            </Transition>

            <div class="modal-section-title">Action Rights</div>
            <div class="modal-perm-grid">
              <label class="custom-checkbox mini"
                ><input type="checkbox" v-model="newUser.canStart" /><span
                  class="checkmark"
                ></span
                ><span>Start</span></label
              >
              <label class="custom-checkbox mini"
                ><input type="checkbox" v-model="newUser.canStop" /><span
                  class="checkmark"
                ></span
                ><span>Stop</span></label
              >
              <label class="custom-checkbox mini"
                ><input type="checkbox" v-model="newUser.canRestart" /><span
                  class="checkmark"
                ></span
                ><span>Restart</span></label
              >
              <label class="custom-checkbox mini"
                ><input type="checkbox" v-model="newUser.canDelete" /><span
                  class="checkmark"
                ></span
                ><span>Delete</span></label
              >
            </div>
            <div v-if="formError" class="form-error">{{ formError }}</div>
            <div class="modal-actions">
              <button
                type="button"
                @click="
                  showCreateModal = false;
                  formError = '';
                "
                class="m-btn secondary"
              >
                Cancel
              </button>
              <button type="submit" class="m-btn primary">
                Create Account
              </button>
            </div>
          </form>
        </div>
      </div>

      <!-- Permissions Modal -->
      <div
        v-if="showPermissionsModal"
        class="modal-overlay"
        @click.self="showPermissionsModal = false"
      >
        <div class="modal-box glass shadow-2xl rules-modal wide-permissions">
          <div class="modal-header">
            <div class="h-top">
              <h3>Security & Access</h3>
              <button
                @click="showPermissionsModal = false"
                class="close-icon-btn"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="20"
                  height="20"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
                >
                  <line x1="18" y1="6" x2="6" y2="18"></line>
                  <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
              </button>
            </div>
            <p>Managing access for @{{ activeUser?.username }}</p>
          </div>

          <div class="merged-modal-scrollable">
            <div v-if="formError" class="form-error mb-4">{{ formError }}</div>
            <div v-if="successMsg" class="form-success mb-4">
              {{ successMsg }}
            </div>

            <div class="modal-section-title">Action Rights</div>
            <div class="modal-perm-list-grid">
              <div
                v-for="p in [
                  'can_start',
                  'can_stop',
                  'can_restart',
                  'can_delete',
                ]"
                :key="p"
                class="perm-row-compact glass"
              >
                <span class="p-name">{{
                  p.replace("can_", "").toUpperCase()
                }}</span>
                <button
                  @click="togglePerm(activeUser, p)"
                  :class="['toggle-pill-mini', { active: activeUser[p] }]"
                >
                  {{ activeUser[p] ? "ON" : "OFF" }}
                </button>
              </div>
            </div>

            <div class="modal-separator"></div>

            <div class="modal-section-title">Container Visibility</div>
            <div class="access-level-picker glass mb-6">
              <button
                type="button"
                @click="setEditAccessLevel('restricted')"
                :class="['al-btn', { active: isRestrictedEdit }]"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="14"
                  height="14"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
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
                Restricted
              </button>
              <button
                type="button"
                @click="setEditAccessLevel('full')"
                :class="['al-btn', { active: !isRestrictedEdit }]"
              >
                <svg
                  viewBox="0 0 24 24"
                  width="14"
                  height="14"
                  stroke="currentColor"
                  stroke-width="2.5"
                  fill="none"
                >
                  <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"></path>
                </svg>
                Full Access
              </button>
            </div>

            <div
              :class="[
                'rule-creation-section glass',
                { 'locked-full': !isRestrictedEdit },
              ]"
            >
              <div class="section-label">Allowed Containers Pattern</div>
              <div class="input-stack">
                <div class="input-group">
                  <label>Allowed Patterns</label>
                  <input
                    v-model="activeUser.allowed_containers"
                    type="text"
                    placeholder="e.g. redis, backend-*"
                  />
                  <p class="field-hint mt-1">
                    Wildcards supported (e.g. redis, backend*)
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div class="modal-footer-merged">
            <button @click="handleDone" class="deploy-btn primary full-width">
              Done
            </button>
          </div>
        </div>
      </div>

      <!-- Password Reset Modal -->
      <div
        v-if="showPasswordModal"
        class="modal-overlay"
        @click.self="showPasswordModal = false"
      >
        <div class="modal-box glass shadow-2xl">
          <div class="modal-header">
            <h3>Reset Account Password</h3>
            <p>
              Define a new credential for
              <strong>@{{ activeUser?.username }}</strong>
            </p>
          </div>
          <div v-if="formError" class="form-error mb-4">{{ formError }}</div>
          <div v-if="successMsg" class="form-success mb-4">
            {{ successMsg }}
          </div>
          <div class="input-field">
            <label>New Secure Password</label>
            <input
              v-model="resetPassword"
              type="password"
              placeholder="••••••••"
              @keyup.enter="updatePassword"
            />
          </div>
          <div class="modal-actions mt-4">
            <button
              @click="updatePassword"
              class="deploy-btn primary full-width"
            >
              Update Password
            </button>
            <button
              @click="showPasswordModal = false"
              class="m-btn secondary full-width mt-2"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from "vue";
import { useRoute, useRouter } from "vue-router";

const route = useRoute();
const router = useRouter();

const currentPage = computed(() => parseInt(route.query.page) || 1);
const totalPages = ref(1);
const totalUsers = ref(0);

const activeTab = ref("staff");
const auditLogs = ref([]);
const auditSearch = ref("");
const loadingLogs = ref(false);
const successMsg = ref("");

const showSuccess = (msg) => {
  successMsg.value = msg;
  setTimeout(() => {
    successMsg.value = "";
  }, 3000);
};

const props = defineProps({
  token: String,
});

const filteredLogs = computed(() => {
  if (!auditSearch.value) return auditLogs.value;
  const q = auditSearch.value.toLowerCase();
  return auditLogs.value.filter(
    (l) =>
      l.username.toLowerCase().includes(q) ||
      l.action.toLowerCase().includes(q) ||
      l.resource.toLowerCase().includes(q) ||
      l.message.toLowerCase().includes(q),
  );
});

const formatAuditTime = (ts) => {
  const d = new Date(ts);
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
};

const fetchAuditLogs = async () => {
  loadingLogs.value = true;
  try {
    const res = await fetch("/api/admin/audit", {
      headers: { Authorization: `Bearer ${props.token}` },
    });
    if (res.ok) auditLogs.value = await res.json();
  } catch (err) {
    console.error(err);
  } finally {
    loadingLogs.value = false;
  }
};

watch(activeTab, (newTab) => {
  if (newTab === "audit" && auditLogs.value.length === 0) {
    fetchAuditLogs();
  }
});

const emit = defineEmits(["close"]);

const users = ref([]);
const activeMenuId = ref(null);
const staffUsers = computed(() => users.value.filter((u) => !u.is_admin));
const activeUser = ref(null);
const showCreateModal = ref(false);
const showPermissionsModal = ref(false);

const newUser = ref({
  username: "",
  password: "",
  isAdmin: false,
  canStart: false,
  canStop: false,
  canRestart: false,
  canDelete: false,
  allowed_containers: ".*",
});
const isFullAccess = ref(true);

const setAccessLevel = (level) => {
  isFullAccess.value = level === "full";
};
const isRestrictedEdit = ref(true);
const showPasswordModal = ref(false);
const resetPassword = ref("");
const formError = ref("");

const toggleMenu = (id) => {
  activeMenuId.value = activeMenuId.value === id ? null : id;
};

const handleDone = async () => {
  const formData = new FormData();
  formData.append(
    "can_start",
    activeUser.value.can_start !== undefined
      ? activeUser.value.can_start
      : activeUser.value.canStart,
  );
  formData.append(
    "can_stop",
    activeUser.value.can_stop !== undefined
      ? activeUser.value.can_stop
      : activeUser.value.canStop,
  );
  formData.append(
    "can_restart",
    activeUser.value.can_restart !== undefined
      ? activeUser.value.can_restart
      : activeUser.value.canRestart,
  );
  formData.append(
    "can_delete",
    activeUser.value.can_delete !== undefined
      ? activeUser.value.can_delete
      : activeUser.value.canDelete,
  );
  formData.append(
    "is_restricted_access",
    activeUser.value.is_restricted_access,
  );
  formData.append(
    "allowed_containers",
    activeUser.value.allowed_containers || ".*",
  );

  const res = await fetch(
    `/api/admin/users/${activeUser.value.id}/permissions`,
    {
      method: "PUT",
      headers: { Authorization: `Bearer ${props.token}` },
      body: formData,
    },
  );
  if (res.ok) {
    showSuccess(`Security policies updated for @${activeUser.value.username}`);
    fetchUsers();
    showPermissionsModal.value = false;
  } else {
    formError.value = "Failed to save permissions.";
  }
};

const openPermissions = (u) => {
  activeUser.value = JSON.parse(JSON.stringify(u)); // Deep clone to avoid immediate list updates
  formError.value = "";
  successMsg.value = "";
  isRestrictedEdit.value = activeUser.value.is_restricted_access;
  showPermissionsModal.value = true;
  activeMenuId.value = null;
};

const setEditAccessLevel = (level) => {
  const restricted = level === "restricted";
  isRestrictedEdit.value = restricted;
  activeUser.value.is_restricted_access = restricted;
};

const openPasswordReset = (u) => {
  activeUser.value = u;
  resetPassword.value = "";
  formError.value = "";
  successMsg.value = "";
  showPasswordModal.value = true;
  activeMenuId.value = null;
};

const updatePassword = async () => {
  if (!resetPassword.value) {
    formError.value = "Password cannot be empty";
    return;
  }
  const formData = new FormData();
  formData.append("password", resetPassword.value);
  const res = await fetch(`/api/admin/users/${activeUser.value.id}/password`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${props.token}` },
    body: formData,
  });
  if (res.ok) {
    showSuccess(`Password updated for @${activeUser.value.username}`);
    setTimeout(() => {
      showPasswordModal.value = false;
    }, 1500);
  } else {
    const data = await res.json().catch(() => ({}));
    formError.value = data.error || "Failed to update password";
  }
};

const fetchUsers = async () => {
  try {
    const res = await fetch(`/api/admin/users?page=${currentPage.value}`, {
      headers: { Authorization: `Bearer ${props.token}` },
    });
    if (res.ok) {
      const data = await res.json();
      users.value = data.users;
      totalUsers.value = data.total;
      totalPages.value = data.pages;
    }
  } catch (err) {
    console.error(err);
  }
};

watch(
  () => route.query.page,
  () => {
    if (activeTab.value === "staff") fetchUsers();
  },
);

const changePage = (p) => {
  router.push({ query: { ...route.query, page: p } });
};

const toggleUserStatus = async (user) => {
  const newStatus = !user.is_active;
  const formData = new FormData();
  formData.append("is_active", newStatus);

  const res = await fetch(`/api/admin/users/${user.id}/active`, {
    method: "PUT",
    headers: { Authorization: `Bearer ${props.token}` },
    body: formData,
  });
  if (res.ok) {
    showSuccess(
      `Account for @${user.username} ${newStatus ? "activated" : "deactivated"}`,
    );
    fetchUsers();
  }
};

const deleteUser = async (id) => {
  if (
    !confirm("Are you sure you want to permanently delete this user account?")
  )
    return;
  const res = await fetch(`/api/admin/users/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${props.token}` },
  });
  if (res.ok) {
    showSuccess("User account terminated");
    fetchUsers();
    activeMenuId.value = null;
  }
};

const createUser = async () => {
  formError.value = "";
  if (!newUser.value.username.trim() || !newUser.value.password.trim()) {
    formError.value = "Username and password are required.";
    return;
  }
  if (newUser.value.password.length > 72) {
    formError.value = "Password must be 72 characters or fewer.";
    return;
  }
  try {
    const formData = new FormData();
    formData.append("username", newUser.value.username);
    formData.append("password", newUser.value.password);
    formData.append("can_start", newUser.value.canStart);
    formData.append("can_stop", newUser.value.canStop);
    formData.append("can_restart", newUser.value.canRestart);
    formData.append("can_delete", newUser.value.canDelete);
    formData.append("is_restricted_access", !isFullAccess.value);
    formData.append(
      "allowed_containers",
      newUser.value.allowed_containers || ".*",
    );

    const res = await fetch("/api/admin/users", {
      method: "POST",
      headers: { Authorization: `Bearer ${props.token}` },
      body: formData,
    });
    if (res.ok) {
      newUser.value = {
        username: "",
        password: "",
        isAdmin: false,
        canStart: false,
        canStop: false,
        canRestart: false,
        canDelete: false,
        allowed_containers: ".*",
      };
      showCreateModal.value = false;
      showSuccess("User created successfully");
      fetchUsers();
    } else {
      const data = await res.json().catch(() => ({}));
      formError.value = data.error || `Failed to create user (${res.status})`;
    }
  } catch (err) {
    formError.value = "Network error. Please try again.";
  }
};

const togglePerm = (user, field) => {
  user[field] = !user[field];
};

onMounted(() => {
  fetchUsers();
  window.addEventListener("click", (e) => {
    if (!e.target.closest(".action-dropdown-wrap")) {
      activeMenuId.value = null;
    }
  });
});
</script>

<style scoped>
.admin-panel {
  height: 100%;
  overflow-y: auto;
  background-color: var(--bg-main);
  color: var(--text-main);
  font-family: var(--font-main);
  padding: 2rem;
  min-width: 0;
}
.panel-container {
  max-width: 1100px;
  margin: 0 auto;
  animation: fadeIn 0.6s cubic-bezier(0.23, 1, 0.32, 1);
  width: 100%;
  min-width: 0;
  min-height: 100%;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  gap: 1rem;
  flex-wrap: wrap;
}
.header-left {
  display: flex;
  align-items: center;
  gap: 1rem;
}
.header-icon {
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, var(--accent), #4f46e5);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  box-shadow: 0 8px 16px -4px rgba(99, 102, 241, 0.4);
}
.header-icon svg {
  width: 18px;
  height: 18px;
}
.title-group h1 {
  font-size: 1.1rem;
  font-weight: 850;
  letter-spacing: -0.02em;
  margin: 0;
  color: var(--text-main);
}
.title-group p {
  color: var(--text-dim);
  font-size: 0.7rem;
  margin: 0.1rem 0 0;
  font-weight: 500;
}

.logo-link {
  text-decoration: none;
  cursor: pointer;
  transition: opacity 0.2s;
}
.logo-link:hover {
  opacity: 0.85;
}
.logo-area {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.logo-box {
  width: 28px;
  height: 28px;
  background: linear-gradient(135deg, var(--accent), #4f46e5);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
}
.logo-box img {
  width: 18px;
  height: 18px;
  object-fit: contain;
  filter: brightness(0) invert(1) !important;
  transition: all 0.4s ease;
}
.brand-name {
  font-size: 1rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  color: var(--text-main);
  margin: 0;
}
.v-divider {
  width: 1px;
  height: 24px;
  background: var(--border);
  margin: 0 0.5rem;
}

.header-actions {
  display: flex;
  gap: 1.25rem;
  align-items: center;
}

.main-user-card {
  border-radius: 20px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  box-shadow: 0 20px 50px -15px var(--shadow);
  min-width: 0;
  flex: 1;
}

.main-user-card.audit-card {
  min-height: calc(100vh - 220px);
}

.main-user-card:not(.audit-card) {
  min-height: calc(100vh - 260px);
}
.refined-table {
  width: 100%;
  border-collapse: collapse;
  min-width: 700px;
}
.refined-table th {
  text-align: left;
  padding: 1.5rem 2rem;
  background: var(--bg-input);
  color: var(--text-mute);
  font-size: 0.62rem;
  font-weight: 900;
  text-transform: uppercase;
  letter-spacing: 0.15em;
  border-bottom: 1px solid var(--border);
}
.refined-table td {
  padding: 1.5rem 2rem;
  border-top: 1px solid var(--border);
  font-size: 0.8rem;
}
.text-right {
  text-align: right;
}

.user-cell {
  display: flex;
  align-items: center;
  gap: 0.85rem;
}
.mini-avatar {
  width: 32px;
  height: 32px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8rem;
  font-weight: 950;
  color: var(--accent);
}
.u-meta {
  display: flex;
  flex-direction: column;
  gap: 0.1rem;
}
.u-name {
  font-weight: 850;
  color: var(--text-main);
  font-size: 0.88rem;
}
.u-id {
  font-size: 0.65rem;
  color: var(--text-mute);
  font-family: var(--font-mono);
  font-weight: 600;
  opacity: 0.7;
}

.badge {
  font-size: 0.6rem;
  font-weight: 950;
  padding: 0.25rem 0.65rem;
  border-radius: 6px;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}
.badge.admin {
  background: rgba(168, 85, 247, 0.1);
  color: #c084fc;
  border: 1px solid rgba(168, 85, 247, 0.2);
}
.badge.staff {
  background: rgba(59, 130, 246, 0.1);
  color: #60a5fa;
  border: 1px solid rgba(59, 130, 246, 0.2);
}

.perm-preview {
  display: flex;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.p-chip {
  font-size: 0.58rem;
  font-weight: 950;
  padding: 0.2rem 0.55rem;
  border-radius: 5px;
  background: var(--bg-input);
  color: var(--text-dim);
  border: 1px solid var(--border);
}
.p-none {
  font-size: 0.7rem;
  color: var(--text-mute);
  font-style: italic;
  font-weight: 600;
}
.all-access {
  font-size: 0.7rem;
  font-weight: 950;
  color: #f59e0b;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.status-pill-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.4rem 0.85rem;
  border-radius: 100px;
  font-size: 0.75rem;
  font-weight: 850;
  cursor: pointer;
  transition: all 0.2s cubic-bezier(0.23, 1, 0.32, 1);
  border: 1px solid var(--border);
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.status-pill-toggle.active {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
  border-color: rgba(16, 185, 129, 0.2);
}

.status-pill-toggle .dot {
  width: 6px;
  height: 6px;
  background: currentColor;
  border-radius: 50%;
  box-shadow: 0 0 10px currentColor;
}

.pagination-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1rem 2rem;
  margin: 1.5rem 0;
  border-radius: 20px;
}

.page-numbers {
  display: flex;
  gap: 0.5rem;
}

.p-btn,
.page-btn {
  padding: 0.6rem 1.2rem;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.p-btn:hover:not(:disabled),
.page-btn:hover {
  background: var(--bg-card);
  border-color: var(--accent);
  transform: translateY(-1px);
}

.page-btn.active {
  background: var(--accent);
  color: #fff;
  border-color: var(--accent);
  box-shadow: 0 8px 16px rgba(99, 102, 241, 0.3);
}

.p-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.dots-btn {
  background: var(--bg-input);
  border: 1px solid var(--border);
  color: var(--text-dim);
  width: 24px;
  height: 24px;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.2s;
  display: flex;
  align-items: center;
  justify-content: center;
}
.dots-btn svg {
  width: 12px;
  height: 12px;
}
.dots-btn:hover,
.dots-btn.active {
  background: var(--bg-card);
  color: var(--text-main);
  border-color: var(--text-mute);
  transform: scale(1.05);
}

.action-dropdown-wrap {
  position: relative;
}
.dropdown-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 12px);
  width: 260px;
  border-radius: 20px;
  border: 1px solid var(--border);
  padding: 0.75rem;
  z-index: 100;
  background: var(--bg-sidebar);
  backdrop-filter: blur(40px);
  box-shadow: 0 20px 50px var(--shadow);
  animation: fadeIn 0.3s cubic-bezier(0.23, 1, 0.32, 1);
}
.menu-item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border: none;
  background: transparent;
  color: var(--text-dim);
  font-size: 0.8rem;
  font-weight: 700;
  cursor: pointer;
  border-radius: 12px;
  transition: all 0.2s;
  text-align: left;
}
.menu-item:hover {
  background: var(--bg-card);
  color: var(--text-main);
  transform: translateX(6px);
}
.menu-item.danger {
  color: #ef4444;
}
.menu-item.danger:hover {
  background: rgba(239, 68, 68, 0.1);
}
.menu-sep {
  height: 1px;
  background: var(--border);
  margin: 0.75rem;
  opacity: 0.5;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(20px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 5000;
  padding: 2rem;
}
[data-theme="dark"] .modal-overlay {
  background: rgba(0, 0, 0, 0.85);
}

.modal-box {
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  border-radius: 28px;
  width: 100%;
  max-width: 800px;
  padding: 3rem 3.5rem;
  position: relative;
  overflow: hidden;
  box-shadow: 0 40px 100px -20px var(--shadow);
}
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1.5rem;
  margin-bottom: 0.5rem;
}
.modal-box::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 4px;
  background: linear-gradient(
    to right,
    transparent,
    var(--accent),
    transparent
  );
}
.modal-box.wide {
  max-width: 600px;
}
.modal-header {
  margin-bottom: 2rem;
}
.modal-header h3 {
  font-size: 1.25rem;
  font-weight: 950;
  margin: 0;
  color: var(--text-main);
  letter-spacing: -0.03em;
}
.modal-header p {
  font-size: 0.85rem;
  color: var(--text-dim);
  margin: 0.4rem 0 0;
  font-weight: 500;
}

.form-error {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  font-size: 0.85rem;
  font-weight: 600;
  border: 1px solid rgba(239, 68, 68, 0.2);
}
.form-success {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  font-size: 0.85rem;
  font-weight: 600;
  border: 1px solid rgba(16, 185, 129, 0.2);
}
.mb-4 {
  margin-bottom: 1rem;
}

.modal-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.input-field {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}
.input-field label {
  font-size: 0.7rem;
  font-weight: 900;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
.input-field input {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 0.85rem 1.15rem;
  color: var(--text-main);
  font-size: 0.88rem;
  font-weight: 600;
  transition: all 0.2s;
}
.input-field input:focus {
  outline: none;
  border-color: var(--accent);
  background: var(--bg-main);
  box-shadow: 0 0 0 3px rgba(99, 102, 241, 0.08);
}

.perm-title {
  font-size: 0.7rem;
  font-weight: 900;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.1em;
  margin-bottom: -0.25rem;
}
.modal-perm-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1rem;
}
.custom-checkbox.mini {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  cursor: pointer;
  user-select: none;
  padding: 0.75rem;
  border-radius: 10px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  transition: all 0.2s;
}
.custom-checkbox.mini:hover {
  background: var(--bg-card);
  border-color: var(--text-mute);
}
.custom-checkbox.mini input {
  display: none;
}
.checkmark {
  width: 18px;
  height: 18px;
  background: var(--bg-main);
  border: 2px solid var(--border);
  border-radius: 6px;
  position: relative;
  transition: all 0.2s;
}
.custom-checkbox.mini input:checked ~ .checkmark {
  background: var(--accent);
  border-color: var(--accent);
  box-shadow: 0 3px 8px rgba(99, 102, 241, 0.3);
}
.custom-checkbox.mini input:checked ~ .checkmark::after {
  content: "✓";
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 10px;
  font-weight: 900;
}
.custom-checkbox.mini span {
  font-size: 0.85rem;
  color: var(--text-main);
  font-weight: 750;
}

.modal-actions {
  display: flex;
  gap: 1rem;
  margin-top: 1rem;
}
.m-btn {
  flex: 1;
  padding: 0.85rem;
  border-radius: 12px;
  font-weight: 800;
  font-size: 0.85rem;
  cursor: pointer;
  border: none;
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
}
.m-btn.primary {
  background: var(--accent);
  color: #fff;
  box-shadow: 0 10px 20px -5px rgba(99, 102, 241, 0.4);
}
.m-btn.primary:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
}
.m-btn.secondary {
  background: var(--bg-input);
  color: var(--text-main);
  border: 1px solid var(--border);
}
.m-btn.secondary:hover {
  background: var(--bg-card);
  transform: translateY(-1px);
}

.modal-perm-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  margin-bottom: 2.5rem;
}
.perm-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-radius: 20px;
  background: var(--bg-input);
  border: 1px solid var(--border);
}
.perm-name {
  font-weight: 900;
  font-size: 0.8rem;
  letter-spacing: 0.15em;
  color: var(--text-mute);
}
.toggle-pill {
  padding: 0.65rem 1.5rem;
  border-radius: 100px;
  font-size: 0.75rem;
  font-weight: 900;
  border: none;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
  min-width: 120px;
}
.toggle-pill.active {
  background: #10b981;
  color: #fff;
  box-shadow: 0 10px 20px -5px rgba(16, 185, 129, 0.4);
}
.toggle-pill:not(.active) {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.1);
}

.rules-editor {
  display: flex;
  flex-direction: column;
  gap: 2.5rem;
}
.rule-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem 2rem;
  border-radius: 20px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  transition: all 0.2s;
}
.rule-bar:hover {
  border-color: var(--text-mute);
  transform: translateX(8px);
}
.r-main {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.r-label {
  font-weight: 900;
  font-size: 1.05rem;
  color: var(--text-main);
  letter-spacing: -0.02em;
}
.r-code {
  font-size: 0.85rem;
  font-family: var(--font-mono);
  color: var(--accent);
  font-weight: 600;
  opacity: 0.8;
}
.r-del {
  background: transparent;
  border: none;
  color: var(--text-mute);
  cursor: pointer;
  transition: 0.2s;
  padding: 0.5rem;
}
.r-del:hover {
  color: #ef4444;
  transform: scale(1.2);
}

.new-rule-area {
  padding: 2.5rem;
  border-radius: 32px;
  display: flex;
  flex-direction: column;
  gap: 2rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  padding: 0.75rem 1.25rem;
  border-radius: 12px;
  font-weight: 800;
  font-size: 0.85rem;
  cursor: pointer;
  border: none;
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
}
.action-btn.primary {
  background: var(--accent);
  color: #fff;
  box-shadow: 0 8px 16px -4px rgba(99, 102, 241, 0.4);
}
.action-btn.primary svg {
  width: 14px;
  height: 14px;
}
.action-btn.secondary {
  background: rgba(99, 102, 241, 0.1);
  color: var(--accent);
  border: 1px solid rgba(99, 102, 241, 0.2);
}
.action-btn:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
}
.full-width {
  width: 100%;
  justify-content: center;
}
.mt-4 {
  margin-top: 1rem;
}

.back-btn {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-mute);
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
}
.back-btn svg {
  width: 16px;
  height: 16px;
}
.back-btn:hover {
  background: var(--bg-card);
  color: var(--text-main);
  border-color: var(--text-mute);
  transform: scale(1.05);
}

.header-center {
  flex: 1;
  display: flex;
  justify-content: center;
}
.tab-switcher {
  display: flex;
  padding: 0.3rem;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
}
.tab-btn {
  padding: 0.4rem 1.15rem;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
  border: none;
  background: transparent;
  color: var(--text-mute);
  transition: all 0.2s;
}
.tab-btn.active {
  background: var(--bg-card);
  color: var(--accent);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}
.tab-btn:hover:not(.active) {
  color: var(--text-main);
}

.audit-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem;
  height: calc(100vh - 220px);
}
.audit-toolbar {
  display: flex;
  gap: 1rem;
  align-items: center;
  flex-wrap: wrap;
}
.search-box {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 1rem;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  min-width: 0;
}
.search-box input {
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 0.85rem;
  width: 100%;
  outline: none;
}
.refresh-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 10px;
  cursor: pointer;
  border: 1px solid var(--border);
  background: var(--bg-input);
  color: var(--text-mute);
}
.refresh-btn:hover {
  color: var(--accent);
  border-color: var(--accent);
}

.audit-table-wrap {
  flex: 1;
  overflow-y: auto;
  border-radius: 12px;
}
.audit-table {
  border-collapse: separate;
  border-spacing: 0;
  width: 100%;
}
.audit-table th {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--bg-card);
  border-bottom: 2px solid var(--border);
}

.time-col {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--text-mute);
  white-space: nowrap;
  width: 160px;
}
.mini-avatar {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: var(--accent);
  color: #fff;
  font-size: 0.65rem;
  font-weight: 900;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 0.75rem;
}
.action-tag {
  font-size: 0.65rem;
  font-weight: 900;
  padding: 0.2rem 0.5rem;
  border-radius: 4px;
  background: rgba(99, 102, 241, 0.1);
  color: var(--accent);
  letter-spacing: 0.05em;
}
.status-pill {
  font-size: 0.65rem;
  font-weight: 900;
  padding: 0.25rem 0.6rem;
  border-radius: 6px;
}
.status-pill.success {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}
.status-pill.forbidden,
.status-pill.failed {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}
.res-col {
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: 0.75rem;
}
.msg-col {
  font-size: 0.8rem;
  color: var(--text-dim);
}

.rotating {
  animation: spin 1s linear infinite;
}
@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.empty-state {
  text-align: center;
  padding: 3rem;
  color: var(--text-mute);
  font-size: 0.85rem;
  font-weight: 500;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(15px) scale(0.98);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* Transitions */
.slide-fade-enter-active {
  transition: all 0.3s cubic-bezier(0.23, 1, 0.32, 1);
}
.slide-fade-leave-active {
  transition: all 0.2s cubic-bezier(0.23, 1, 0.32, 1);
}
.slide-fade-enter-from,
.slide-fade-leave-to {
  transform: translateY(10px);
  opacity: 0;
}
/* Shared Rules */
.shared-rules-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 1.25rem;
  padding: 1.5rem;
}
.global-rule-card {
  padding: 1.25rem;
  border-radius: 18px;
  border: 1px solid var(--border);
  background: var(--bg-sidebar);
  transition: all 0.3s;
}
.global-rule-card:hover {
  transform: translateY(-4px);
  border-color: var(--accent);
  box-shadow: 0 10px 30px -10px var(--shadow);
}
.gr-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}
.gr-info {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}
.gr-name {
  font-size: 0.95rem;
  font-weight: 850;
  color: var(--text-main);
}
.gr-badge {
  font-size: 0.65rem;
  font-weight: 900;
  color: var(--accent);
  background: rgba(99, 102, 241, 0.1);
  padding: 0.15rem 0.5rem;
  border-radius: 4px;
  width: fit-content;
}
.r-del-btn {
  border: none;
  background: transparent;
  color: var(--text-mute);
  cursor: pointer;
  transition: all 0.2s;
  padding: 4px;
}
.r-del-btn:hover {
  color: #ef4444;
  transform: scale(1.1);
}
.gr-body .r-code {
  display: block;
  padding: 0.75rem;
  background: var(--bg-input);
  border-radius: 10px;
  font-family: var(--font-mono);
  font-size: 0.8rem;
  color: var(--accent);
  overflow-x: auto;
}
.empty-rules-state {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 5rem 2rem;
  text-align: center;
  color: var(--text-mute);
}
.empty-rules-state p {
  font-size: 1.1rem;
  font-weight: 800;
  color: var(--text-dim);
  margin-bottom: 0.5rem;
}
.empty-rules-state span {
  font-size: 0.85rem;
}
/* Rules Modal Improvements */
.rules-stack {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  margin-bottom: 2rem;
}
.rule-entry {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 1.25rem;
  border-radius: 16px;
  border: 1px solid var(--border);
  background: var(--bg-sidebar);
  transition: all 0.2s;
}
.rule-entry:hover {
  transform: translateX(5px);
  border-color: var(--accent);
}
.re-info {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}
.re-label {
  font-weight: 850;
  font-size: 0.9rem;
  color: var(--text-main);
}
.re-pattern {
  font-family: var(--font-mono);
  font-size: 0.75rem;
  color: var(--accent);
  opacity: 0.9;
}
.re-delete {
  background: transparent;
  border: none;
  color: var(--text-mute);
  cursor: pointer;
  transition: 0.2s;
  padding: 0.5rem;
}
.re-delete:hover {
  color: #ef4444;
  transform: scale(1.15);
}

.empty-rules-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 1rem;
  border-radius: 20px;
  border: 1px dashed var(--border);
  color: var(--text-mute);
  margin-bottom: 2rem;
  text-align: center;
}
.empty-rules-box p {
  font-weight: 800;
  color: var(--text-dim);
  margin: 0.75rem 0 0.25rem;
}
.empty-rules-box span {
  font-size: 0.75rem;
}

.new-rule-composer {
  padding: 1.5rem;
  border-radius: 24px;
  background: var(--bg-input);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
.composer-header {
  font-size: 0.8rem;
  font-weight: 900;
  color: var(--text-dim);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.composer-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 1rem;
}
.deploy-btn {
  background: var(--accent);
  color: #fff;
  padding: 0.85rem;
  border-radius: 12px;
  font-weight: 800;
  border: none;
  cursor: pointer;
  transition: all 0.3s;
  box-shadow: 0 8px 20px -6px rgba(99, 102, 241, 0.4);
}
.deploy-btn:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
  box-shadow: 0 12px 24px -6px rgba(99, 102, 241, 0.5);
}
/* Rules Modal Redesign */
.rules-modal {
  max-width: 480px !important;
  padding: 2rem !important;
}
.h-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.close-icon-btn {
  background: transparent;
  border: none;
  color: var(--text-mute);
  cursor: pointer;
  padding: 4px;
  border-radius: 8px;
  transition: all 0.2s;
}
.close-icon-btn:hover {
  background: var(--bg-input);
  color: var(--text-main);
}

.rules-view-container {
  margin-bottom: 2rem;
}
.rules-scroll-area {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-height: 220px;
  overflow-y: auto;
  padding-right: 4px;
}
.rules-scroll-area::-webkit-scrollbar {
  width: 4px;
}
.rules-scroll-area::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 10px;
}

.rule-card-mini {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.85rem 1rem;
  border-radius: 14px;
  border: 1px solid var(--border);
  background: var(--bg-input);
  transition: all 0.2s;
}
.rule-card-mini:hover {
  border-color: var(--accent);
  transform: translateX(4px);
}
.rc-left {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}
.rc-name {
  font-size: 0.85rem;
  font-weight: 850;
  color: var(--text-main);
}
.rc-pattern {
  font-family: var(--font-mono);
  font-size: 0.7rem;
  color: var(--accent);
  opacity: 0.8;
}
.rc-del {
  background: transparent;
  border: none;
  color: var(--text-mute);
  cursor: pointer;
  transition: 0.2s;
}
.rc-del:hover {
  color: #ef4444;
  transform: scale(1.1);
}

.empty-rules-state-compact {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 2.5rem 1.5rem;
  border-radius: 18px;
  border: 1px dashed var(--border);
  text-align: center;
  color: var(--text-mute);
  background: rgba(255, 255, 255, 0.02);
}
.esc-icon {
  margin-bottom: 0.75rem;
  opacity: 0.6;
  color: var(--accent);
}
.empty-rules-state-compact p {
  font-size: 0.9rem;
  font-weight: 800;
  color: var(--text-dim);
  margin: 0;
}
.empty-rules-state-compact span {
  font-size: 0.7rem;
  margin-top: 0.25rem;
}

.rule-creation-section {
  padding: 1.25rem;
  border-radius: 20px;
  background: var(--bg-sidebar);
  border: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.section-label {
  font-size: 0.65rem;
  font-weight: 950;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
.input-stack {
  display: flex;
  flex-direction: column;
  gap: 0.85rem;
}
.input-group {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
.input-group label {
  font-size: 0.65rem;
  font-weight: 850;
  color: var(--text-dim);
}
.input-group input {
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 10px;
  padding: 0.65rem 0.85rem;
  color: var(--text-main);
  font-size: 0.85rem;
  font-weight: 600;
}
.input-group input:focus {
  border-color: var(--accent);
  outline: none;
}

.action-btn-primary {
  background: var(--accent);
  color: #fff;
  padding: 0.75rem;
  border-radius: 10px;
  font-weight: 800;
  font-size: 0.85rem;
  border: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  transition: all 0.3s;
  box-shadow: 0 4px 12px rgba(99, 102, 241, 0.3);
}
.action-btn-primary:hover {
  transform: translateY(-2px);
  filter: brightness(1.1);
}

.modal-footer-minimal {
  margin-top: 1.5rem;
  display: flex;
  justify-content: center;
}
.footer-close-btn {
  background: transparent;
  border: none;
  color: var(--text-mute);
  font-size: 0.75rem;
  font-weight: 800;
  cursor: pointer;
  transition: all 0.2s;
  padding: 0.5rem 1rem;
}
.footer-close-btn:hover {
  color: var(--text-main);
  text-decoration: underline;
}

.access-level-picker {
  display: flex;
  background: var(--bg-input);
  padding: 0.35rem;
  border-radius: 12px;
  border: 1px solid var(--border);
  margin-bottom: 0.5rem;
}
.al-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.65rem;
  border: none;
  background: transparent;
  color: var(--text-dim);
  font-size: 0.75rem;
  font-weight: 850;
  border-radius: 9px;
  cursor: pointer;
  transition: all 0.2s;
}
.al-btn.active {
  background: var(--bg-sidebar);
  color: var(--accent);
  box-shadow: 0 4px 12px var(--shadow);
  border: 1px solid var(--border);
}

.locked-full {
  opacity: 0.6;
  pointer-events: none;
  filter: grayscale(0.5);
}

/* Merged Modal Styles */
.wide-permissions {
  max-width: 520px !important;
}
.merged-modal-scrollable {
  max-height: 70vh;
  overflow-y: auto;
  padding-right: 8px;
  margin-bottom: 1rem;
}
.merged-modal-scrollable::-webkit-scrollbar {
  width: 4px;
}
.merged-modal-scrollable::-webkit-scrollbar-thumb {
  background: var(--border);
  border-radius: 10px;
}

.modal-section-title {
  font-size: 0.65rem;
  font-weight: 950;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.15em;
  margin-bottom: 1rem;
}
.modal-perm-list-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.75rem;
  margin-bottom: 2rem;
}
.perm-row-compact {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0.75rem 1rem;
  border-radius: 12px;
  background: var(--bg-input);
  border: 1px solid var(--border);
}
.p-name {
  font-weight: 800;
  font-size: 0.75rem;
  color: var(--text-dim);
}
.toggle-pill-mini {
  padding: 0.35rem 0.85rem;
  border-radius: 8px;
  font-size: 0.65rem;
  font-weight: 900;
  border: none;
  cursor: pointer;
  transition: all 0.2s;
  min-width: 50px;
}
.toggle-pill-mini.active {
  background: #10b981;
  color: #fff;
}
.toggle-pill-mini:not(.active) {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.modal-separator {
  height: 1px;
  background: var(--border);
  margin: 2rem 0;
  opacity: 0.5;
}
.rules-view-container.compact {
  margin-bottom: 1.5rem;
}
.modal-footer-merged {
  border-top: 1px solid var(--border);
  padding-top: 1.5rem;
  margin-top: 1rem;
}
.mb-6 {
  margin-bottom: 2.5rem;
}

@media (max-width: 600px) {
  .modal-box {
    padding: 2rem 1.5rem;
    border-radius: 20px;
    max-height: 95vh;
    overflow-y: auto;
    width: 92% !important;
  }
  .modal-perm-grid {
    grid-template-columns: 1fr;
  }
  .modal-actions {
    flex-direction: column-reverse;
  }
  .access-level-picker {
    flex-direction: column;
    gap: 0.5rem;
  }
  .al-btn {
    padding: 0.5rem;
  }
  .modal-header h3 {
    font-size: 1.15rem;
  }
}

@media (max-width: 1024px) {
  .modal-perm-list-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .panel-header {
    padding: 1.25rem;
    flex-direction: column;
    align-items: flex-start;
    gap: 1rem;
  }
  .header-left,
  .header-center,
  .header-actions {
    width: 100%;
  }
  .header-center {
    justify-content: flex-start;
  }
  .tab-switcher {
    width: 100%;
  }
  .tab-btn {
    flex: 1;
  }
  .header-actions {
    width: 100%;
    justify-content: flex-end;
  }
  .action-btn {
    width: 100%;
    justify-content: center;
  }
  .audit-table-wrap {
    overflow-x: auto;
    margin: 0 -1rem;
    padding: 0 1rem;
  }
  .audit-table {
    min-width: 800px;
  }
  .admin-panel {
    padding: 1rem 0.75rem;
  }
  .panel-container {
    max-width: 100%;
  }
  .main-user-card,
  .audit-card {
    border-radius: 16px;
  }
  .audit-card {
    height: auto;
    min-height: 0;
  }

  .refined-table.desktop-only {
    display: none !important;
  }
  .audit-table-wrap.desktop-only {
    display: none !important;
  }

  /* Mobile Card Styles */
  .mobile-card-list,
  .mobile-audit-list {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding: 0.75rem 0;
  }
  .mobile-card {
    border-radius: 16px;
    border: 1px solid var(--border);
    overflow: hidden;
    background: var(--bg-card);
  }
  .card-header {
    padding: 1rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid var(--border);
    background: var(--bg-input);
  }
  .card-body {
    padding: 1rem;
  }
  .card-info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
  }
  .card-info-row:not(:last-child) {
    border-bottom: 1px dotted var(--border);
  }
  .info-label {
    font-size: 0.65rem;
    font-weight: 800;
    color: var(--text-mute);
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .audit-item .card-header {
    background: transparent;
  }
  .audit-main-action {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.75rem;
    flex-wrap: wrap;
  }
  .res-target {
    font-size: 0.75rem;
    font-weight: 700;
    color: var(--text-dim);
  }
  .audit-msg {
    font-size: 0.78rem;
    color: var(--text-dim);
    line-height: 1.4;
    margin: 0;
  }
  .mobile-dropdown {
    right: 10px;
    top: 40px;
  }
}
@media (min-width: 769px) {
  .mobile-only {
    display: none !important;
  }
}

@media (max-width: 600px) {
  .panel-header {
    padding: 0.75rem;
  }
  .tab-switcher {
    gap: 0.25rem;
    padding: 0.25rem;
  }
  .tab-btn {
    padding: 0.45rem 0.75rem;
    font-size: 0.7rem;
  }
  .header-actions {
    flex-direction: column;
    align-items: stretch;
  }
  .action-btn {
    min-height: 42px;
  }
  .pagination-bar {
    flex-direction: column;
    gap: 0.75rem;
    padding: 1rem;
  }
  .page-numbers {
    width: 100%;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }
  .page-btn {
    flex: 0 0 auto;
  }
  .modal-overlay {
    padding: 1rem 0.75rem;
  }
  .modal-box {
    width: 100% !important;
  }
}
</style>
