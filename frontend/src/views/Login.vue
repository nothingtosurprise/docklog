<template>
  <div class="login-overlay">
    <div class="login-card glass">
      <div class="login-header">
        <img src="/logo-vertical.png?v=2" alt="DockLog" class="logo-hero" />
        <h1>DockLog</h1>
        <p>Resource Management & Monitoring</p>
      </div>
      <form @submit.prevent="handleLogin">
        <div class="input-group">
          <label>Username</label>
          <input
            v-model="username"
            type="text"
            placeholder="e.g. admin"
            required
          />
        </div>
        <div class="input-group">
          <label>Password</label>
          <input
            v-model="password"
            type="password"
            placeholder="••••••••"
            required
          />
        </div>
        <button type="submit" :disabled="loading" class="login-btn">
          {{ loading ? "Authenticating..." : "Access Dashboard" }}
        </button>
        <p v-if="error" class="error-msg">{{ error }}</p>
      </form>
      <div class="login-footer">Premium Container Monitoring System</div>
    </div>
  </div>
</template>

<script setup>
import { ref } from "vue";
import { useRouter } from "vue-router";
import { secureStorage } from "../utils/storage";
import { sharedState } from "../utils/sharedState";

const router = useRouter();
const username = ref("");
const password = ref("");
const loading = ref(false);
const error = ref("");

const handleLogin = async () => {
  loading.value = true;
  error.value = "";

  try {
    const formData = new FormData();
    formData.append("username", username.value);
    formData.append("password", password.value);

    const res = await fetch("/api/token", {
      method: "POST",
      body: formData,
    });

    if (res.ok) {
      const data = await res.json();
      secureStorage.setItem("token", data.access_token);
      sharedState.forcePasswordChange = data.password_changed === false;
      sharedState.showPasswordModal = data.password_changed === false;
      router.push("/dashboard");
    } else {
      error.value = "Invalid credentials. Please try again.";
    }
  } catch (err) {
    error.value = "System connection failed";
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.login-overlay {
  position: fixed;
  inset: 0;
  background: var(--bg-main);
  background-image:
    radial-gradient(at 0% 0%, rgba(99, 102, 241, 0.1) 0px, transparent 50%),
    radial-gradient(at 100% 100%, rgba(16, 185, 129, 0.1) 0px, transparent 50%);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  overflow-y: auto;
  padding: 2rem 1rem;
}

.login-card {
  width: 100%;
  max-width: 520px;
  padding: 4rem;
  border-radius: 32px;
  border: 1px solid var(--border);
  background: var(--glass-bg);
  box-shadow: 0 40px 100px var(--shadow);
}

.login-header {
  text-align: center;
  margin-bottom: 3rem;
}

.logo-hero {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  margin: 0 auto 1.5rem;
  display: block;
  object-fit: cover;
  box-shadow: 0 10px 20px rgba(59, 130, 246, 0.2);
  filter: var(--logo-filter);
  transition: filter 0.4s ease;
}

.login-header h1 {
  font-size: 2.5rem;
  font-weight: 900;
  letter-spacing: -0.05em;
  margin: 0;
  color: var(--text-main);
}

.login-header p {
  color: var(--text-dim);
  font-size: 0.95rem;
  margin-top: 0.5rem;
  font-weight: 600;
}

.input-group {
  margin-bottom: 1.5rem;
}
.input-group label {
  display: block;
  font-size: 0.7rem;
  font-weight: 800;
  color: var(--text-mute);
  margin-bottom: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.1em;
}
.input-group input {
  width: 100%;
  padding: 1rem 1.25rem;
  background: var(--bg-input);
  border: 1px solid var(--border);
  border-radius: 14px;
  color: var(--text-main);
  font-size: 1rem;
  transition: all 0.2s;
}
.input-group input:focus {
  outline: none;
  border-color: var(--accent);
  background: var(--bg-card);
  box-shadow: 0 0 0 4px rgba(59, 130, 246, 0.1);
}

.login-btn {
  width: 100%;
  padding: 1rem;
  background: var(--accent);
  color: white;
  border: none;
  border-radius: 14px;
  font-size: 1rem;
  font-weight: 800;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  margin-top: 1.5rem;
}
.login-btn:hover:not(:disabled) {
  background: #4f46e5;
  transform: translateY(-2px);
  box-shadow: 0 10px 20px rgba(59, 130, 246, 0.4);
}
.login-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.error-msg {
  color: #ef4444;
  font-size: 0.85rem;
  text-align: center;
  margin-top: 1.5rem;
  font-weight: 600;
}
.login-footer {
  margin-top: 3rem;
  text-align: center;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--text-mute);
  text-transform: uppercase;
  letter-spacing: 0.15em;
}
@media (max-width: 480px) {
  .login-card {
    padding: 2.5rem 1.5rem !important;
    border-radius: 20px;
  }
  .login-header h1 {
    font-size: 2rem;
  }
  .login-header {
    margin-bottom: 2rem;
  }
}
</style>
