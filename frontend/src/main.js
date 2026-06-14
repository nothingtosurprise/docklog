import { createApp } from 'vue';
import './style.css';
import App from './App.vue';
import router from './router';
import { sharedState } from './utils/sharedState';
import { secureStorage } from './utils/storage';
import { getRequestUrl, shouldForceLogout } from './utils/authSession';

const forceLogout = () => {
  secureStorage.removeItem('token');
  secureStorage.removeItem('user');
  sharedState.currentUser = null;
  sharedState.showPasswordModal = false;
  sharedState.forcePasswordChange = false;

  if (router.currentRoute.value.path !== '/login') {
    router.replace('/login').catch(() => {});
  }
};

if (!window.__docklogFetchPatched) {
  const originalFetch = window.fetch.bind(window);

  window.fetch = async (...args) => {
    const response = await originalFetch(...args);

    if (await shouldForceLogout(response, getRequestUrl(args[0]))) {
      forceLogout();
    }

    return response;
  };

  window.__docklogFetchPatched = true;
}

const app = createApp(App);
app.use(router);
app.mount('#app');
